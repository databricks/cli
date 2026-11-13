package aitools

import (
	"cmp"
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
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
)

// Agent entry statuses of `mcp list`. Server statuses are the drift kinds of mcp.CheckEntries, plus statusOK and
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
	// Error and ErrorCategory are set when the state cannot be read. The listing is then empty.
	Error         string                      `json:"error,omitempty"`
	ErrorCategory protos.AitoolsErrorCategory `json:"error_category,omitempty"`
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
	Name    string `json:"name"`
	Service string `json:"service"`
	// Host is the host the server was set up for. The profile's host can differ after a change.
	Host           string         `json:"host"`
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
	// fix says how to clear a modified entry, for the text listing.
	fix string
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

--profile lists the servers of that profile only. Unmanaged entries have no profile, so they are not
listed then.

This command makes no network calls. It always exits 0 when entries have drifted or cannot be checked;
an entry or agent config that cannot be checked is a warning.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			profileName := profileFlag(cmd)
			out, err := buildMcpList(ctx, profileName)
			if !installOutputIsJSON(cmd) {
				if err != nil {
					return err
				}
				renderMcpList(ctx, out, profileName)
				return nil
			}
			// The document carries the failure, as add and remove write theirs.
			if err != nil {
				out.Error = err.Error()
				out.ErrorCategory = errorCategory(err)
			}
			if renderErr := renderJSON(cmd.OutOrStdout(), out); renderErr != nil {
				return errors.Join(err, renderErr)
			}
			if err != nil {
				return root.ErrAlreadyPrinted
			}
			return nil
		},
	}
}

// buildMcpList reads the state and the agent configs and returns the listing. profileName, when set, lists the
// servers of that profile only.
func buildMcpList(ctx context.Context, profileName string) (mcpListOutput, error) {
	out := mcpListOutput{Profiles: []mcpListProfile{}, Unmanaged: []mcpListUnmanaged{}, Warnings: []string{}}

	state, err := mcp.LoadState(ctx)
	if err != nil {
		return out, err
	}
	var entries []mcp.Entry
	for _, e := range state.Servers {
		if profileName == "" || e.Profile == profileName {
			entries = append(entries, e)
		}
	}
	// Without --profile, the configs of every registry agent are read. The agents that the listed servers are recorded
	// for are read too, so that a record for an agent outside the registry is reported as unknown, not as a moved config.
	var names []string
	if profileName == "" {
		names = mcpAgentNames()
	}
	for _, e := range entries {
		for _, name := range slices.Sorted(maps.Keys(e.Agents)) {
			if !slices.Contains(names, name) {
				names = append(names, name)
			}
		}
	}
	files := newAgentFiles()
	for _, name := range names {
		a := agents.ByName(name)
		if a == nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf("could not read the %s config: %s is not a known agent", name, name))
			continue
		}
		if !a.SupportsMCP() {
			// Its records are warned about with their servers, since the agent has no config to read.
			continue
		}
		if cfg := files.current(ctx, a); cfg.err != nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf("could not read the %s config: %s", agentDisplayName(name), cfg.err))
		}
	}

	checks := mcp.CheckEntries(ctx, &mcp.State{Servers: entries})
	groups := map[profileKey]*mcpListProfile{}
	// A config file that cannot be read is warned about once, for every server that reads it.
	warnedFiles := map[string]bool{}
	for i, e := range entries {
		check := checks[i]
		path := check.Path
		if path == "" {
			// The file cannot be resolved, so the entry is grouped under its file as written.
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
		s := mcpListServer{Name: e.Name, Service: e.Service, Host: e.Host, SignInRequired: e.SignInRequired, Status: statusOK, Agents: []mcpListAgent{}, entry: e}
		switch {
		case check.Err != nil:
			s.Status = statusUnknown
			p.Status = worseStatus(p.Status, statusUnknown)
			if check.Path == "" {
				out.Warnings = append(out.Warnings, fmt.Sprintf("MCP server %s: %s", e.Name, check.Err))
			} else if !warnedFiles[check.Path] {
				warnedFiles[check.Path] = true
				out.Warnings = append(out.Warnings, fmt.Sprintf("%s: %s", filepath.ToSlash(check.Path), check.Err))
			}
		case check.Drift != nil:
			s.Status = string(check.Drift.Kind)
			s.drift = check.Drift
			p.Status = worseStatus(p.Status, s.Status)
			p.noHost = p.noHost || check.Drift.NoHost
			if check.Drift.Kind == mcp.DriftHostChanged {
				p.Host = check.Drift.CurrentHost
			}
		}
		for _, agentName := range slices.Sorted(maps.Keys(e.Agents)) {
			// An agent whose config cannot be read is warned about once, for the agent; here its entries are unknown.
			a := agents.ByName(agentName)
			status, path, fix := statusUnknown, e.Agents[agentName].Path, ""
			switch {
			case a == nil:
			case !a.SupportsMCP():
				out.Warnings = append(out.Warnings, fmt.Sprintf("MCP server %s: %s does not take MCP servers; remove drops this record", e.Name, agentDisplayName(agentName)))
			default:
				var err error
				status, path, fix, err = agentStatus(ctx, files, e, a, checkedBinding(check))
				// An error the entry check or the agent's config already reported is not warned about again.
				if err != nil && !errors.Is(err, check.Err) && !errors.Is(err, files.current(ctx, a).err) {
					out.Warnings = append(out.Warnings, fmt.Sprintf("MCP server %s: could not check %s: %s", e.Name, agentDisplayName(agentName), err))
				}
			}
			s.Agents = append(s.Agents, mcpListAgent{
				Name:   agentName,
				Path:   filepath.ToSlash(path),
				Status: status,
				fix:    fix,
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

	// Unmanaged entries have no profile, so a --profile listing has none.
	if profileName != "" {
		return out, nil
	}
	for _, a := range agents.MCPAgents() {
		cfg := files.current(ctx, a)
		if cfg.err != nil {
			continue
		}
		for _, entryName := range slices.Sorted(maps.Keys(cfg.entries)) {
			kind := unmanagedKind(cfg.entries[entryName])
			if kind == "" || recordedFor(state, a.Name, entryName) {
				continue
			}
			out.Unmanaged = append(out.Unmanaged, mcpListUnmanaged{Agent: a.Name, Name: entryName, Kind: kind})
		}
	}
	return out, nil
}

// worseRank orders the statuses of a profile group. A server that could not be checked makes its profile unknown,
// which is worse than ok and less serious than a drift.
var worseRank = map[string]int{statusOK: 0, statusUnknown: 1, string(mcp.DriftHostChanged): 2, string(mcp.DriftProfileMissing): 3}

// worseStatus returns the more serious of two server statuses, for the status of a profile.
func worseStatus(a, b string) string {
	if worseRank[b] > worseRank[a] {
		return b
	}
	return a
}

// agentStatus compares the server's entry for agent a with the entry that remove acts on (see locateEntry). It returns
// the status, the path of the file the entry is in, and how to clear a modified entry. A recorded entry is path_changed
// when the agent now reads another file: the record names the file the entry was written to, which the agent no longer
// reads. pin decides ownership. err is set when the entry cannot be located or checked.
func agentStatus(ctx context.Context, files *agentFiles, rec mcp.Entry, a *agents.Agent, pin bindingFile) (status, path, fix string, err error) {
	loc, err := files.locateEntry(ctx, rec, a)
	if err != nil {
		return statusUnknown, rec.Agents[a.Name].Path, "", err
	}
	if !loc.found {
		return agentStatusMissingOnDisk, loc.path, "", nil
	}
	if loc.recorded {
		current, err := files.currentPath(ctx, a)
		if err != nil {
			return statusUnknown, loc.path, "", err
		}
		if !mcp.SamePath(loc.path, current) {
			return agentStatusPathChanged, loc.path, "", nil
		}
	}
	owned, err := pin.owns(ctx, rec, loc.entry)
	if err != nil {
		return statusUnknown, loc.path, "", err
	}
	if !owned {
		return agentStatusModified, loc.path, changedEntryFix(rec.Name, loc.path, loc.entry), nil
	}
	return agentStatusInstalled, loc.path, "", nil
}

// unmanagedKind names how an entry reaches managed MCP servers without being recorded for its agent, or returns ""
// for any other entry. An entry that runs `aitools mcp serve` and is not recorded has no install record.
func unmanagedKind(entry map[string]any) string {
	switch {
	case mcpconfig.IsDatabricksEntry(entry):
		return unmanagedKindNoRecord
	case agentconfig.IsUGProxy(entry):
		return unmanagedKindUGProxy
	case agentconfig.IsUnmanaged(entry):
		return unmanagedKindAIGateway
	}
	return ""
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

// renderMcpList writes the text listing: the profiles, the unmanaged entries, then the warnings. profileName is the
// --profile value, which only changes the message for an empty listing.
func renderMcpList(ctx context.Context, out mcpListOutput, profileName string) {
	if len(out.Profiles) == 0 && len(out.Unmanaged) == 0 {
		if profileName != "" {
			cmdio.LogString(ctx, fmt.Sprintf("No Databricks MCP servers installed by this CLI on profile %s.", profileName))
		} else {
			cmdio.LogString(ctx, "No Databricks MCP servers installed by this CLI.")
		}
	}
	for i, p := range out.Profiles {
		if i > 0 {
			cmdio.LogString(ctx, "")
		}
		cmdio.LogString(ctx, profileHeader(p))
		for _, s := range p.Servers {
			cmdio.LogString(ctx, fmt.Sprintf("  %-24s %-30s %s", s.Name, s.Service, listAgentsText(s.Agents)))
			if hint := serverHint(ctx, p, s); hint != "" {
				cmdio.LogString(ctx, "  "+hint)
			}
			for _, a := range s.Agents {
				if a.Status == agentStatusModified {
					cmdio.LogString(ctx, fmt.Sprintf("  changed outside this CLI in %s: %s", agentDisplayName(a.Name), a.fix))
				}
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

// profileHeader names a profile group: its host, or the config file it was not found in when the profile is missing.
// A pinned group also names its config file, so that groups of one profile and host are told apart.
func profileHeader(p mcpListProfile) string {
	if p.Status == string(mcp.DriftProfileMissing) {
		problem := "profile not found in"
		if p.noHost {
			problem = "profile has no host in"
		}
		return fmt.Sprintf("Profile %s  (%s %s)", p.Profile, problem, filepath.ToSlash(p.configPath))
	}
	if p.ConfigFile != "" {
		return fmt.Sprintf("Profile %s  (%s, config file %s)", p.Profile, p.Host, p.ConfigFile)
	}
	return fmt.Sprintf("Profile %s  (%s)", p.Profile, p.Host)
}

// serverHint tells the user how to repair a server whose profile no longer serves it. The remove command names the
// server, so that the other servers of the profile are kept.
func serverHint(ctx context.Context, p mcpListProfile, s mcpListServer) string {
	if s.Status == statusUnknown {
		return "unchecked: see the warning below"
	}
	if s.drift == nil {
		return ""
	}
	remove := mcp.RemoveNamesCommand(s.Name)
	if s.drift.Kind == mcp.DriftProfileMissing {
		return fmt.Sprintf("broken: run `%s` or `%s`", remove, mcp.RecreateCommand(s.entry.Host, p.Profile, loginFile(ctx, p, s.entry)))
	}
	return fmt.Sprintf("broken: %s; run `%s` and add it again", mcp.DriftMessage("it", s.entry, p.Profile, *s.drift), remove)
}

// loginFile returns the config file that a login hint for a server with a missing profile must name, or "" when the
// login's default file is that file. A pinned server logs in to the file it is pinned to. An unpinned server is checked
// against ~/.databrickscfg, and login writes to the file DATABRICKS_CONFIG_FILE names, so the hint names the checked
// file when the environment names another one, or one that cannot be resolved.
func loginFile(ctx context.Context, p mcpListProfile, entry mcp.Entry) string {
	if entry.ConfigFile != "" {
		return filepath.ToSlash(p.configPath)
	}
	environment := env.Get(ctx, mcpconfig.ConfigFileEnv)
	if environment == "" {
		return ""
	}
	path, err := mcp.ResolveConfigFile(ctx, environment)
	if err == nil && mcp.SamePath(path, p.configPath) {
		return ""
	}
	return filepath.ToSlash(p.configPath)
}

// listAgentsText lists the agents of a server, with the status of each entry that needs a word.
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

// unmanagedKindText describes an unmanaged entry kind for the text listing.
func unmanagedKindText(kind string) string {
	switch kind {
	case unmanagedKindUGProxy:
		return "ug mcp-proxy"
	case unmanagedKindNoRecord:
		return "databricks aitools mcp serve, no install record"
	}
	return "/ai-gateway/mcp-services/ URL"
}
