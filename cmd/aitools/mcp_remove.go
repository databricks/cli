package aitools

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/aitools/mcp/agentconfig"
	"github.com/databricks/cli/libs/aitools/mcpconfig"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
)

// removeFlagsHint is the error for a run that names no servers. Tests check it too.
const removeFlagsHint = "pass --names, --profile or --all so the command runs without interactive prompts"

type mcpRemoveFlags struct {
	names  string
	agents string
	all    bool
}

// removeOutput is the `mcp remove --output json` document.
type removeOutput struct {
	Removed       []removeRemoved             `json:"removed"`
	Skipped       []removeSkipped             `json:"skipped"`
	Error         string                      `json:"error,omitempty"`
	ErrorCategory protos.AitoolsErrorCategory `json:"error_category,omitempty"`

	// results is one telemetry result per (service, agent) attempted. It is not in the JSON.
	results []protos.AitoolsMcpServerResult
	// stateBefore is the state as it was read before the removal. It is not in the JSON.
	stateBefore *mcp.State
	// profile is the profile and config file that the selected servers are bound to, when they all are bound to one. It is
	// the zero key otherwise. It is not in the JSON.
	profile profileKey
}

type removeRemoved struct {
	Name    string   `json:"name"`
	Service string   `json:"service"`
	Profile string   `json:"profile"`
	Agents  []string `json:"agents"`
	// RecordOnly names the agents whose record was dropped without an edit, because this CLI cannot edit their config.
	RecordOnly []string `json:"record_only,omitempty"`
}

// removeSkipped is one server that was left in one agent: the agent is named, since a server can be removed from
// some agents and skipped in others.
type removeSkipped struct {
	Name   string `json:"name"`
	Agent  string `json:"agent"`
	Reason string `json:"reason"`
}

// removeOutcome is what happened to one server in one agent config.
type removeOutcome int

const (
	// removeDeleted means the entry was removed from the agent config.
	removeDeleted removeOutcome = iota
	// removePruned means no entry this CLI wrote is in the agent config, so only the state record goes.
	removePruned
	// removeKept means a Databricks entry that this CLI did not write is left in place, so the server is kept.
	removeKept
)

// saveMCPState writes the MCP state. It is a variable so that a test can make the save fail.
var saveMCPState = mcp.SaveState

// recordDrop names one agent record to delete from state: the server's name and the agent.
type recordDrop struct {
	name  string
	agent string
}

// NewMcpRemoveCmd returns `aitools mcp remove`. The edit needs no network; the telemetry upload after it is best-effort.
func NewMcpRemoveCmd() *cobra.Command {
	var f mcpRemoveFlags
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove Databricks managed MCP servers from coding agents",
		Long: `Remove Databricks managed MCP servers that this CLI installed.

Choose servers with --names (server names as listed by 'aitools mcp list', or service names; the
last part of a service name, such as github, is enough when only one schema has it), --profile
(every server of one profile), or --all. --profile also narrows --names or --all to one profile, and
--agents narrows any of them to some agents.
In a terminal, with none of --names, --profile or --all, the command shows the installed servers to
choose from. Without a terminal, one of them is required.
With --profile or --all and nothing installed to remove, the command succeeds and removes nothing.

Only entries this CLI wrote are removed. An entry that this CLI did not write, and that is not a Databricks
entry, is left in place, and the install record for its agent is dropped. A Databricks entry that was changed outside
this CLI is left in place and reported, and a --names run that names its server is refused. When an agent's config
moved, the entry is removed from the file it was written to and from the file the agent reads now.
Entries that run 'aitools mcp serve' without an install record are not chosen by any flag;
'aitools mcp list' shows them as unmanaged. Your sign-ins to services are kept; 'aitools mcp logout'
removes one.`,
		Example: `  databricks aitools mcp remove --names github-logfood
  databricks aitools mcp remove --profile logfood --agents cursor
  databricks aitools mcp remove --all`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			jsonMode := installOutputIsJSON(cmd)

			out := &removeOutput{Removed: []removeRemoved{}, Skipped: []removeSkipped{}}
			runErr := runMcpRemove(ctx, cmd, f, jsonMode, out)
			err := runErr
			if jsonMode {
				err = writeRemoveJSON(cmd, out, runErr)
			}
			logRemoveEvent(ctx, out)
			return err
		},
	}

	cmd.Flags().StringVar(&f.names, "names", "", "Servers to remove: server names or service names, comma-separated")
	cmd.Flags().StringVar(&f.agents, "agents", "", "Agents to remove from, comma-separated (default: every agent the server is installed for)")
	cmd.Flags().BoolVar(&f.all, "all", false, "Remove every server this CLI installed")
	return cmd
}

// writeRemoveJSON writes the JSON document of a run that ended with runErr and returns the command's error.
func writeRemoveJSON(cmd *cobra.Command, out *removeOutput, runErr error) error {
	// The document carries a top-level failure; per-agent failures are already in skipped.
	if top := topLevelFailure(runErr); top != nil {
		out.Error = top.Error()
		out.ErrorCategory = errorCategory(top)
	}
	if err := renderJSON(cmd.OutOrStdout(), out); err != nil {
		return errors.Join(runErr, err)
	}
	if runErr != nil {
		return root.ErrAlreadyPrinted
	}
	return nil
}

// logRemoveEvent buffers the REMOVE event and sets the config its exit-time upload authenticates with. It runs after the
// output is written. A run that logged no event touched no server, so it has nothing to upload and sets no config.
func logRemoveEvent(ctx context.Context, out *removeOutput) {
	if !logMcpEvent(ctx, protos.AitoolsMcpEvent{
		CallingAgent: root.CallingAgentType(),
		Operation:    protos.AitoolsMcpOperationRemove,
		Servers:      out.results,
	}) {
		return
	}
	if out.profile.profile == "" {
		// The servers are bound to several profiles, so no one profile's token may stand in for the upload.
		telemetry.SkipUpload(ctx)
		return
	}
	// The upload sends the profile's token to the profile's host, so a profile whose recorded servers no longer match it
	// uploads nothing. The records are the ones read before the removal.
	targets := recordedUploadTargets(ctx, out.stateBefore, out.profile)
	if len(targets) == 0 {
		log.Debugf(ctx, "telemetry: profile %s no longer matches an MCP server recorded for it; not uploading telemetry", out.profile.profile)
		telemetry.SkipUpload(ctx)
		return
	}
	setUploadConfig(ctx, out.profile.profile, out.profile.path, nil, "", targets...)
}

// runMcpRemove removes the chosen servers from the agent configs and records the result. Results are printed
// before a state failure is returned. An agent failure fails the command when --agents named the agent, or when a
// selected server was removed from no agent; otherwise it is reported in skipped.
func runMcpRemove(ctx context.Context, cmd *cobra.Command, f mcpRemoveFlags, jsonMode bool, out *removeOutput) error {
	names := splitAndTrim(f.names)
	agentNames := splitAndTrim(f.agents)
	profileName := profileFlag(cmd)
	// A value that names nothing, such as ",", must not widen the removal to every server or agent.
	if cmd.Flags().Changed("names") && len(names) == 0 {
		return failure(protos.AitoolsErrorCategoryInvalidFlags, "--names names no server; pass one or more server names")
	}
	if cmd.Flags().Changed("agents") && len(agentNames) == 0 {
		return failure(protos.AitoolsErrorCategoryInvalidFlags, "--agents names no agent; pass one or more of: %s", strings.Join(mcpAgentNames(), ", "))
	}
	interactive := !jsonMode && promptsSupported(ctx)
	if err := validateMCPRemoveFlags(f.all, names, profileName, interactive); err != nil {
		return err
	}

	// A nil targets means every agent a server is installed for.
	var targets []*agents.Agent
	if len(agentNames) > 0 {
		var err error
		targets, err = resolveMCPAgents(agentNames)
		if err != nil {
			return err
		}
	}

	state, err := mcp.LoadState(ctx)
	if err != nil {
		return err
	}
	out.stateBefore = state
	var selected []mcp.Entry
	if interactive && !f.all && len(names) == 0 && profileName == "" {
		// An agent with no installed server is refused as it is without a terminal, before the picker offers nothing for it.
		if err := checkTargetsInstalled(state.Servers, targets); err != nil {
			return err
		}
		var offered bool
		selected, offered, err = chooseRemovals(state, targets)
		if err != nil {
			return err
		}
		if offered && len(selected) == 0 {
			cmdio.LogString(ctx, "No servers selected; nothing removed.")
			return nil
		}
		if offered {
			// The picker chose the servers, so the removal is confirmed, as add confirms its plan and uninstall its removal.
			printRemovePlan(ctx, selected, targets)
			proceed, err := promptMCPProceed()
			if err != nil {
				return err
			}
			if !proceed {
				cmdio.LogString(ctx, cancelledMessage)
				return nil
			}
		}
	} else {
		selected, err = selectInstalls(state, names, profileName)
		if err != nil {
			return withAdoptHint(ctx, err, state)
		}
		if err := checkTargetsInstalled(selected, targets); err != nil {
			return err
		}
	}
	if len(names) > 0 {
		if err := checkNamedInstalled(selected, targets); err != nil {
			return err
		}
	}
	if targets != nil {
		// --agents chooses the servers installed for those agents, so a server with no agent left is not chosen.
		selected = slices.DeleteFunc(selected, func(e mcp.Entry) bool { return len(e.Agents) == 0 })
	}
	out.profile = sharedProfile(selected, mcp.CheckEntries(ctx, &mcp.State{Servers: selected}))

	// Each agent config file is read once per command, and each server's config file is resolved once.
	files := newAgentFiles()
	// A named server that is kept, or whose records are all for unknown agents, refuses the whole run before anything is
	// written, as a name that matches nothing does.
	if len(names) > 0 {
		if refused := refusedByName(ctx, files, selected, targets); len(refused) > 0 {
			out.Skipped = append(out.Skipped, refused...)
			if !jsonMode {
				printRemoveResults(ctx, out)
			}
			return failure(protos.AitoolsErrorCategoryMCPNameConflict, "nothing was removed: named MCP servers are left in place: %s", refusedNames(refused))
		}
	}

	var drops []recordDrop
	var agentErrs []error
	// recordOnly are the indexes of out.results whose removal only drops a state record.
	var recordOnly []int
	for _, e := range selected {
		removed := removeRemoved{Name: e.Name, Service: e.Service, Profile: e.Profile, Agents: []string{}}
		pin := resolveBinding(ctx, e)
		var errs []error
		for _, agentName := range slices.Sorted(maps.Keys(e.Agents)) {
			if targets != nil && !slices.ContainsFunc(targets, func(a *agents.Agent) bool { return a.Name == agentName }) {
				continue
			}
			a := agents.ByName(agentName)
			if a == nil {
				// This CLI does not know the agent, so it cannot edit the config. The record is kept, so that the CLI that
				// knows the agent can still remove the server.
				out.Skipped = append(out.Skipped, removeSkipped{Name: e.Name, Agent: agentName, Reason: unknownAgentReason(agentName)})
				continue
			}
			if !a.SupportsMCP() {
				// There is no config this CLI can edit for the agent, so only the record can go.
				removed.Agents = append(removed.Agents, agentName)
				removed.RecordOnly = append(removed.RecordOnly, agentName)
				recordOnly = append(recordOnly, len(out.results))
				out.results = append(out.results, removeServerResult(e.Service, agentName, removeDeleted, nil))
				drops = append(drops, recordDrop{name: e.Name, agent: agentName})
				continue
			}
			plan, err := removeFromAgent(ctx, files, e, a, pin, len(agentNames) > 0)
			// A pruned entry changes the state only, so a save that fails undoes it, as for a record-only removal.
			if plan.outcome == removePruned && err == nil {
				recordOnly = append(recordOnly, len(out.results))
			}
			out.results = append(out.results, removeServerResult(e.Service, agentName, plan.outcome, err))
			if err != nil {
				errs = append(errs, err)
				out.Skipped = append(out.Skipped, removeSkipped{Name: e.Name, Agent: agentName, Reason: fmt.Sprintf("could not update %s: %s", agentDisplayName(agentName), err)})
				continue
			}
			if plan.outcome == removeKept {
				out.Skipped = append(out.Skipped, removeSkipped{Name: e.Name, Agent: agentName, Reason: plan.reason})
				continue
			}
			removed.Agents = append(removed.Agents, agentName)
			drops = append(drops, recordDrop{name: e.Name, agent: agentName})
			for _, reason := range plan.leftAlone {
				out.Skipped = append(out.Skipped, removeSkipped{Name: e.Name, Agent: agentName, Reason: reason})
			}
		}
		// A server with no agent left in state is removed from state too, so it is reported as removed. Only its
		// record goes, which is a change to the state.
		if len(e.Agents) == 0 {
			recordOnly = append(recordOnly, len(out.results))
			out.results = append(out.results, protos.AitoolsMcpServerResult{Service: mcpServiceType(e.Service), Outcome: protos.AitoolsTargetOutcomeChanged})
		}
		if len(removed.Agents) > 0 || len(e.Agents) == 0 {
			out.Removed = append(out.Removed, removed)
		}
		// A server that no agent gave up on is a failure. So is any agent failure when --agents named the agent.
		if len(errs) > 0 && (len(removed.Agents) == 0 || targets != nil) {
			for _, err := range errs {
				agentErrs = appendDistinct(agentErrs, err)
			}
		}
	}

	var saveErr error
	if len(drops) > 0 || slices.ContainsFunc(selected, func(e mcp.Entry) bool { return len(e.Agents) == 0 }) {
		saveErr = dropRecords(ctx, drops, selected)
	}
	if saveErr != nil {
		// A record-only removal changes nothing but the state, so the save that failed undid it.
		for _, i := range recordOnly {
			out.results[i].Outcome = protos.AitoolsTargetOutcomeFailed
			out.results[i].ErrorCategory = errorCategory(saveErr)
		}
	}
	if !jsonMode {
		printRemoveResults(ctx, out)
	}
	if saveErr != nil {
		return saveErr
	}
	if len(agentErrs) > 0 {
		return &agentErrors{err: errors.Join(agentErrs...)}
	}
	return nil
}

// validateMCPRemoveFlags rejects a choice of servers that cannot run, before anything is read or written.
// Without a choice, a terminal run shows the picker; any other run needs one.
func validateMCPRemoveFlags(all bool, names []string, profileName string, interactive bool) error {
	if all && len(names) > 0 {
		return failure(protos.AitoolsErrorCategoryInvalidFlags, "--all cannot be combined with --names")
	}
	if !all && len(names) == 0 && profileName == "" && !interactive {
		return failure(protos.AitoolsErrorCategoryInvalidFlags, "%s", removeFlagsHint)
	}
	return nil
}

// selectInstalls returns the state entries that names pick, in state order, or every entry in scope when names is
// empty. profileName, when set, narrows the scope to that profile. A name that matches nothing is an error, so a typo
// does not look like a successful removal. A profile with no installed server selects nothing, so removing from it is a
// no-op, as --all is on an empty state.
func selectInstalls(state *mcp.State, names []string, profileName string) ([]mcp.Entry, error) {
	var scope []mcp.Entry
	for _, e := range state.Servers {
		if profileName == "" || e.Profile == profileName {
			scope = append(scope, e)
		}
	}
	where := ""
	if profileName != "" {
		where = " on profile " + profileName
	}
	chosen := make([]bool, len(scope))
	for _, name := range names {
		matched, err := matchInstalls(scope, name)
		if err != nil {
			return nil, err
		}
		if len(matched) == 0 {
			return nil, &noRecordError{name: name, err: failure(protos.AitoolsErrorCategoryMCPServiceNotFound, "no installed MCP server named %s%s", name, where)}
		}
		for _, i := range matched {
			chosen[i] = true
		}
	}
	if len(names) == 0 {
		return scope, nil
	}
	var selected []mcp.Entry
	for i, e := range scope {
		if chosen[i] {
			selected = append(selected, e)
		}
	}
	return selected, nil
}

// noRecordError is the failure for a --names value that no install record matches. It keeps the failure's category and
// message, and names the value, so that runMcpRemove can say how to adopt an unmanaged entry of that name.
type noRecordError struct {
	name string
	err  error
}

func (e *noRecordError) Error() string {
	return e.err.Error()
}

func (e *noRecordError) Unwrap() error {
	return e.err
}

// withAdoptHint adds to a noRecordError how to adopt the unmanaged entry of its name, when an agent config holds one
// that runs `aitools mcp serve` with no install record. Any other error is returned as it is.
func withAdoptHint(ctx context.Context, err error, state *mcp.State) error {
	missing, ok := errors.AsType[*noRecordError](err)
	if !ok {
		return err
	}
	for _, u := range unmanagedEntries(ctx, newAgentFiles(), state) {
		if u.Kind != unmanagedKindNoRecord || !strings.EqualFold(u.Name, missing.name) {
			continue
		}
		hint := fmt.Sprintf("%s has an entry named %s that runs aitools mcp serve with no install record", agentDisplayName(u.Agent), u.Name)
		if u.fix != "" {
			hint += ". To adopt it, so that remove works, run `" + u.fix + "`"
		}
		return withCategory(fmt.Errorf("%w; %s", err, hint), protos.AitoolsErrorCategoryMCPServiceNotFound)
	}
	return err
}

// matchInstalls returns the indexes of the entries of scope that name means. A server name selects that server: an
// exact match first, and otherwise a match without case, which is refused when several servers match that way, since
// state can hold names that differ only in case. Only when no server has the name, it means the service: the full
// service name, or the leaf of a service name (its last part, when the name has no dot). A leaf is refused when the
// entries it matches are of more than one service, since then it does not say which one is meant.
func matchInstalls(scope []mcp.Entry, name string) ([]int, error) {
	if i := slices.IndexFunc(scope, func(e mcp.Entry) bool { return e.Name == name }); i >= 0 {
		return []int{i}, nil
	}
	var byName []int
	var serverNames []string
	for i, e := range scope {
		if strings.EqualFold(e.Name, name) {
			byName = append(byName, i)
			serverNames = append(serverNames, e.Name)
		}
	}
	if len(byName) > 1 {
		return nil, failure(protos.AitoolsErrorCategoryInvalidFlags, "%s matches more than one MCP server (%s); pass the exact server name",
			name, strings.Join(serverNames, ", "))
	}
	if len(byName) == 1 {
		return byName, nil
	}
	var matched []int
	var services []string
	for i, e := range scope {
		if !matchesService(e, name) {
			continue
		}
		matched = append(matched, i)
		if !slices.ContainsFunc(services, func(s string) bool { return strings.EqualFold(s, e.Service) }) {
			services = append(services, e.Service)
		}
	}
	if len(services) > 1 && !strings.Contains(name, ".") {
		return nil, failure(protos.AitoolsErrorCategoryInvalidFlags, "%s matches more than one MCP service (%s); pass the full service name or a server name",
			name, strings.Join(services, ", "))
	}
	return matched, nil
}

// matchesService reports whether a --names value means the entry's service: its full name, or the leaf of its service
// name when the value has no dot. Names compare without case.
func matchesService(e mcp.Entry, name string) bool {
	if strings.EqualFold(e.Service, name) {
		return true
	}
	return !strings.Contains(name, ".") && strings.EqualFold(mcp.LeafOf(e.Service), name)
}

// checkNamedInstalled refuses a --names run in which --agents narrows a named server to no agent, as checkTargetsInstalled
// refuses an agent that no server is installed for. The run is all or nothing, so no other server is removed either. Without
// --agents, every named server is removed from the agents it is installed for, so there is nothing to check.
func checkNamedInstalled(selected []mcp.Entry, targets []*agents.Agent) error {
	if targets == nil {
		return nil
	}
	var servers []string
	for _, e := range selected {
		if !slices.ContainsFunc(targets, func(a *agents.Agent) bool {
			_, ok := e.Agents[a.Name]
			return ok
		}) {
			servers = append(servers, e.Name)
		}
	}
	if len(servers) == 0 {
		return nil
	}
	var agentNames []string
	for _, a := range targets {
		agentNames = append(agentNames, a.Name)
	}
	return failure(protos.AitoolsErrorCategoryMCPServiceNotFound, "nothing was removed: named MCP servers are not installed for %s: %s",
		mcpAgentList(agentNames), strings.Join(servers, ", "))
}

// checkTargetsInstalled refuses an --agents agent that none of the selected servers is installed for. With no
// selected server there is nothing to check, so removing everything from an empty state stays a no-op.
func checkTargetsInstalled(selected []mcp.Entry, targets []*agents.Agent) error {
	if len(selected) == 0 {
		return nil
	}
	for _, a := range targets {
		if !slices.ContainsFunc(selected, func(e mcp.Entry) bool {
			_, ok := e.Agents[a.Name]
			return ok
		}) {
			return failure(protos.AitoolsErrorCategoryMCPServiceNotFound, "no selected MCP server is installed for %s", agentDisplayName(a.Name))
		}
	}
	return nil
}

// keptReason says why a Databricks entry that was changed outside this CLI is left in place, and how to clear it. Only
// such an entry keeps a server.
func keptReason(e mcp.Entry, a *agents.Agent, path string, entry map[string]any) string {
	return fmt.Sprintf("%s has an entry named %s that was changed outside this CLI; left in place. To remove it, %s",
		agentDisplayName(a.Name), e.Name, changedEntryFix(e.Name, path, entry))
}

// otherEntryReason says why an entry of the server's name, in path, is left in place: this CLI did not write it, and it
// is not a Databricks entry. This CLI does not change it, and the record for the agent is dropped.
func otherEntryReason(e mcp.Entry, a *agents.Agent, path string) string {
	return fmt.Sprintf("%s has an entry named %s in %s that this CLI did not write for this server; left in place and not changed. The install record for %s was dropped.",
		agentDisplayName(a.Name), e.Name, filepath.ToSlash(path), agentDisplayName(a.Name))
}

// removeAgentError adds the --agents hint to a removal error for agent a, when --agents named a. Only then is --agents
// the way out, as it is for add.
func removeAgentError(err error, agent string, explicit bool) error {
	if !explicit {
		return err
	}
	return withAgentsHint(err, agent)
}

// removeFromAgent removes the server's entry for agent a, as planRemoval plans it, and returns the plan. An agent config
// that holds no entry of this CLI's is pruned: the record goes without an edit. Each copy of this CLI's entry is removed,
// in the file the record names and in the file the agent reads now when the config moved by copying, since the agent would
// otherwise keep running the server. Entries this CLI did not write are left in place, and the plan says so.
func removeFromAgent(ctx context.Context, files *agentFiles, e mcp.Entry, a *agents.Agent, pin bindingFile, explicit bool) (agentRemoval, error) {
	plan, err := planRemoval(ctx, files, e, a, pin)
	if err != nil {
		return agentRemoval{}, err
	}
	if plan.outcome != removeDeleted {
		return plan, nil
	}
	for _, path := range plan.paths {
		if _, err := agentconfig.RemoveAt(ctx, a, path, e.Name); err != nil {
			return agentRemoval{}, removeAgentError(err, a.Name, explicit)
		}
	}
	return plan, nil
}

// agentRemoval is what removing a server from one agent would do, decided before anything is written. paths are the
// config files the server's own entry is removed from. reason says why the server is kept, when a Databricks entry that
// this CLI did not write is in the agent config. leftAlone holds the reasons for the entries left in place, and each one
// drops the record for the agent.
type agentRemoval struct {
	outcome   removeOutcome
	reason    string
	paths     []string
	leftAlone []string
}

// planRemoval decides what removeFromAgent does for agent a, without writing anything. See removeFromAgent. The entry of
// the server's name is looked for where locateEntry finds it, and also in the file the agent reads now, which can hold
// its own copy. Each copy is this CLI's, which is removed; a Databricks entry this CLI did not write, which keeps the
// server; or any other entry, which is left in place.
func planRemoval(ctx context.Context, files *agentFiles, e mcp.Entry, a *agents.Agent, pin bindingFile) (agentRemoval, error) {
	loc, err := files.locateEntry(ctx, e, a)
	if err != nil {
		return agentRemoval{}, err
	}
	if !loc.found {
		return agentRemoval{outcome: removePruned}, nil
	}
	type entryCopy struct {
		path  string
		entry map[string]any
	}
	copies := []entryCopy{{path: loc.path, entry: loc.entry}}
	current, err := files.currentPath(ctx, a)
	if err != nil {
		return agentRemoval{}, err
	}
	if !mcp.SamePath(current, loc.path) {
		cur := files.file(a, current)
		if cur.err != nil {
			return agentRemoval{}, cur.err
		}
		if entry, found := cur.entries[e.Name]; found {
			copies = append(copies, entryCopy{path: current, entry: entry})
		}
	}
	var paths, leftAlone []string
	for _, c := range copies {
		owned, err := pin.owns(ctx, e, c.entry)
		if err != nil {
			return agentRemoval{}, err
		}
		switch {
		case owned:
			paths = append(paths, c.path)
		case mcpconfig.IsDatabricksEntry(c.entry):
			return agentRemoval{outcome: removeKept, reason: keptReason(e, a, c.path, c.entry)}, nil
		default:
			leftAlone = append(leftAlone, otherEntryReason(e, a, c.path))
		}
	}
	outcome := removeDeleted
	if len(paths) == 0 {
		outcome = removePruned
	}
	return agentRemoval{outcome: outcome, paths: paths, leftAlone: leftAlone}, nil
}

// unknownAgentReason says why a record for an agent this CLI does not know is kept.
func unknownAgentReason(agent string) string {
	return "unknown agent " + agent + "; this CLI cannot edit its config, so its record is kept"
}

// refusedByName returns the skipped entries that refuse a --names run, with their reasons. A named server refuses the
// run when an entry of it is kept, or when every record of it is for an agent this CLI does not know, since then nothing
// of it can be removed. It only reads the agent configs. A read error is left to the removal, which reports it per agent.
func refusedByName(ctx context.Context, files *agentFiles, selected []mcp.Entry, targets []*agents.Agent) []removeSkipped {
	var refused []removeSkipped
	for _, e := range selected {
		pin := resolveBinding(ctx, e)
		var entry []removeSkipped
		var hasKept, hasUnknown, removable bool
		for _, agentName := range slices.Sorted(maps.Keys(e.Agents)) {
			if targets != nil && !slices.ContainsFunc(targets, func(a *agents.Agent) bool { return a.Name == agentName }) {
				continue
			}
			a := agents.ByName(agentName)
			if a == nil {
				hasUnknown = true
				entry = append(entry, removeSkipped{Name: e.Name, Agent: agentName, Reason: unknownAgentReason(agentName)})
				continue
			}
			if !a.SupportsMCP() {
				removable = true
				continue
			}
			plan, err := planRemoval(ctx, files, e, a, pin)
			if err != nil {
				removable = true
				continue
			}
			if plan.outcome != removeKept {
				removable = true
				continue
			}
			hasKept = true
			entry = append(entry, removeSkipped{Name: e.Name, Agent: agentName, Reason: plan.reason})
		}
		if hasKept || (hasUnknown && !removable) {
			refused = append(refused, entry...)
		}
	}
	return refused
}

// refusedNames returns the distinct server names of the refused entries, in order, for the failure message.
func refusedNames(refused []removeSkipped) string {
	var names []string
	for _, s := range refused {
		if !slices.Contains(names, s.Name) {
			names = append(names, s.Name)
		}
	}
	return strings.Join(names, ", ")
}

// dropRecords deletes the agent records in drops from state, then the entries that were selected or dropped and have
// no agent left. Each record is matched by the name of its server, which is unique in state. The state file is re-read
// first, so entries written since the selection are kept, and an entry that was not selected is never removed here.
func dropRecords(ctx context.Context, drops []recordDrop, selected []mcp.Entry) error {
	state, err := mcp.LoadState(ctx)
	if err != nil {
		return err
	}
	for _, d := range drops {
		i := slices.IndexFunc(state.Servers, func(e mcp.Entry) bool {
			return e.Name == d.name
		})
		if i >= 0 {
			delete(state.Servers[i].Agents, d.agent)
		}
	}
	touched := map[string]bool{}
	for _, d := range drops {
		touched[d.name] = true
	}
	for _, e := range selected {
		touched[e.Name] = true
	}
	state.Servers = slices.DeleteFunc(state.Servers, func(e mcp.Entry) bool {
		return touched[e.Name] && len(e.Agents) == 0
	})
	return saveMCPState(ctx, state)
}

// printRemovePlan lists what the picked servers are removed from, before the confirm of an interactive remove.
func printRemovePlan(ctx context.Context, selected []mcp.Entry, targets []*agents.Agent) {
	cmdio.LogString(ctx, "")
	for _, line := range removePlanLines(selected, targets) {
		cmdio.LogString(ctx, line)
	}
	cmdio.LogString(ctx, "")
}

// removePlanLines returns the plan of a remove: one line per agent, with the servers removed from it. Only the agents
// named with --agents, when it is set, get a line.
func removePlanLines(selected []mcp.Entry, targets []*agents.Agent) []string {
	servers := map[string][]string{}
	for _, e := range selected {
		for name := range e.Agents {
			if targets != nil && !slices.ContainsFunc(targets, func(a *agents.Agent) bool { return a.Name == name }) {
				continue
			}
			servers[name] = append(servers[name], e.Name)
		}
	}
	lines := []string{"Plan:"}
	for _, name := range slices.Sorted(maps.Keys(servers)) {
		lines = append(lines, fmt.Sprintf("  %-14s remove %s", agentDisplayName(name), strings.Join(servers[name], ", ")))
	}
	return lines
}

// printRemoveResults writes the text result: each removed server, then each skipped entry with its reason.
func printRemoveResults(ctx context.Context, out *removeOutput) {
	if len(out.Removed) == 0 && len(out.Skipped) == 0 {
		cmdio.LogString(ctx, "No Databricks MCP servers to remove.")
		return
	}
	for _, r := range out.Removed {
		line := fmt.Sprintf("Removed %s  %s  profile %s", r.Name, r.Service, r.Profile)
		if len(r.Agents) > 0 {
			line += fmt.Sprintf("  (%s)", mcpAgentList(r.Agents))
		}
		cmdio.LogString(ctx, line)
		for _, name := range r.RecordOnly {
			cmdio.LogString(ctx, fmt.Sprintf("  only the record of %s was dropped: this CLI has no MCP config to edit for it", agentDisplayName(name)))
		}
	}
	for _, s := range out.Skipped {
		cmdio.LogString(ctx, fmt.Sprintf("Skipped %s: %s", s.Name, s.Reason))
	}
}
