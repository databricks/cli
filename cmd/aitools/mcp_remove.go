package aitools

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/aitools/mcp/agentconfig"
	"github.com/databricks/cli/libs/cmdio"
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
	// removePruned means the entry was already gone from the agent config, so only the state record goes.
	removePruned
	// removeKept means the entry is left in place, because it is not the one this CLI wrote.
	removeKept
)

// saveMCPState writes the MCP state. It is a variable so that a test can make the save fail.
var saveMCPState = mcp.SaveState

// recordDrop names one agent record to delete from state: the server's name and the agent.
type recordDrop struct {
	name  string
	agent string
}

// NewMcpRemoveCmd returns `aitools mcp remove`. It needs no network.
func NewMcpRemoveCmd() *cobra.Command {
	var f mcpRemoveFlags
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove Databricks managed MCP servers from coding agents",
		Long: `Remove Databricks managed MCP servers that this CLI installed.

Choose servers with --names (server names as listed by 'aitools mcp list', or service names; the
last part of a service name, such as github, is enough when only one schema has it), --profile, or
--all. --profile narrows --all or --names to one profile, and --agents narrows them to some agents.
In a terminal, with none of --names, --profile or --all, the command shows the installed servers to
choose from. Without a terminal, one of them is required.

Only entries this CLI wrote are removed. An entry that was changed outside this CLI is left in place
and reported. When an agent's config moved, the entry is removed from the file it was written to.
Entries that run 'aitools mcp serve' without an install record are not chosen by any flag;
'aitools mcp list' shows them as unmanaged. Provider credentials are not deleted.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			jsonMode := installOutputIsJSON(cmd)

			out := &removeOutput{Removed: []removeRemoved{}, Skipped: []removeSkipped{}}
			runErr := runMcpRemove(ctx, cmd, f, jsonMode, out)
			configureUploadAuth(cmd)
			logMcpEvent(ctx, protos.AitoolsMcpEvent{
				CallingAgent: callingAgentType(),
				Operation:    protos.AitoolsMcpOperationRemove,
				Servers:      out.results,
			})
			if !jsonMode {
				return runErr
			}

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
		},
	}

	cmd.Flags().StringVar(&f.names, "names", "", "Servers to remove: server names or service names, comma-separated")
	cmd.Flags().StringVar(&f.agents, "agents", "", "Agents to remove from, comma-separated (default: every agent the server is installed for)")
	cmd.Flags().BoolVar(&f.all, "all", false, "Remove every server this CLI installed")
	return cmd
}

// runMcpRemove removes the chosen servers from the agent configs and records the result. Results are printed
// before a state failure is returned. An agent failure fails the command when --agents named the agent, or when a
// selected server was removed from no agent; otherwise it is reported in skipped.
func runMcpRemove(ctx context.Context, cmd *cobra.Command, f mcpRemoveFlags, jsonMode bool, out *removeOutput) error {
	names := splitAndTrim(f.names)
	agentNames := splitAndTrim(f.agents)
	profileName := profileFlag(cmd)
	interactive := !jsonMode && cmdio.IsPromptSupported(ctx)
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
			proceed, err := promptProceed()
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
			return err
		}
		if err := checkTargetsInstalled(selected, targets); err != nil {
			return err
		}
	}
	if targets != nil {
		// --agents chooses the servers installed for those agents, so a server with no agent left is not chosen.
		selected = slices.DeleteFunc(selected, func(e mcp.Entry) bool { return len(e.Agents) == 0 })
	}

	// Each agent config file is read once per command, and each server's config file is resolved once.
	files := newAgentFiles()
	var drops []recordDrop
	var agentErrs []error
	for _, e := range selected {
		removed := removeRemoved{Name: e.Name, Service: e.Service, Profile: e.Profile, Agents: []string{}}
		pin := resolveBinding(ctx, e)
		var errs []error
		for _, agentName := range slices.Sorted(maps.Keys(e.Agents)) {
			if targets != nil && !slices.ContainsFunc(targets, func(a *agents.Agent) bool { return a.Name == agentName }) {
				continue
			}
			a := agents.ByName(agentName)
			if a == nil || !a.SupportsMCP() {
				// There is no config this CLI can edit for the agent, so only the record can go.
				removed.Agents = append(removed.Agents, agentName)
				removed.RecordOnly = append(removed.RecordOnly, agentName)
				out.results = append(out.results, removeServerResult(e.Service, agentName, removeDeleted, nil))
				drops = append(drops, recordDrop{name: e.Name, agent: agentName})
				continue
			}
			outcome, reason, err := removeFromAgent(ctx, files, e, a, pin)
			out.results = append(out.results, removeServerResult(e.Service, agentName, outcome, err))
			if err != nil {
				errs = append(errs, err)
				out.Skipped = append(out.Skipped, removeSkipped{Name: e.Name, Agent: agentName, Reason: fmt.Sprintf("could not update %s: %s", agentDisplayName(agentName), err)})
				continue
			}
			if outcome == removeKept {
				out.Skipped = append(out.Skipped, removeSkipped{Name: e.Name, Agent: agentName, Reason: reason})
				continue
			}
			removed.Agents = append(removed.Agents, agentName)
			drops = append(drops, recordDrop{name: e.Name, agent: agentName})
		}
		// A server with no agent left in state is removed from state too, so it is reported as removed. Only its
		// record goes, which is a change to the state.
		if len(e.Agents) == 0 {
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
		return failure(protos.AitoolsErrorCategoryInvalidFlags, nil, "--all cannot be combined with --names")
	}
	if !all && len(names) == 0 && profileName == "" && !interactive {
		return failure(protos.AitoolsErrorCategoryInvalidFlags, nil, removeFlagsHint)
	}
	return nil
}

// selectInstalls returns the state entries that names pick, in state order, or every entry in scope when names is
// empty. profileName, when set, narrows the scope to that profile. A name that matches nothing, and a profile with no
// installed server, are errors, so a typo does not look like a successful removal.
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
			return nil, failure(protos.AitoolsErrorCategoryMCPServiceNotFound, nil, "no installed MCP server named %s%s", name, where)
		}
		for _, i := range matched {
			chosen[i] = true
		}
	}
	if profileName != "" && len(scope) == 0 {
		return nil, failure(protos.AitoolsErrorCategoryMCPServiceNotFound, nil, "no installed MCP servers on profile %s", profileName)
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
		return nil, failure(protos.AitoolsErrorCategoryInvalidFlags, nil, "%s matches more than one MCP server (%s); pass the exact server name",
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
		return nil, failure(protos.AitoolsErrorCategoryInvalidFlags, nil, "%s matches more than one MCP service (%s); pass the full service name or a server name",
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
			return failure(protos.AitoolsErrorCategoryMCPServiceNotFound, nil, "no MCP server is installed for %s", agentDisplayName(a.Name))
		}
	}
	return nil
}

// keptReason says why an entry is left in place, and how to clear it.
func keptReason(e mcp.Entry, a *agents.Agent, path string, entry map[string]any) string {
	return fmt.Sprintf("%s has an entry named %s that was changed outside this CLI; left in place. To remove it, %s",
		agentDisplayName(a.Name), e.Name, changedEntryFix(e.Name, path, entry))
}

// removeFromAgent removes the server's entry for agent a, but only when the entry there is the one this CLI wrote. The
// entry is looked for where locateEntry finds it. An agent config that holds no entry is pruned: the record goes without
// an edit.
func removeFromAgent(ctx context.Context, files *agentFiles, e mcp.Entry, a *agents.Agent, pin bindingFile) (removeOutcome, string, error) {
	loc, err := files.locateEntry(ctx, e, a)
	if err != nil {
		return 0, "", err
	}
	if !loc.found {
		return removePruned, "", nil
	}
	owned, err := pin.owns(ctx, e, loc.entry)
	if err != nil {
		return 0, "", err
	}
	if !owned {
		return removeKept, keptReason(e, a, loc.path, loc.entry), nil
	}
	if _, err := agentconfig.RemoveAt(ctx, a, loc.path, e.Name); err != nil {
		return 0, "", err
	}
	return removeDeleted, "", nil
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
