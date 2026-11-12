package aitools

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/aitools/mcp/agentconfig"
	"github.com/databricks/cli/libs/aitools/mcpconfig"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

// Agent entry statuses of `mcp list`. Server statuses are the drift kinds of mcp.Reconcile, plus statusOK and
// statusUnknown, which is an entry or config that could not be checked.
const (
	agentStatusInstalled     = "installed"
	agentStatusMissingOnDisk = "missing_on_disk"
	agentStatusModified      = "modified"
	agentStatusPathChanged   = "path_changed"
	statusUnknown            = "unknown"
	statusOK                 = "ok"
)

// Unmanaged entry kinds of `mcp list`.
const (
	unmanagedKindUGProxy   = "ug-proxy"
	unmanagedKindAIGateway = "ai-gateway-url"
	unmanagedKindNoRecord  = "no-record"
)

// mcpListOutput is the `mcp list` document.
type mcpListOutput struct {
	Profiles  []mcpListProfile   `json:"profiles"`
	Unmanaged []mcpListUnmanaged `json:"unmanaged"`
	// Warnings name the entries and agent configs that could not be checked. The rest is still listed.
	Warnings []string `json:"warnings"`
}

// mcpListProfile groups the servers that bind to one profile in one config file.
type mcpListProfile struct {
	Profile string `json:"profile"`
	Host    string `json:"host"`
	// ConfigFile is the config file the servers are pinned to. It is set only when a server is pinned.
	ConfigFile string          `json:"config_file,omitempty"`
	Status     string          `json:"status"`
	Servers    []mcpListServer `json:"servers"`

	// configPath is the config file the profile is read from, resolved. The text names it when the profile is missing.
	configPath string
	// noHost is set when Status is profile_missing because the profile's section has no host.
	noHost bool
}

type mcpListServer struct {
	Name           string         `json:"name"`
	Service        string         `json:"service"`
	SignInRequired bool           `json:"sign_in_required"`
	Status         string         `json:"status"`
	Agents         []mcpListAgent `json:"agents"`

	// entry is the state record, and drift the drift of the server from its profile, when it has one.
	entry mcp.Entry
	drift *mcp.Drift
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

// profileKey groups the servers of one profile name in one resolved config file.
type profileKey struct {
	profile string
	path    string
}

// NewMcpListCmd returns `aitools mcp list`. It reads the state file and the agent configs only, so it
// needs no network. Drift and unreadable entries are reported, not failed: the command exits 0.
func NewMcpListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List Databricks managed MCP servers installed by this CLI",
		Long: `List Databricks managed MCP servers installed by this CLI, grouped by profile, with the
coding agents each one is in. Entries that drifted from their profile or from the agent config are
reported. Entries in agent configs that this CLI did not write are listed as unmanaged and never changed.
Entries that run 'aitools mcp serve' but have no install record are listed as unmanaged too.

This command makes no network calls. It always exits 0 when entries have drifted or cannot be checked;
an entry or agent config that cannot be checked is a warning.`,
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
	out := mcpListOutput{Profiles: []mcpListProfile{}, Unmanaged: []mcpListUnmanaged{}, Warnings: []string{}}

	state, err := mcp.LoadState(ctx)
	if err != nil {
		return out, err
	}
	// The configs of the agents named in the state are read too, so that a record for an agent outside the
	// registry is reported as unknown, not as a moved config.
	names := mcpAgentNames()
	for _, e := range state.Servers {
		for name := range e.Agents {
			if !slices.Contains(names, name) {
				names = append(names, name)
			}
		}
	}
	configs := readAgentConfigs(ctx, names)
	for _, name := range names {
		if cfg := configs[name]; cfg.err != nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf("could not read the %s config: %s", agentDisplayName(name), cfg.err))
		}
	}

	groups := map[profileKey]*mcpListProfile{}
	for _, e := range state.Servers {
		path, err := mcp.ResolveConfigFile(ctx, e.ConfigFile)
		if err != nil {
			// The entry cannot be checked, so it is grouped under its file as written.
			path = e.ConfigFile
		}
		key := profileKey{profile: e.Profile, path: path}
		p, ok := groups[key]
		if !ok {
			p = &mcpListProfile{Profile: e.Profile, Host: e.Host, Status: statusOK, Servers: []mcpListServer{}, configPath: path}
			groups[key] = p
		}
		if e.ConfigFile != "" {
			p.ConfigFile = filepath.ToSlash(path)
		}
		s := mcpListServer{Name: e.Name, Service: e.Service, SignInRequired: e.SignInRequired, Status: statusOK, Agents: []mcpListAgent{}, entry: e}
		drift, err := checkServer(ctx, e)
		switch {
		case err != nil:
			s.Status = statusUnknown
			out.Warnings = append(out.Warnings, err.Error())
		case drift != nil:
			s.Status = string(drift.Kind)
			s.drift = drift
			p.Status = worseStatus(p.Status, s.Status)
			p.noHost = p.noHost || drift.NoHost
			if drift.Kind == mcp.DriftHostChanged {
				p.Host = drift.CurrentHost
			}
		}
		for _, agentName := range slices.Sorted(maps.Keys(e.Agents)) {
			// A config that cannot be read is warned about once, for the agent; here its entries are unknown.
			status := statusUnknown
			if cfg := configs[agentName]; cfg.err == nil {
				var err error
				status, err = agentStatus(ctx, e, agentName, cfg)
				if err != nil {
					out.Warnings = append(out.Warnings, fmt.Sprintf("MCP server %s: could not check %s: %s", e.Name, agentDisplayName(agentName), err))
				}
			}
			s.Agents = append(s.Agents, mcpListAgent{
				Name:   agentName,
				Path:   filepath.ToSlash(e.Agents[agentName].Path),
				Status: status,
			})
		}
		p.Servers = append(p.Servers, s)
	}

	keys := slices.Collect(maps.Keys(groups))
	slices.SortFunc(keys, func(a, b profileKey) int {
		return cmp.Or(strings.Compare(a.profile, b.profile), strings.Compare(a.path, b.path))
	})
	for _, key := range keys {
		out.Profiles = append(out.Profiles, *groups[key])
	}

	for _, name := range mcpAgentNames() {
		cfg := configs[name]
		if cfg.err != nil {
			continue
		}
		for _, entryName := range slices.Sorted(maps.Keys(cfg.entries)) {
			kind := unmanagedKind(cfg.entries[entryName])
			if kind == "" || recordedFor(state, name, entryName) {
				continue
			}
			out.Unmanaged = append(out.Unmanaged, mcpListUnmanaged{Agent: name, Name: entryName, Kind: kind})
		}
	}
	return out, nil
}

// checkServer returns the drift of one state entry from its profile, or nil when the profile still names the
// recorded host and workspace. err is set when the entry cannot be checked.
func checkServer(ctx context.Context, e mcp.Entry) (*mcp.Drift, error) {
	drifts, err := mcp.Reconcile(ctx, &mcp.State{Servers: []mcp.Entry{e}})
	if err != nil || len(drifts) == 0 {
		return nil, err
	}
	return &drifts[0], nil
}

// worseStatus returns the more serious of two server statuses, for the status of a profile.
func worseStatus(a, b string) string {
	rank := map[string]int{statusOK: 0, string(mcp.DriftHostChanged): 1, string(mcp.DriftProfileMissing): 2}
	if rank[b] > rank[a] {
		return b
	}
	return a
}

// agentConfig is one agent's MCP config, read once for a command. err is set when the config cannot be located or
// parsed; entries is nil then.
type agentConfig struct {
	agent   *agents.Agent
	path    string
	entries map[string]map[string]any
	err     error
}

// readAgentConfigs reads the MCP configs of the named agents, keyed by name. A config that cannot be read is an error
// of its agent, so the command can report it and go on with the others.
func readAgentConfigs(ctx context.Context, names []string) map[string]agentConfig {
	configs := make(map[string]agentConfig, len(names))
	for _, name := range names {
		configs[name] = readAgentConfig(ctx, name)
	}
	return configs
}

func readAgentConfig(ctx context.Context, name string) agentConfig {
	a := agents.ByName(name)
	if a == nil {
		return agentConfig{err: fmt.Errorf("%s is not a known agent", name)}
	}
	path, err := agentconfig.Path(ctx, a)
	if err != nil {
		return agentConfig{agent: a, err: err}
	}
	entries, err := agentconfig.Read(ctx, a)
	if err != nil {
		return agentConfig{agent: a, path: path, err: err}
	}
	return agentConfig{agent: a, path: path, entries: entries}
}

// agentStatus compares the server's entry for one agent with that agent's config, which must have been read. An
// agent whose config moved is path_changed: the entry is in the file the record names, which this command does
// not read. err is set when the entry cannot be checked for ownership.
func agentStatus(ctx context.Context, rec mcp.Entry, agentName string, cfg agentConfig) (string, error) {
	if !mcp.SamePath(rec.Agents[agentName].Path, cfg.path) {
		return agentStatusPathChanged, nil
	}
	current, ok := cfg.entries[rec.Name]
	if !ok {
		return agentStatusMissingOnDisk, nil
	}
	owned, err := ownedBy(ctx, rec, current)
	if err != nil {
		return statusUnknown, err
	}
	if !owned {
		return agentStatusModified, nil
	}
	return agentStatusInstalled, nil
}

// ownedBy reports whether the entry on disk is the one this CLI wrote for rec: a `databricks aitools mcp serve`
// for the record's service and profile, through the record's config file. The command is not compared, because
// an add from another binary replaces the record's command for every agent. err is set when the record's config
// file cannot be resolved.
func ownedBy(ctx context.Context, rec mcp.Entry, current map[string]any) (bool, error) {
	configFile, err := mcp.ResolveConfigFile(ctx, rec.ConfigFile)
	if err != nil {
		return false, err
	}
	return mcp.EntryServes(ctx, current, rec.Service, rec.Profile, configFile)
}

// unmanagedKind names how an entry reaches managed MCP servers without being recorded for its agent, or returns ""
// for any other entry. An entry that runs `aitools mcp serve` and is not recorded has no install record.
func unmanagedKind(entry map[string]any) string {
	switch {
	case mcpconfig.IsDatabricksEntry(entry):
		return unmanagedKindNoRecord
	case !agentconfig.IsUnmanaged(entry):
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
	}
	for i, p := range out.Profiles {
		if i > 0 {
			cmdio.LogString(ctx, "")
		}
		cmdio.LogString(ctx, profileHeader(p))
		for _, s := range p.Servers {
			cmdio.LogString(ctx, fmt.Sprintf("  %-24s %-30s %s", s.Name, s.Service, listAgentsText(s.Agents)))
			if hint := serverHint(p, s); hint != "" {
				cmdio.LogString(ctx, "  "+hint)
			}
		}
	}
	if len(out.Unmanaged) > 0 {
		if len(out.Profiles) > 0 {
			cmdio.LogString(ctx, "")
		}
		cmdio.LogString(ctx, "Unmanaged Databricks MCP entries (not changed by this command):")
		for _, u := range out.Unmanaged {
			cmdio.LogString(ctx, fmt.Sprintf("  %-14s %-20s %s", agentDisplayName(u.Agent), u.Name, unmanagedKindText(u.Kind)))
		}
	}
	for _, w := range out.Warnings {
		cmdio.LogString(ctx, "Warning: "+w)
	}
}

func profileHeader(p mcpListProfile) string {
	if p.Status == string(mcp.DriftProfileMissing) {
		problem := "profile not found in"
		if p.noHost {
			problem = "profile has no host in"
		}
		return fmt.Sprintf("Profile %s  (%s %s)", p.Profile, problem, filepath.ToSlash(p.configPath))
	}
	return fmt.Sprintf("Profile %s  (%s)", p.Profile, p.Host)
}

// serverHint tells the user how to repair a server whose profile no longer serves it. The remove command names the
// server, so that the other servers of the profile are kept.
func serverHint(p mcpListProfile, s mcpListServer) string {
	if s.Status == statusUnknown {
		return "unchecked: see the warning below"
	}
	if s.drift == nil {
		return ""
	}
	remove := mcp.RemoveNamesCommand(s.Name)
	if s.drift.Kind == mcp.DriftProfileMissing {
		pinned := ""
		if s.entry.ConfigFile != "" {
			pinned = p.configPath
		}
		return fmt.Sprintf("broken: run `%s` or `%s`", remove, mcp.RecreateCommand(s.entry.Host, p.Profile, pinned))
	}
	if mcp.NormalizeHost(s.entry.Host) != mcp.NormalizeHost(s.drift.CurrentHost) {
		return fmt.Sprintf("broken: it was set up for %s, but profile %s now points at %s; run `%s` and add it again",
			s.entry.Host, p.Profile, s.drift.CurrentHost, remove)
	}
	return fmt.Sprintf("broken: it was set up for workspace %s, but profile %s now points at workspace %s; run `%s` and add it again",
		s.entry.WorkspaceID, p.Profile, s.drift.CurrentWorkspaceID, remove)
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
		case agentStatusPathChanged:
			text += " (path changed)"
		case statusUnknown:
			text += " (unknown)"
		}
		parts = append(parts, text)
	}
	return strings.Join(parts, ", ")
}

func unmanagedKindText(kind string) string {
	switch kind {
	case unmanagedKindUGProxy:
		return "ug mcp-proxy"
	case unmanagedKindNoRecord:
		return "databricks aitools mcp serve, no install record"
	}
	return "/ai-gateway/mcp-services/ URL"
}
