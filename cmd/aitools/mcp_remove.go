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
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
)

// removeFlagsHint is the error for a run that names no servers. Tests check it too.
const removeFlagsHint = "pass --names, --profile or --all; interactive selection is not available yet"

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
}

type removeRemoved struct {
	Name    string   `json:"name"`
	Service string   `json:"service"`
	Profile string   `json:"profile"`
	Agents  []string `json:"agents"`
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
	// removeKept means the entry is left in place, because it is not the one this CLI wrote or the agent config moved.
	removeKept
)

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

Choose servers with --names (server names as listed by 'aitools mcp list', or service names),
--profile, or --all. --profile narrows --all or --names to one profile, and --agents narrows
them to some agents. Interactive selection is not available yet, so one of --names, --profile
or --all is required.

Only entries this CLI wrote are removed. An entry that was changed outside this CLI, or whose agent
config moved, is left in place and reported. Entries that run 'aitools mcp serve' without an install
record are not chosen by any flag; 'aitools mcp list' shows them as unmanaged. Provider credentials
are not deleted.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			jsonMode := installOutputIsJSON(cmd)
			if jsonMode {
				// JSON mode writes only the result document to stdout.
				ctx = cmdio.WithQuiet(ctx)
			}

			out := &removeOutput{Removed: []removeRemoved{}, Skipped: []removeSkipped{}}
			runErr := runMcpRemove(ctx, cmd, f, jsonMode, out)
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
	if err := validateMCPRemoveFlags(f.all, names, profileName); err != nil {
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
	selected, err := selectInstalls(state, names, profileName)
	if err != nil {
		return err
	}
	if err := checkTargetsInstalled(selected, targets); err != nil {
		return err
	}

	// Each agent config is read once, when the first selected server needs it.
	configs := map[string]agentConfig{}
	var drops []recordDrop
	var agentErrs []error
	for _, e := range selected {
		removed := removeRemoved{Name: e.Name, Service: e.Service, Profile: e.Profile, Agents: []string{}}
		var errs []error
		for _, agentName := range slices.Sorted(maps.Keys(e.Agents)) {
			if targets != nil && !slices.ContainsFunc(targets, func(a *agents.Agent) bool { return a.Name == agentName }) {
				continue
			}
			cfg, ok := configs[agentName]
			if !ok {
				cfg = readAgentConfig(ctx, agentName)
				configs[agentName] = cfg
			}
			outcome, reason, err := removeAgentEntry(ctx, e, agentName, cfg)
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
		if len(removed.Agents) > 0 {
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
	if len(drops) > 0 {
		saveErr = dropRecords(ctx, drops)
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
func validateMCPRemoveFlags(all bool, names []string, profileName string) error {
	if all && len(names) > 0 {
		return failure(protos.AitoolsErrorCategoryInvalidFlags, nil, "--all cannot be combined with --names")
	}
	if !all && len(names) == 0 && profileName == "" {
		return failure(protos.AitoolsErrorCategoryInvalidFlags, nil, removeFlagsHint)
	}
	return nil
}

// selectInstalls returns the state entries named by names (server or service names), or every entry when names is
// empty. profileName, when set, narrows the choice to that profile. A name that matches nothing, and a profile with no
// installed server, are errors, so a typo does not look like a successful removal.
func selectInstalls(state *mcp.State, names []string, profileName string) ([]mcp.Entry, error) {
	where := ""
	if profileName != "" {
		where = " on profile " + profileName
	}
	var selected []mcp.Entry
	found := make([]bool, len(names))
	inProfile := 0
	for _, e := range state.Servers {
		if profileName != "" && e.Profile != profileName {
			continue
		}
		inProfile++
		chosen := len(names) == 0
		for i, name := range names {
			if matchesInstall(e, name) {
				found[i] = true
				chosen = true
			}
		}
		if chosen {
			selected = append(selected, e)
		}
	}
	for i, name := range names {
		if !found[i] {
			return nil, failure(categoryServiceNotFound, nil, "no installed MCP server named %s%s", name, where)
		}
	}
	if profileName != "" && inProfile == 0 {
		return nil, failure(categoryServiceNotFound, nil, "no installed MCP servers on profile %s", profileName)
	}
	return selected, nil
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
			return failure(categoryServiceNotFound, nil, "no selected MCP server is installed for %s", a.DisplayName)
		}
	}
	return nil
}

// matchesInstall reports whether a --names value means the entry: its server name, or its service name.
func matchesInstall(e mcp.Entry, name string) bool {
	return e.Name == name || strings.EqualFold(e.Service, name)
}

// removeAgentEntry removes the server from one agent config, but only when the entry there is the one this CLI
// wrote. The config was read by the caller; a config that could not be read is an error, not a removal. A config
// that moved is kept, since the entry may be in the file the record names.
func removeAgentEntry(ctx context.Context, e mcp.Entry, agentName string, cfg agentConfig) (removeOutcome, string, error) {
	if cfg.err != nil {
		return 0, "", cfg.err
	}
	recorded := e.Agents[agentName].Path
	if !mcp.SamePath(recorded, cfg.path) {
		return removeKept, fmt.Sprintf("%s reads its config from %s now, but %s was written to %s; left in place, and its record kept",
			agentDisplayName(agentName), filepath.ToSlash(cfg.path), e.Name, filepath.ToSlash(recorded)), nil
	}
	current, found := cfg.entries[e.Name]
	if !found {
		return removePruned, "", nil
	}
	owned, err := ownedBy(ctx, e, current)
	if err != nil {
		return 0, "", err
	}
	if !owned {
		return removeKept, fmt.Sprintf("%s has an entry named %s that was changed outside this CLI; left in place", agentDisplayName(agentName), e.Name), nil
	}
	if _, err := agentconfig.Remove(ctx, cfg.agent, e.Name); err != nil {
		return 0, "", err
	}
	return removeDeleted, "", nil
}

// dropRecords deletes the agent records from state and then the entries left without any agent. Each record is
// matched by the name of its server, which is unique in state. The state file is re-read first, so entries written
// since the selection are kept.
func dropRecords(ctx context.Context, drops []recordDrop) error {
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
	state.Servers = slices.DeleteFunc(state.Servers, func(e mcp.Entry) bool {
		return len(e.Agents) == 0
	})
	return mcp.SaveState(ctx, state)
}

func printRemoveResults(ctx context.Context, out *removeOutput) {
	if len(out.Removed) == 0 && len(out.Skipped) == 0 {
		cmdio.LogString(ctx, "No Databricks MCP servers to remove.")
		return
	}
	for _, r := range out.Removed {
		cmdio.LogString(ctx, fmt.Sprintf("Removed %s  %s  profile %s  (%s)", r.Name, r.Service, r.Profile, mcpAgentList(r.Agents)))
	}
	for _, s := range out.Skipped {
		cmdio.LogString(ctx, fmt.Sprintf("Skipped %s: %s", s.Name, s.Reason))
	}
}
