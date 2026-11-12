package aitools

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/aitools/mcp/agentconfig"
	"github.com/databricks/cli/libs/aitools/mcpconfig"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/spf13/cobra"
)

// Agent entry statuses of `mcp list`. Profile statuses are the drift kinds of mcp.Reconcile, plus ok.
const (
	agentStatusInstalled     = "installed"
	agentStatusMissingOnDisk = "missing_on_disk"
	agentStatusModified      = "modified"
	profileStatusOK          = "ok"
)

// Unmanaged entry kinds of `mcp list`.
const (
	unmanagedKindUGProxy   = "ug-proxy"
	unmanagedKindAIGateway = "ai-gateway-url"
)

// mcpListOutput is the `mcp list` document.
type mcpListOutput struct {
	Profiles  []mcpListProfile   `json:"profiles"`
	Unmanaged []mcpListUnmanaged `json:"unmanaged"`

	// configPath is the Databricks config file, named in the text of a missing profile.
	configPath string
}

type mcpListProfile struct {
	Profile string          `json:"profile"`
	Host    string          `json:"host"`
	Status  string          `json:"status"`
	Servers []mcpListServer `json:"servers"`
}

type mcpListServer struct {
	Name           string         `json:"name"`
	Service        string         `json:"service"`
	SignInRequired bool           `json:"sign_in_required"`
	Agents         []mcpListAgent `json:"agents"`
}

type mcpListAgent struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Status string `json:"status"`
}

type mcpListUnmanaged struct {
	Agent string `json:"agent"`
	Name  string `json:"name"`
	Kind  string `json:"kind"`
}

// NewMcpListCmd returns `aitools mcp list`. It reads the state file and the agent configs only, so it
// needs no network. Drift is reported, not failed: the command exits 0 when a profile is missing.
func NewMcpListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List Databricks managed MCP servers installed by this CLI",
		Long: `List Databricks managed MCP servers installed by this CLI, grouped by profile, with the
coding agents each one is in. Entries that drifted from their profile or from the agent config are
reported. Entries in agent configs that this CLI did not write are listed as unmanaged and never changed.

This command makes no network calls and always exits 0 when entries have drifted.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			out, err := buildMcpList(ctx)
			if err != nil {
				return err
			}
			if installOutputIsJSON(cmd) {
				return renderJSON(cmd.OutOrStdout(), out)
			}
			renderMcpList(ctx, out)
			return nil
		},
	}
}

func buildMcpList(ctx context.Context) (mcpListOutput, error) {
	out := mcpListOutput{Profiles: []mcpListProfile{}, Unmanaged: []mcpListUnmanaged{}}

	state, err := mcp.LoadState(ctx)
	if err != nil {
		return out, err
	}
	profiler := profile.GetProfiler(ctx)
	drifts, err := mcp.Reconcile(ctx, profiler, state)
	if err != nil {
		return out, err
	}
	out.configPath, err = profiler.GetPath(ctx)
	if err != nil {
		return out, err
	}
	disk, err := readAgentEntries(ctx)
	if err != nil {
		return out, err
	}

	// A missing profile is reported before a changed host, since it has no host to compare.
	drift := map[string]mcp.Drift{}
	for _, d := range drifts {
		if prev, ok := drift[d.Entry.Profile]; ok && prev.Kind == mcp.DriftProfileMissing {
			continue
		}
		drift[d.Entry.Profile] = d
	}

	var profileNames []string
	for _, e := range state.Servers {
		if !slices.Contains(profileNames, e.Profile) {
			profileNames = append(profileNames, e.Profile)
		}
	}
	slices.Sort(profileNames)

	for _, name := range profileNames {
		p := mcpListProfile{Profile: name, Status: profileStatusOK, Servers: []mcpListServer{}}
		for _, e := range state.Servers {
			if e.Profile != name {
				continue
			}
			if p.Host == "" {
				p.Host = e.Host
			}
			s := mcpListServer{Name: e.Name, Service: e.Service, SignInRequired: e.SignInRequired, Agents: []mcpListAgent{}}
			for _, agentName := range slices.Sorted(maps.Keys(e.Agents)) {
				s.Agents = append(s.Agents, mcpListAgent{
					Name:   agentName,
					Path:   e.Agents[agentName].Path,
					Status: entryStatus(e, agentName, disk),
				})
			}
			p.Servers = append(p.Servers, s)
		}
		if d, ok := drift[name]; ok {
			p.Status = string(d.Kind)
			if d.Kind == mcp.DriftHostChanged {
				p.Host = d.CurrentHost
			}
		}
		out.Profiles = append(out.Profiles, p)
	}

	for _, a := range agents.MCPAgents() {
		entries := disk[a.Name]
		for _, name := range slices.Sorted(maps.Keys(entries)) {
			kind := unmanagedKind(entries[name])
			if kind == "" || recordedFor(state, a.Name, name) {
				continue
			}
			out.Unmanaged = append(out.Unmanaged, mcpListUnmanaged{Agent: a.Name, Name: name, Kind: kind})
		}
	}
	return out, nil
}

// readAgentEntries returns the MCP server entries of every MCP-capable agent, keyed by agent name then
// server name. A config that cannot be parsed fails the command: the listing would otherwise be wrong.
func readAgentEntries(ctx context.Context) (map[string]map[string]map[string]any, error) {
	disk := map[string]map[string]map[string]any{}
	for _, a := range agents.MCPAgents() {
		entries, err := agentconfig.Read(ctx, a)
		if err != nil {
			return nil, err
		}
		disk[a.Name] = entries
	}
	return disk, nil
}

// entryStatus compares the agent's entry for rec with the agent config on disk.
func entryStatus(rec mcp.Entry, agentName string, disk map[string]map[string]map[string]any) string {
	current, ok := disk[agentName][rec.Name]
	switch {
	case !ok:
		return agentStatusMissingOnDisk
	case isOwnedEntry(current, rec):
		return agentStatusInstalled
	}
	return agentStatusModified
}

// isOwnedEntry reports whether an on-disk entry is the one this CLI wrote for rec: it runs
// `databricks aitools mcp serve` and its command is the recorded one.
func isOwnedEntry(current map[string]any, rec mcp.Entry) bool {
	argv, ok := mcpconfig.Argv(current)
	return ok && mcpconfig.IsDatabricksEntry(current) && argv[0] == rec.Command
}

// unmanagedKind names how an entry reaches managed MCP servers without being written by this CLI, or
// returns "" for any other entry.
func unmanagedKind(entry map[string]any) string {
	if !agentconfig.IsUnmanaged(entry) {
		return ""
	}
	if argv, ok := mcpconfig.Argv(entry); ok && len(argv) > 1 && argv[1] == "mcp-proxy" {
		return unmanagedKindUGProxy
	}
	return unmanagedKindAIGateway
}

// recordedFor reports whether state has a server with this name installed for the agent.
func recordedFor(state *mcp.State, agentName, serverName string) bool {
	return slices.ContainsFunc(state.Servers, func(e mcp.Entry) bool {
		_, ok := e.Agents[agentName]
		return ok && e.Name == serverName
	})
}

// mcpAgentList joins the display names of agents, in the order given.
func mcpAgentList(names []string) string {
	display := make([]string, 0, len(names))
	for _, name := range names {
		display = append(display, agentDisplayName(name))
	}
	return strings.Join(display, ", ")
}

func renderMcpList(ctx context.Context, out mcpListOutput) {
	if len(out.Profiles) == 0 && len(out.Unmanaged) == 0 {
		cmdio.LogString(ctx, "No Databricks MCP servers installed by this CLI.")
		return
	}
	for i, p := range out.Profiles {
		if i > 0 {
			cmdio.LogString(ctx, "")
		}
		cmdio.LogString(ctx, profileHeader(p, out.configPath))
		for _, s := range p.Servers {
			cmdio.LogString(ctx, fmt.Sprintf("  %-24s %-30s %s", s.Name, s.Service, listAgentsText(s.Agents)))
		}
		if hint := profileHint(p); hint != "" {
			cmdio.LogString(ctx, "  "+hint)
		}
	}
	if len(out.Unmanaged) == 0 {
		return
	}
	if len(out.Profiles) > 0 {
		cmdio.LogString(ctx, "")
	}
	cmdio.LogString(ctx, "Unmanaged Databricks MCP entries (not changed by this command):")
	for _, u := range out.Unmanaged {
		cmdio.LogString(ctx, fmt.Sprintf("  %-14s %-20s %s", agentDisplayName(u.Agent), u.Name, unmanagedKindText(u.Kind)))
	}
}

func profileHeader(p mcpListProfile, configPath string) string {
	switch mcp.DriftKind(p.Status) {
	case mcp.DriftProfileMissing:
		return fmt.Sprintf("Profile %s  (profile not found in %s)", p.Profile, configPath)
	case mcp.DriftHostChanged:
		return fmt.Sprintf("Profile %s  (host changed, now %s)", p.Profile, p.Host)
	}
	return fmt.Sprintf("Profile %s  (%s)", p.Profile, p.Host)
}

// profileHint tells the user how to repair a profile whose servers no longer work.
func profileHint(p mcpListProfile) string {
	switch mcp.DriftKind(p.Status) {
	case mcp.DriftProfileMissing:
		return fmt.Sprintf("broken: run `databricks aitools mcp remove --profile %s` or `databricks auth login --host %s --profile %s`", p.Profile, p.Host, p.Profile)
	case mcp.DriftHostChanged:
		return fmt.Sprintf("broken: run `databricks aitools mcp remove --profile %s` and add the servers again", p.Profile)
	}
	return ""
}

func listAgentsText(agentList []mcpListAgent) string {
	parts := make([]string, 0, len(agentList))
	for _, a := range agentList {
		text := agentDisplayName(a.Name)
		switch a.Status {
		case agentStatusMissingOnDisk:
			text += " (missing on disk)"
		case agentStatusModified:
			text += " (modified)"
		}
		parts = append(parts, text)
	}
	return strings.Join(parts, ", ")
}

func unmanagedKindText(kind string) string {
	if kind == unmanagedKindUGProxy {
		return "ug mcp-proxy"
	}
	return "/ai-gateway/mcp-services/ URL"
}
