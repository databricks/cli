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

type removeSkipped struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// removeOutcome is what happened to one server in one agent config.
type removeOutcome int

const (
	// removeDeleted means the entry was removed from the agent config.
	removeDeleted removeOutcome = iota
	// removePruned means the entry was already gone from the agent config, so only the state record goes.
	removePruned
	// removeKept means the entry on disk is not the one this CLI wrote, so it is left in place.
	removeKept
)

// recordDrop names one agent record to delete from state, identified by service and profile.
type recordDrop struct {
	service string
	profile string
	agent   string
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

Only entries this CLI wrote are removed. An entry that was changed outside this CLI is left in
place and reported. Provider credentials are not deleted.`,
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
				return err
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

// runMcpRemove removes the chosen servers from the agent configs and records the result. An agent
// failure is reported in skipped; it is returned as an agentErrors only when --agents named the agent.
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

	var drops []recordDrop
	var agentErrs []error
	for _, e := range selected {
		removed := removeRemoved{Name: e.Name, Service: e.Service, Profile: e.Profile, Agents: []string{}}
		for _, agentName := range slices.Sorted(maps.Keys(e.Agents)) {
			if targets != nil && !slices.ContainsFunc(targets, func(a *agents.Agent) bool { return a.Name == agentName }) {
				continue
			}
			outcome, err := removeAgentEntry(ctx, e, agentName)
			if err != nil {
				agentErrs = append(agentErrs, err)
				out.Skipped = append(out.Skipped, removeSkipped{Name: e.Name, Reason: fmt.Sprintf("could not update %s: %s", agentDisplayName(agentName), err)})
				continue
			}
			if outcome == removeKept {
				out.Skipped = append(out.Skipped, removeSkipped{Name: e.Name, Reason: fmt.Sprintf("%s has an entry named %s that was changed outside this CLI; left in place", agentDisplayName(agentName), e.Name)})
				continue
			}
			removed.Agents = append(removed.Agents, agentName)
			drops = append(drops, recordDrop{service: e.Service, profile: e.Profile, agent: agentName})
		}
		if len(removed.Agents) > 0 {
			out.Removed = append(out.Removed, removed)
		}
	}

	if len(drops) > 0 {
		if err := dropRecords(ctx, drops); err != nil {
			return err
		}
	}
	if !jsonMode {
		printRemoveResults(ctx, out)
	}
	if len(targets) > 0 && len(agentErrs) > 0 {
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
		return failure(protos.AitoolsErrorCategoryInvalidFlags, nil, "pass --names, --profile or --all; interactive selection is not available yet")
	}
	return nil
}

// selectInstalls returns the state entries named by names (server or service names), or every entry
// when names is empty. profileName, when set, narrows the choice to that profile. A name that matches
// nothing is an error, so a typo does not look like a successful removal.
func selectInstalls(state *mcp.State, names []string, profileName string) ([]mcp.Entry, error) {
	var inProfile, selected []mcp.Entry
	for _, e := range state.Servers {
		if profileName == "" || e.Profile == profileName {
			inProfile = append(inProfile, e)
		}
	}
	for _, e := range inProfile {
		if len(names) == 0 || slices.ContainsFunc(names, func(n string) bool { return matchesInstall(e, n) }) {
			selected = append(selected, e)
		}
	}
	where := ""
	if profileName != "" {
		where = " on profile " + profileName
	}
	for _, name := range names {
		if !slices.ContainsFunc(inProfile, func(e mcp.Entry) bool { return matchesInstall(e, name) }) {
			return nil, failure(categoryServiceNotFound, nil, "no installed MCP server named %s%s", name, where)
		}
	}
	return selected, nil
}

// matchesInstall reports whether a --names value means the entry: its server name, or its service name.
func matchesInstall(e mcp.Entry, name string) bool {
	return e.Name == name || strings.EqualFold(e.Service, name)
}

// removeAgentEntry removes the server from one agent config, but only when the entry there is the one
// this CLI wrote. The config is read first, so a config that cannot be parsed is an error, not a removal.
func removeAgentEntry(ctx context.Context, e mcp.Entry, agentName string) (removeOutcome, error) {
	a := agents.ByName(agentName)
	if a == nil {
		return 0, fmt.Errorf("%s is not a known agent", agentName)
	}
	entries, err := agentconfig.Read(ctx, a)
	if err != nil {
		return 0, err
	}
	current, found := entries[e.Name]
	switch {
	case !found:
		return removePruned, nil
	case !isOwnedEntry(current, e):
		return removeKept, nil
	}
	if _, err := agentconfig.Remove(ctx, a, e.Name); err != nil {
		return 0, err
	}
	return removeDeleted, nil
}

// dropRecords deletes the agent records from state and then the entries left without any agent. The
// state file is re-read first, so entries written since the selection are kept.
func dropRecords(ctx context.Context, drops []recordDrop) error {
	state, err := mcp.LoadState(ctx)
	if err != nil {
		return err
	}
	for _, d := range drops {
		i := slices.IndexFunc(state.Servers, func(e mcp.Entry) bool {
			return e.Service == d.service && e.Profile == d.profile
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
