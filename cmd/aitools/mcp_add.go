package aitools

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/aitools/mcp/agentconfig"
	"github.com/databricks/cli/libs/aitools/mcpconfig"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/shellquote"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

// Per-agent results of `mcp add`. installed means the agent's entry was added or changed; reasserted
// means the file already held the same entry.
const (
	statusInstalled  = "installed"
	statusReasserted = "reasserted"
	statusFailed     = "failed"
)

// cursorAgent is the registry name of Cursor, which needs a server approved before use.
const cursorAgent = "cursor"

type mcpAddFlags struct {
	names    string
	location string
	agents   string
	name     string
	force    bool
}

// addOutput is the `mcp add --output json` document.
type addOutput struct {
	Profile       string                      `json:"profile"`
	Host          string                      `json:"host"`
	Scope         string                      `json:"scope"`
	Servers       []addServerOutput           `json:"servers"`
	ErrorCategory protos.AitoolsErrorCategory `json:"error_category"`
	Error         string                      `json:"error,omitempty"`
}

type addServerOutput struct {
	Name           string           `json:"name"`
	Service        string           `json:"service"`
	SignInRequired bool             `json:"sign_in_required"`
	Agents         []addAgentOutput `json:"agents"`
}

type addAgentOutput struct {
	Name          string                      `json:"name"`
	Status        string                      `json:"status"`
	Path          string                      `json:"path,omitempty"`
	ErrorCategory protos.AitoolsErrorCategory `json:"error_category,omitempty"`
	Message       string                      `json:"message,omitempty"`

	// err is the failure of a failed agent, for the exit status. It is not in the JSON.
	err error
}

// addPlan is one service to install for the profile. entry is the state record it updates,
// nil for a new install.
type addPlan struct {
	service mcp.Service
	name    string
	entry   *mcp.Entry
}

// addRequest is what one `mcp add` run applies to every server.
type addRequest struct {
	// jsonMode suppresses the text output; the JSON document is the only output.
	jsonMode    bool
	profile     string
	host        string
	workspaceID string
	location    string
	agents      []*agents.Agent
	explicit    bool
	force       bool
	binary      string
	// configFile is the config file pinned into the entries, "" when DATABRICKS_CONFIG_FILE is unset.
	configFile string
	// env is the entry env: DATABRICKS_CONFIG_FILE when it was set.
	env map[string]string
}

// NewMcpAddCmd returns `aitools mcp add`.
func NewMcpAddCmd() *cobra.Command {
	var f mcpAddFlags
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Install Databricks managed MCP servers into coding agents",
		Long: `Install Databricks managed MCP servers into coding agents, bound to a Databricks profile.
Each server runs through 'databricks aitools mcp serve' with that profile's credentials.

Servers come from --names: full names such as system.ai.github, or leaf names resolved in
--location (default system.ai). Agents come from --agents. Without it, every detected
MCP-capable agent is used.

Interactive selection is not available yet: in a terminal, pass both --names and --agents.
With --output json, both are required.

Supported agents: ` + strings.Join(mcpAgentNames(), ", "),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			jsonMode := installOutputIsJSON(cmd)
			if jsonMode {
				// JSON mode writes only the result document to stdout.
				ctx = cmdio.WithQuiet(ctx)
			}

			out := &addOutput{
				Scope:         "global",
				Servers:       []addServerOutput{},
				ErrorCategory: protos.AitoolsErrorCategoryUnspecified,
			}
			runErr := runMcpAdd(ctx, cmd, f, jsonMode, out)
			if !jsonMode {
				return runErr
			}

			// The document carries a top-level failure; per-agent failures are already in its agents.
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

	cmd.Flags().StringVar(&f.names, "names", "", "Services to install: full names (system.ai.github) or leaf names (github), comma-separated")
	cmd.Flags().StringVar(&f.location, "location", "", "Schema to resolve leaf names in, as <catalog>.<schema> (default system.ai)")
	cmd.Flags().StringVar(&f.agents, "agents", "", "Agents to install for, comma-separated (default: every detected MCP-capable agent)")
	cmd.Flags().StringVar(&f.name, "name", "", "Server name to use instead of the generated one; only with exactly one service")
	cmd.Flags().BoolVar(&f.force, "force", false, "Overwrite an existing server with the same name that this CLI did not write")
	return cmd
}

// runMcpAdd installs the requested servers and fills out. Every refusal that does not need the
// workspace comes before the first request: the flags, the agents, the binary name, and the binding of
// each service that a recorded entry already holds. A failure of an agent named with --agents is
// returned as an agentErrors; detected agents only report theirs.
func runMcpAdd(ctx context.Context, cmd *cobra.Command, f mcpAddFlags, jsonMode bool, out *addOutput) error {
	names := splitAndTrim(f.names)
	agentNames := splitAndTrim(f.agents)
	if err := validateMCPAddFlags(ctx, f, names, agentNames, jsonMode); err != nil {
		return err
	}
	location := f.location
	if location == "" {
		location = defaultMCPLocation
	}
	if err := validateLocation(location); err != nil {
		return err
	}
	fullNames, err := serviceNames(location, names)
	if err != nil {
		return err
	}
	targets, explicit, err := targetAgents(ctx, agentNames)
	if err != nil {
		return err
	}
	binary, absolute := mcpBinaryPath()
	if !mcpconfig.IsDatabricksBinary(binary) {
		return failure(protos.AitoolsErrorCategoryUncategorized, nil, "%s is not the databricks binary; coding agents start MCP servers with databricks or databricks.exe", binary)
	}

	profileName, err := resolveProfile(ctx, profileFlag(cmd))
	if err != nil {
		return err
	}
	out.Profile = profileName
	if err := requireProfile(ctx, profileName); err != nil {
		return err
	}

	state, err := mcp.LoadState(ctx)
	if err != nil {
		return err
	}
	for _, name := range fullNames {
		if err := mcp.CheckBinding(ctx, state, name, profileName); err != nil {
			return err
		}
	}

	configFile, pinned, err := addConfigFile(ctx)
	if err != nil {
		return err
	}
	w, err := profileClient(ctx, profileName, configFile)
	if err != nil {
		return err
	}
	out.Host = w.Config.Host
	workspaceID, err := auth.ResolveWorkspaceID(ctx, w)
	if err != nil {
		return fmt.Errorf("cannot read the workspace ID of %s: %w", w.Config.Host, err)
	}

	services, err := mcp.Discover(ctx, w, location)
	if err != nil {
		return describeListError(profileName, err)
	}
	picked, err := pickServices(location, fullNames, services)
	if err != nil {
		return err
	}
	plans, err := planAdds(ctx, f.name, profileName, state, picked, targets)
	if err != nil {
		return err
	}

	req := addRequest{
		jsonMode:    jsonMode,
		profile:     profileName,
		host:        w.Config.Host,
		workspaceID: workspaceID,
		location:    location,
		agents:      targets,
		explicit:    explicit,
		force:       f.force,
		binary:      binary,
		configFile:  pinned,
	}
	if pinned != "" {
		req.env = map[string]string{mcpconfig.ConfigFileEnv: pinned}
	}
	if !absolute && !jsonMode {
		cmdio.LogString(ctx, "Note: databricks is not an absolute path; coding agents must find it on PATH to start the MCP servers.")
	}

	printAddHeader(ctx, req)
	servers := make([]addServerOutput, 0, len(plans))
	for _, plan := range plans {
		servers = append(servers, installServer(ctx, req, plan))
	}
	out.Servers = servers

	// The results are printed even when the state cannot be saved: the agent files were written.
	recordErr := recordInstalls(ctx, req, plans, servers)
	printAddResults(ctx, req, servers)
	if recordErr != nil {
		return recordErr
	}
	return agentFailureError(req, servers)
}

// validateMCPAddFlags rejects flag combinations that cannot run, before anything is read or written.
func validateMCPAddFlags(ctx context.Context, f mcpAddFlags, names, agentNames []string, jsonMode bool) error {
	if f.name != "" {
		if len(names) != 1 {
			return failure(protos.AitoolsErrorCategoryInvalidFlags, nil, "--name needs exactly one service in --names")
		}
		// A name must come out of mcp.Sanitize unchanged apart from case, so that mcp__<name>__<tool>
		// splits the same way as a generated name.
		if mcp.Sanitize(f.name) != strings.ToLower(f.name) {
			return failure(protos.AitoolsErrorCategoryInvalidFlags, nil, "--name %q is not a valid server name: use letters, digits, '-' and '_', with no \"__\" and no leading or trailing '-' or '_'", f.name)
		}
	}

	if jsonMode {
		var missing []string
		if len(names) == 0 {
			missing = append(missing, "--names")
		}
		if len(agentNames) == 0 {
			missing = append(missing, "--agents")
		}
		if len(missing) > 0 {
			return failure(protos.AitoolsErrorCategoryInvalidFlags, nil, "--output json requires %s so the command runs without interactive prompts", strings.Join(missing, " and "))
		}
		return nil
	}
	if cmdio.IsPromptSupported(ctx) {
		if len(names) == 0 || len(agentNames) == 0 {
			return failure(protos.AitoolsErrorCategoryInvalidFlags, nil, "interactive selection is not available yet; pass --names and --agents")
		}
		return nil
	}
	if len(names) == 0 {
		return failure(protos.AitoolsErrorCategoryInvalidFlags, nil, "--names is required so the command runs without interactive prompts")
	}
	return nil
}

func validateLocation(location string) error {
	catalog, schema, ok := strings.Cut(location, ".")
	if !ok || catalog == "" || schema == "" || strings.Contains(schema, ".") {
		return failure(protos.AitoolsErrorCategoryInvalidFlags, nil, "--location must be <catalog>.<schema>, got %q", location)
	}
	return nil
}

// requireProfile fails when the named profile is not in the Databricks config file.
func requireProfile(ctx context.Context, name string) error {
	profiler := profile.GetProfiler(ctx)
	profiles, err := profiler.LoadProfiles(ctx, profile.WithName(name))
	if err != nil && !errors.Is(err, profile.ErrNoConfiguration) {
		return err
	}
	if len(profiles) > 0 {
		return nil
	}
	path, err := profiler.GetPath(ctx)
	if err != nil {
		return err
	}
	return failure(protos.AitoolsErrorCategory(mcp.BindProfileNotFound), nil, "profile %s not found in %s", name, filepath.ToSlash(path))
}

// addConfigFile returns the config file that the server binds to, resolved as Bind resolves it, and the
// file to pin into the agent entries: the same file when DATABRICKS_CONFIG_FILE is set, "" otherwise.
func addConfigFile(ctx context.Context) (path, pinned string, err error) {
	value := env.Get(ctx, mcpconfig.ConfigFileEnv)
	path, err = mcp.ResolveConfigFile(ctx, value)
	if err != nil || value == "" {
		return path, "", err
	}
	return path, path, nil
}

// profileClient returns a workspace client bound to exactly the named profile in configFile. Its loaders
// are the ones Bind uses: the profile and the non-auth environment. The environment's credentials are
// never sent to the profile's host.
func profileClient(ctx context.Context, name, configFile string) (*databricks.WorkspaceClient, error) {
	cfg := &config.Config{
		Profile:    name,
		ConfigFile: configFile,
		Loaders:    []config.Loader{databrickscfg.ResolveNonAuthFromEnv, config.ConfigFile},
	}
	w, err := databricks.NewWorkspaceClient((*databricks.Config)(cfg))
	if err != nil {
		// Configuration that the SDK rejects is not a credential failure, so a login would not fix it.
		return nil, err
	}
	// Authenticate only writes headers onto req, which is never sent.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "", nil)
	if err != nil {
		return nil, err
	}
	if err := w.Config.Authenticate(req); err != nil {
		return nil, mcp.AuthError(name, err)
	}
	return w, nil
}

// describeListError names the profile that made a failed listing call. The
// ListError text already carries the server's message, so it is only prefixed. The cause
// keeps the *mcp.ListError for errors.As.
func describeListError(profileName string, err error) error {
	listErr, ok := errors.AsType[*mcp.ListError](err)
	if !ok {
		return err
	}
	category := protos.AitoolsErrorCategoryUncategorized
	if listErr.Kind == mcp.ListErrRateLimited {
		category = protos.AitoolsErrorCategoryRateLimited
	}
	return failure(category, listErr, "profile %s: %s", profileName, listErr.Error())
}

// targetAgents returns the agents named by --agents, or else every detected MCP-capable agent.
// explicit reports whether the names were given.
func targetAgents(ctx context.Context, names []string) ([]*agents.Agent, bool, error) {
	if len(names) > 0 {
		list, err := resolveMCPAgents(names)
		return list, true, err
	}
	var detected []*agents.Agent
	for _, a := range agents.MCPAgents() {
		installed, err := mcpInstalled(ctx, a)
		// An error is kept, so the agent reports it when its config is written.
		if installed || err != nil {
			detected = append(detected, a)
		}
	}
	if len(detected) == 0 {
		return nil, false, fmt.Errorf("no MCP-capable coding agents detected; pass --agents with one of: %s", strings.Join(mcpAgentNames(), ", "))
	}
	return detected, false, nil
}

// resolveMCPAgents maps --agents names to MCP-capable agents. Unknown and non-MCP agents are errors.
func resolveMCPAgents(names []string) ([]*agents.Agent, error) {
	supported := strings.Join(mcpAgentNames(), ", ")
	var result []*agents.Agent
	for _, name := range names {
		a := agents.ByName(name)
		switch {
		case a == nil:
			return nil, failure(protos.AitoolsErrorCategoryUnknownAgent, nil, "unknown agent %q. Supported: %s", name, supported)
		case !a.SupportsMCP():
			return nil, failure(protos.AitoolsErrorCategoryUnknownAgent, nil, "%s does not support MCP servers. Supported: %s", name, supported)
		}
		if !slices.Contains(result, a) {
			result = append(result, a)
		}
	}
	return result, nil
}

func mcpAgentNames() []string {
	var names []string
	for _, a := range agents.MCPAgents() {
		names = append(names, a.Name)
	}
	return names
}

// serviceNames returns the full name of each service in --names. A leaf name is resolved in location. A
// full name must be in location, so its catalog and schema must match it. Any other dotted name is refused,
// since it cannot be either.
func serviceNames(location string, names []string) ([]string, error) {
	full := make([]string, 0, len(names))
	for _, name := range names {
		switch parts := strings.Split(name, "."); len(parts) {
		case 1:
			full = append(full, location+"."+name)
		case 3:
			if !strings.EqualFold(parts[0]+"."+parts[1], location) {
				return nil, failure(protos.AitoolsErrorCategoryInvalidFlags, nil, "%s is not in %s; pass --location to match its schema", name, location)
			}
			full = append(full, name)
		default:
			return nil, failure(protos.AitoolsErrorCategoryInvalidFlags, nil, "%s is not a service name: use a leaf such as github or a full name such as system.ai.github", name)
		}
	}
	return full, nil
}

// pickServices maps the full names of --names to the discovered services of location. Unity Catalog names
// are case-insensitive, so the comparison is too.
func pickServices(location string, names []string, services []mcp.Service) ([]mcp.Service, error) {
	var picked []mcp.Service
	for _, name := range names {
		i := slices.IndexFunc(services, func(s mcp.Service) bool { return strings.EqualFold(s.FullName, name) })
		if i < 0 {
			available := make([]string, 0, len(services))
			for _, s := range services {
				available = append(available, s.FullName)
			}
			return nil, failure(categoryServiceNotFound, nil, "%s is not an MCP service in %s. Available: %s", name, location, strings.Join(available, ", "))
		}
		if !slices.ContainsFunc(picked, func(s mcp.Service) bool { return s.FullName == services[i].FullName }) {
			picked = append(picked, services[i])
		}
	}
	return picked, nil
}

// planAdds chooses the server name of each picked service before anything is written. A service
// already installed for this profile keeps its name. A new name avoids names of other state
// entries and of unmanaged entries in the target agents. The entries have been checked against the
// profile by mcp.CheckBinding.
func planAdds(ctx context.Context, nameFlag, profileName string, state *mcp.State, picked []mcp.Service, targets []*agents.Agent) ([]addPlan, error) {
	taken := map[string]bool{}
	for i := range state.Servers {
		taken[state.Servers[i].Name] = true
	}
	for _, a := range targets {
		for name := range unmanagedNames(ctx, a) {
			taken[name] = true
		}
	}

	plans := make([]addPlan, 0, len(picked))
	for _, svc := range picked {
		entry := stateEntry(state, svc.FullName, profileName)
		var name string
		switch {
		case entry != nil && nameFlag != "" && nameFlag != entry.Name:
			return nil, failure(protos.AitoolsErrorCategoryInvalidFlags, nil, "%s is already installed on profile %s as %s; remove it before using --name %s", svc.FullName, profileName, entry.Name, nameFlag)
		case entry != nil:
			name = entry.Name
		case nameFlag != "":
			if slices.ContainsFunc(state.Servers, func(e mcp.Entry) bool { return e.Name == nameFlag }) {
				return nil, failure(protos.AitoolsErrorCategoryInvalidFlags, nil, "--name %s is used by another installed server; pick a different name", nameFlag)
			}
			name = nameFlag
		default:
			name = mcp.UniqueName(mcp.ServerName(svc.FullName, profileName), func(n string) bool { return taken[n] })
		}
		taken[name] = true
		plans = append(plans, addPlan{service: svc, name: name, entry: entry})
	}
	return plans, nil
}

// unmanagedNames returns the server names in the agent's config that this CLI did not write. A
// config that cannot be read has none here; Upsert reports that failure for the agent.
func unmanagedNames(ctx context.Context, a *agents.Agent) map[string]bool {
	entries, err := agentconfig.Read(ctx, a)
	if err != nil {
		return nil
	}
	names := make(map[string]bool, len(entries))
	for name, entry := range entries {
		if !mcpconfig.IsDatabricksEntry(entry) {
			names[name] = true
		}
	}
	return names
}

func stateEntry(state *mcp.State, service, profileName string) *mcp.Entry {
	i := slices.IndexFunc(state.Servers, func(e mcp.Entry) bool {
		return e.Service == service && e.Profile == profileName
	})
	if i < 0 {
		return nil
	}
	return &state.Servers[i]
}

// installServer writes one server into every target agent. Each agent's result is reported; one
// agent's failure does not stop the others.
func installServer(ctx context.Context, req addRequest, plan addPlan) addServerOutput {
	out := addServerOutput{
		Name:           plan.name,
		Service:        plan.service.FullName,
		SignInRequired: plan.service.SignInRequired,
		Agents:         make([]addAgentOutput, 0, len(req.agents)),
	}
	for _, a := range req.agents {
		var path string
		var status agentconfig.Status
		var err error
		if req.explicit {
			err = requireInstalled(ctx, a)
		}
		if err == nil {
			args := []string{"aitools", "mcp", "serve", "--profile", req.profile, "--service", plan.service.FullName, "--agent", a.Name}
			path, status, err = agentconfig.Upsert(ctx, a, plan.name, req.binary, args, req.env, req.force)
		}

		result := addAgentOutput{Name: a.Name}
		switch {
		case err != nil:
			result.Status = statusFailed
			result.ErrorCategory = errorCategory(err)
			result.Message = err.Error()
			result.err = err
		case status == agentconfig.StatusUnchanged:
			result.Status = statusReasserted
			result.Path = filepath.ToSlash(path)
		default:
			result.Status = statusInstalled
			result.Path = filepath.ToSlash(path)
		}
		out.Agents = append(out.Agents, result)
	}
	return out
}

// requireInstalled fails when an agent named with --agents has neither its MCP config file nor the
// directory that holds it. Writing the config would create that directory, so the agent is refused.
func requireInstalled(ctx context.Context, a *agents.Agent) error {
	installed, err := mcpInstalled(ctx, a)
	if err != nil {
		return err
	}
	if installed {
		return nil
	}
	dir, err := mcpDir(ctx, a)
	if err != nil {
		return err
	}
	return failure(protos.AitoolsErrorCategoryUnknownAgent, nil, "%s is not installed (no %s); install it first or leave it out of --agents", a.DisplayName, filepath.ToSlash(dir))
}

// mcpInstalled reports whether the agent's MCP config file or the directory that holds it exists.
func mcpInstalled(ctx context.Context, a *agents.Agent) (bool, error) {
	path, err := a.MCP.Path(ctx)
	if err != nil {
		return false, err
	}
	exists, err := pathExists(path)
	if err != nil || exists {
		return exists, err
	}
	dir, err := mcpDir(ctx, a)
	if err != nil {
		return false, err
	}
	return pathExists(dir)
}

// mcpDir returns the directory that holds the agent's MCP config file. Claude Code keeps that file in the
// home directory, which always exists, so its config directory stands for it instead.
func mcpDir(ctx context.Context, a *agents.Agent) (string, error) {
	path, err := a.MCP.Path(ctx)
	if err != nil {
		return "", err
	}
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(path)
	if filepath.Clean(dir) == filepath.Clean(home) {
		return a.ConfigDir(ctx)
	}
	return dir, nil
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, fs.ErrNotExist):
		return false, nil
	default:
		return false, err
	}
}

// recordInstalls adds each server that reached at least one agent to the state file. The file is
// re-read first, so entries written since planning are kept.
func recordInstalls(ctx context.Context, req addRequest, plans []addPlan, servers []addServerOutput) error {
	state, err := mcp.LoadState(ctx)
	if err != nil {
		return err
	}
	recorded := false
	for i, server := range servers {
		paths := map[string]string{}
		for _, a := range server.Agents {
			if a.Status != statusFailed {
				paths[a.Name] = a.Path
			}
		}
		if len(paths) == 0 {
			continue
		}
		recordServer(state, req, plans[i], paths)
		recorded = true
	}
	if !recorded {
		return nil
	}
	return mcp.SaveState(ctx, state)
}

// recordServer stores the install of plan's server for req.profile, with the agents that succeeded
// and their config paths.
func recordServer(state *mcp.State, req addRequest, plan addPlan, paths map[string]string) {
	entry := stateEntry(state, plan.service.FullName, req.profile)
	if entry == nil {
		state.Servers = append(state.Servers, mcp.Entry{
			Name:        plan.name,
			Service:     plan.service.FullName,
			Profile:     req.profile,
			InstalledAt: time.Now().UTC(),
		})
		entry = &state.Servers[len(state.Servers)-1]
	}
	entry.Host = req.host
	entry.WorkspaceID = req.workspaceID
	entry.SignInRequired = plan.service.SignInRequired
	entry.Command = req.binary
	entry.ConfigFile = req.configFile
	entry.CLIVersion = build.GetInfo().Version
	if entry.Agents == nil {
		// A hand-edited state file can hold "agents": null.
		entry.Agents = map[string]mcp.AgentRecord{}
	}
	for agent, path := range paths {
		entry.Agents[agent] = mcp.AgentRecord{Path: path, Scope: "user"}
	}
}

// agentFailureError returns the agents' failures as an agentErrors. They fail the command when an agent was
// named with --agents, or when no agent was installed at all. Otherwise detected agents only warn, as in
// install, and the results already show them.
func agentFailureError(req addRequest, servers []addServerOutput) error {
	var errs []error
	installed := false
	for _, server := range servers {
		for _, a := range server.Agents {
			if a.err != nil {
				errs = append(errs, a.err)
			} else {
				installed = true
			}
		}
	}
	if len(errs) == 0 || (!req.explicit && installed) {
		return nil
	}
	return &agentErrors{err: errors.Join(errs...)}
}

func printAddHeader(ctx context.Context, req addRequest) {
	if req.jsonMode {
		return
	}
	cmdio.LogString(ctx, fmt.Sprintf("Profile %s (%s)", req.profile, req.host))
	names := make([]string, 0, len(req.agents))
	for _, a := range req.agents {
		names = append(names, a.Name)
	}
	source := "named with --agents"
	if !req.explicit {
		source = "detected"
	}
	cmdio.LogString(ctx, fmt.Sprintf("Agents (%s): %s", source, strings.Join(names, ", ")))
}

// printAddResults prints each server, its agents, and the follow-up steps the server needs.
func printAddResults(ctx context.Context, req addRequest, servers []addServerOutput) {
	if req.jsonMode {
		return
	}
	for _, server := range servers {
		cmdio.LogString(ctx, "")
		cmdio.LogString(ctx, server.Name+"  "+server.Service)
		succeeded := false
		cursor := false
		for _, a := range server.Agents {
			cmdio.LogString(ctx, "  "+agentResultLine(ctx, a))
			if a.Status != statusFailed {
				succeeded = true
			}
			if a.Name == cursorAgent && a.Status == statusInstalled {
				cursor = true
			}
		}
		if server.SignInRequired && succeeded {
			cmdio.LogString(ctx, fmt.Sprintf("%s needs sign-in. Run: databricks aitools mcp login %s --profile %s", server.Service, shellquote.BashArg(server.Service), shellquote.BashArg(req.profile)))
		}
		if cursor {
			cmdio.LogString(ctx, cursorApprovalHint(server.Name))
		}
	}
}

func agentResultLine(ctx context.Context, a addAgentOutput) string {
	switch a.Status {
	case statusFailed:
		return cmdio.Yellow(ctx, fmt.Sprintf("%-12s failed: %s", a.Name, a.Message))
	case statusReasserted:
		return fmt.Sprintf("%-12s already installed, re-asserted  %s", a.Name, a.Path)
	default:
		return fmt.Sprintf("%-12s installed  %s", a.Name, a.Path)
	}
}

// cursorApprovalHint explains the approval step Cursor asks for on a new server. The Cursor docs
// (https://cursor.com/docs/cli/mcp) call the CLI "agent" and list "agent mcp enable <identifier>";
// the installed binary is cursor-agent, so the hint names that.
func cursorApprovalHint(name string) string {
	return fmt.Sprintf("Cursor may ask you to approve %s. Run: cursor-agent mcp enable %s (the Cursor docs call the CLI agent), or approve it in Cursor's MCP settings.", name, name)
}
