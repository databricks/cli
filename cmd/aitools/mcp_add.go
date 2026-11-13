package aitools

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/aitools/mcp/agentconfig"
	"github.com/databricks/cli/libs/aitools/mcpconfig"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/shellquote"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/spf13/cobra"
)

// Per-agent results of `mcp add`. installed means the agent's entry was added or changed; reasserted
// means the file already held the same entry.
const (
	statusInstalled  = "installed"
	statusReasserted = "reasserted"
	statusFailed     = "failed"
)

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

// addPlan is one service to install for the profile. failed holds the agents that cannot take the server, with the
// reason; they are not written.
type addPlan struct {
	service mcp.Service
	name    string
	failed  map[*agents.Agent]error
}

// addRequest is what one `mcp add` run applies to every server.
type addRequest struct {
	// jsonMode suppresses the text output; the JSON document is the only output.
	jsonMode    bool
	profile     string
	host        string
	workspaceID string
	agents      []*agents.Agent
	explicit    bool
	force       bool
	binary      string
	// configFile is the resolved config file that the servers bind to.
	configFile string
	// pinned is the config file pinned into the entries and the state, "" when DATABRICKS_CONFIG_FILE is unset.
	pinned string
	// env is the entry env: DATABRICKS_CONFIG_FILE when it was set.
	env map[string]string
	// refused holds the agents named with --agents that are not installed, with the reason. Their
	// config is not written.
	refused map[*agents.Agent]error
}

// slashPaths rewrites the agent config paths of the document to forward slashes, as the text output
// shows them. The state keeps the native paths.
func (o *addOutput) slashPaths() {
	for i := range o.Servers {
		for j := range o.Servers[i].Agents {
			agent := &o.Servers[i].Agents[j]
			agent.Path = filepath.ToSlash(agent.Path)
		}
	}
}

// NewMcpAddCmd returns `aitools mcp add`.
func NewMcpAddCmd() *cobra.Command {
	var f mcpAddFlags
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Install Databricks managed MCP servers into coding agents",
		Long: `Install Databricks managed MCP servers into coding agents, bound to a Databricks profile.
Each server runs through 'databricks aitools mcp serve' with that profile's credentials.

Servers come from --names: full names such as system.ai.github, or leaf names such as github. Leaf names
resolve in --location, which defaults to system.ai, or to the one schema that the full names in --names share.
Agents come from --agents. Without it, every detected MCP-capable agent is used.

Interactive selection is not available yet: in a terminal, pass both --names and --agents.
With --output json, both are required.

Supported agents: ` + strings.Join(mcpAgentNames(), ", "),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			jsonMode := installOutputIsJSON(cmd)

			out := &addOutput{
				Scope:         installer.ScopeGlobal,
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
			out.slashPaths()
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
	cmd.Flags().StringVar(&f.location, "location", "", "Schema to resolve leaf names in, as <catalog>.<schema> (default: system.ai, or the one schema that the full names in --names share)")
	cmd.Flags().StringVar(&f.agents, "agents", "", "Agents to install for, comma-separated (default: every detected MCP-capable agent)")
	cmd.Flags().StringVar(&f.name, "name", "", "Server name to use instead of the generated one; only with exactly one service")
	cmd.Flags().BoolVar(&f.force, "force", false, "Overwrite an existing server with the same name that this CLI did not write")
	return cmd
}

// runMcpAdd installs the requested servers and fills out. Every refusal that needs no request comes before
// the first one: the flags, the agents, the binary name, the binding of each service that a recorded entry
// holds, and the server names. A server that no agent can take is refused the same way, unless a detected agent
// can take it. A failure of an agent named with --agents is returned as an agentErrors; detected agents only report
// theirs. A server that reaches no agent fails the run.
func runMcpAdd(ctx context.Context, cmd *cobra.Command, f mcpAddFlags, jsonMode bool, out *addOutput) error {
	names := splitAndTrim(f.names)
	agentNames := splitAndTrim(f.agents)
	if err := validateMCPAddFlags(ctx, f, names, agentNames, jsonMode); err != nil {
		return err
	}
	location, fullNames, err := resolveServices(f.location, names)
	if err != nil {
		return err
	}
	if f.name != "" && len(fullNames) != 1 {
		return failure(protos.AitoolsErrorCategoryInvalidFlags, nil, "--name needs exactly one service in --names")
	}
	targets, explicit, err := targetAgents(ctx, agentNames)
	if err != nil {
		return err
	}
	binary, absolute := mcpBinaryPath()
	if !mcpconfig.IsDatabricksBinary(binary) {
		return failure(protos.AitoolsErrorCategoryUncategorized, nil, "%s is not the databricks binary; coding agents start MCP servers with databricks or databricks.exe", filepath.ToSlash(binary))
	}
	var refused map[*agents.Agent]error
	if explicit {
		refused = refusedAgents(ctx, targets)
		// Nothing can be written when every agent is refused, so the refusals end the run before any request.
		if len(refused) == len(targets) {
			var errs []error
			for _, a := range targets {
				errs = append(errs, refused[a])
			}
			return errors.Join(errs...)
		}
	}

	// The config file is resolved once, as Bind resolves it. The profile is read from that same file, so the
	// profile lookups and the bindings below all name one file.
	configFile, pinned, err := addConfigFile(ctx)
	if err != nil {
		return err
	}
	ctx = env.Set(ctx, mcpconfig.ConfigFileEnv, configFile)
	profileName, err := resolveProfile(ctx, profileFlag(cmd))
	if err != nil {
		return err
	}
	out.Profile = profileName

	state, err := mcp.LoadState(ctx)
	if err != nil {
		return err
	}
	for _, name := range fullNames {
		if _, err := mcp.CheckBinding(ctx, state, name, profileName, configFile); err != nil {
			return err
		}
	}
	in := planInput{
		binding:  addBinding{profile: profileName, configFile: configFile},
		nameFlag: f.name,
		force:    f.force,
		explicit: explicit,
		targets:  targets,
		held:     readAgentEntries(ctx, targets),
	}
	plans, err := planAdds(ctx, in, state, fullNames)
	if err != nil {
		return err
	}
	if err := planningRefusals(plans, targets); err != nil {
		return err
	}

	w, err := profileClient(ctx, profileName, configFile)
	if err != nil {
		return err
	}
	out.Host = w.Config.Host
	workspaceID, err := auth.ResolveWorkspaceID(ctx, w)
	if err != nil {
		return describeRequestError(profileName, fmt.Errorf("cannot read the workspace ID of %s: %w", w.Config.Host, err))
	}

	services, err := mcp.Discover(ctx, w, location)
	if err != nil {
		return describeRequestError(profileName, err)
	}
	picked, err := pickServices(location, fullNames, services)
	if err != nil {
		return err
	}
	for i := range plans {
		plans[i].service = picked[i]
	}

	req := addRequest{
		jsonMode:    jsonMode,
		profile:     profileName,
		host:        w.Config.Host,
		workspaceID: workspaceID,
		agents:      targets,
		explicit:    explicit,
		force:       f.force,
		binary:      binary,
		configFile:  configFile,
		pinned:      pinned,
		refused:     refused,
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
		// A name must come out of mcp.Sanitize unchanged, so that mcp__<name>__<tool> splits the same way as
		// a generated name.
		if mcp.Sanitize(f.name) != f.name {
			return failure(protos.AitoolsErrorCategoryInvalidFlags, nil, "--name %q is not a valid server name: use lowercase letters, digits, '-' and '_', with no \"__\" and no leading or trailing '-' or '_'", f.name)
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

// profileClient returns a workspace client bound to exactly the named profile in configFile. Its config is the one
// Bind builds, so the environment's credentials are never sent to the profile's host.
func profileClient(ctx context.Context, name, configFile string) (*databricks.WorkspaceClient, error) {
	cfg, err := mcp.BoundConfig(name, configFile)
	if err != nil {
		return nil, err
	}
	w, err := databricks.NewWorkspaceClient((*databricks.Config)(cfg))
	if err != nil {
		return nil, err
	}
	if err := mcp.AuthenticateBound(ctx, cfg, name); err != nil {
		return nil, err
	}
	return w, nil
}

// describeRequestError classifies a failed request to the profile's workspace. A credential the workspace
// rejects is AUTH_FAILED, with the login command for the profile. Other failures are described by
// describeListError.
func describeRequestError(profileName string, err error) error {
	if errors.Is(err, apierr.ErrUnauthenticated) {
		return mcp.AuthError(profileName, err)
	}
	return describeListError(profileName, err)
}

// describeListError names the profile that made a failed listing call. The
// ListError text already carries the server's message, so it is only prefixed. The cause
// keeps the *mcp.ListError for errors.As.
func describeListError(profileName string, err error) error {
	listErr, ok := errors.AsType[*mcp.ListError](err)
	if !ok {
		return err
	}
	var category protos.AitoolsErrorCategory
	switch listErr.Kind {
	case mcp.ListErrLocationNotFound:
		category = categoryLocationNotFound
	case mcp.ListErrPermissionDenied:
		category = categoryPermissionDenied
	case mcp.ListErrRateLimited:
		category = protos.AitoolsErrorCategoryRateLimited
	default:
		category = protos.AitoolsErrorCategoryUncategorized
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
		installed, err := a.InstalledForMCP(ctx)
		// An error is kept, so the agent reports it when its config is written.
		if installed || err != nil {
			detected = append(detected, a)
		}
	}
	if len(detected) == 0 {
		return nil, false, failure(protos.AitoolsErrorCategoryUnknownAgent, nil, "no MCP-capable coding agents detected; pass --agents with one of: %s", strings.Join(mcpAgentNames(), ", "))
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

// resolveServices returns the schema that leaf names resolve in and the full name of each distinct service
// that names pick. The names are lowercase, as the API returns them.
func resolveServices(location string, names []string) (string, []string, error) {
	location, err := locationOf(location, names)
	if err != nil {
		return "", nil, err
	}
	if err := validateLocation(location); err != nil {
		return "", nil, err
	}
	full, err := serviceNames(location, names)
	if err != nil {
		return "", nil, err
	}
	return location, full, nil
}

// locationOf returns the schema that leaf names resolve in, and that full names must be in. An explicit --location
// is used in lowercase. Without one, full names that all name one schema give that schema, and otherwise the default
// is system.ai. Leaf names resolve in that schema too, so a leaf with full names outside system.ai is refused rather
// than resolved in a schema the user did not name.
func locationOf(explicit string, names []string) (string, error) {
	if explicit != "" {
		return strings.ToLower(explicit), nil
	}
	var schemas []string
	leaves := false
	for _, name := range names {
		parts := strings.Split(strings.ToLower(name), ".")
		switch {
		case len(parts) == 1:
			leaves = true
		case isFullName(parts):
			if schema := parts[0] + "." + parts[1]; !slices.Contains(schemas, schema) {
				schemas = append(schemas, schema)
			}
		}
	}
	switch {
	case len(schemas) == 0:
		return mcp.SystemAILocation, nil
	case len(schemas) > 1:
		return "", failure(protos.AitoolsErrorCategoryInvalidFlags, nil, "--names are in different schemas (%s); run add once for each schema", strings.Join(schemas, ", "))
	case leaves && schemas[0] != mcp.SystemAILocation:
		return "", failure(protos.AitoolsErrorCategoryInvalidFlags, nil, "leaf names resolve in %s, but the full names are in %s; pass --location %s to resolve the leaves there, or use full names only", mcp.SystemAILocation, schemas[0], schemas[0])
	}
	return schemas[0], nil
}

// serviceNames returns the full name of each distinct service in --names. A leaf name is resolved in location. A
// full name must be in location, so its catalog and schema must match it. Any other dotted name is refused,
// since it cannot be either. The names are returned in lowercase, as the API returns them.
func serviceNames(location string, names []string) ([]string, error) {
	full := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.ToLower(name)
		var fullName string
		switch parts := strings.Split(name, "."); {
		case len(parts) == 1:
			fullName = location + "." + name
		case isFullName(parts):
			if parts[0]+"."+parts[1] != location {
				return nil, failure(protos.AitoolsErrorCategoryInvalidFlags, nil, "%s is not in %s; pass --location to match its schema", name, location)
			}
			fullName = name
		default:
			return nil, failure(protos.AitoolsErrorCategoryInvalidFlags, nil, "%s is not a service name: use a leaf such as github or a full name such as system.ai.github", name)
		}
		if !slices.Contains(full, fullName) {
			full = append(full, fullName)
		}
	}
	return full, nil
}

// isFullName reports whether parts are the catalog, schema and name of a full service name, each one non-empty.
func isFullName(parts []string) bool {
	return len(parts) == 3 && parts[0] != "" && parts[1] != "" && parts[2] != ""
}

// pickServices returns the discovered service of each full name, in the same order. Unity Catalog names are
// case-insensitive, so the comparison is too.
func pickServices(location string, names []string, services []mcp.Service) ([]mcp.Service, error) {
	picked := make([]mcp.Service, 0, len(names))
	for _, name := range names {
		i := slices.IndexFunc(services, func(s mcp.Service) bool { return strings.EqualFold(s.FullName, name) })
		if i < 0 {
			if len(services) == 0 {
				return nil, failure(categoryServiceNotFound, nil, "%s is not an MCP service in %s, which has none", name, location)
			}
			available := make([]string, 0, len(services))
			for _, s := range services {
				available = append(available, s.FullName)
			}
			return nil, failure(categoryServiceNotFound, nil, "%s is not an MCP service in %s. Available: %s", name, location, strings.Join(available, ", "))
		}
		picked = append(picked, services[i])
	}
	return picked, nil
}

// addBinding is what the servers of one run bind to: the profile, and the config file it is read from.
type addBinding struct {
	profile    string
	configFile string
}

// agentEntries holds the server entries of each target agent, as they were read once before planning. An agent whose
// config cannot be read has none here: its write reports the failure.
type agentEntries map[*agents.Agent]map[string]map[string]any

func readAgentEntries(ctx context.Context, targets []*agents.Agent) agentEntries {
	held := make(agentEntries, len(targets))
	for _, a := range targets {
		entries, err := agentconfig.Read(ctx, a)
		if err != nil {
			continue
		}
		held[a] = entries
	}
	return held
}

// planInput is what planAdds plans the servers of one run against.
type planInput struct {
	binding  addBinding
	nameFlag string
	force    bool
	// explicit is set when the agents were named with --agents. A name conflict in a named agent refuses the run. A
	// conflict in a detected agent fails only that agent for that server.
	explicit bool
	targets  []*agents.Agent
	held     agentEntries
}

// planAdds chooses the server name of each service before anything is written, and refuses what would overwrite
// another server. A service already installed for the binding keeps its name, and so does a name that an agent entry
// holds for the same binding when the state has none. A new name avoids the names of other state entries and of every
// agent entry, so that a name this CLI wrote for another profile is never overwritten.
func planAdds(ctx context.Context, in planInput, state *mcp.State, fullNames []string) ([]addPlan, error) {
	taken := map[string]bool{}
	for i := range state.Servers {
		taken[state.Servers[i].Name] = true
	}

	plans := make([]addPlan, 0, len(fullNames))
	for _, fullName := range fullNames {
		entry, err := mcp.FindEntry(ctx, state, fullName, in.binding.profile, in.binding.configFile)
		if err != nil {
			return nil, err
		}
		name, failed, err := chooseName(ctx, in, state, fullName, entry, taken)
		if err != nil {
			return nil, err
		}
		taken[name] = true
		plans = append(plans, addPlan{service: mcp.Service{FullName: fullName}, name: name, failed: failed})
	}
	return plans, nil
}

// chooseName returns the server name for fullName, and the target agents that cannot take it. entry is the server's
// state entry, nil for a new install. The name it holds is kept. Otherwise --name is used when no other server holds
// it, and the name is generated.
func chooseName(ctx context.Context, in planInput, state *mcp.State, fullName string, entry *mcp.Entry, taken map[string]bool) (string, map[*agents.Agent]error, error) {
	b := in.binding
	stored := ""
	if entry != nil {
		stored = entry.Name
	} else {
		owned, err := ownedNames(ctx, in.held, in.targets, fullName, b)
		if err != nil {
			return "", nil, err
		}
		switch len(owned) {
		case 0:
		case 1:
			stored = owned[0]
		default:
			return "", nil, failure(categoryNameConflict, nil, "%s is installed on profile %s under more than one name in the agents (%s); remove the extra entries", fullName, b.profile, strings.Join(owned, ", "))
		}
	}

	if stored != "" {
		if in.nameFlag != "" && in.nameFlag != stored {
			return "", nil, failure(protos.AitoolsErrorCategoryInvalidFlags, nil, "%s is already installed on profile %s as %s; remove it before using --name %s", fullName, b.profile, stored, in.nameFlag)
		}
		if stateHolds(state, stored, entry) {
			return "", nil, failure(categoryNameConflict, nil, "%s is installed as %s, but another server in the state holds %s; remove that entry first", fullName, stored, stored)
		}
		failed, err := nameFailures(ctx, in, fullName, stored, true)
		return stored, failed, err
	}

	if in.nameFlag != "" {
		if stateHolds(state, in.nameFlag, nil) {
			return "", nil, failure(protos.AitoolsErrorCategoryInvalidFlags, nil, "--name %s is used by another installed server; pick a different name", in.nameFlag)
		}
		failed, err := nameFailures(ctx, in, fullName, in.nameFlag, false)
		return in.nameFlag, failed, err
	}

	base := mcp.ServerName(fullName, b.profile)
	if base == "" {
		return "", nil, failure(protos.AitoolsErrorCategoryInvalidFlags, nil, "cannot make a server name from %s for profile %s; pass --name", fullName, b.profile)
	}
	// No entry of this binding exists, so every name that an agent holds belongs to another server.
	reserved := map[string]bool{}
	for _, a := range in.targets {
		for name := range in.held[a] {
			reserved[name] = true
		}
	}
	return mcp.UniqueName(base, func(n string) bool { return taken[n] || reserved[n] }), nil, nil
}

// stateHolds reports whether a state entry other than self holds name. self is nil when no entry is excluded.
func stateHolds(state *mcp.State, name string, self *mcp.Entry) bool {
	for i := range state.Servers {
		if state.Servers[i].Name == name && &state.Servers[i] != self {
			return true
		}
	}
	return false
}

// ownedNames returns the distinct names that the target agents hold for the binding of fullName, sorted.
func ownedNames(ctx context.Context, held agentEntries, targets []*agents.Agent, fullName string, b addBinding) ([]string, error) {
	var names []string
	for _, a := range targets {
		for name, entry := range held[a] {
			serves, err := mcp.EntryServes(ctx, entry, fullName, b.profile, b.configFile)
			if err != nil {
				return nil, err
			}
			if serves && !slices.Contains(names, name) {
				names = append(names, name)
			}
		}
	}
	slices.Sort(names)
	return names, nil
}

// nameFailures returns the target agents that cannot take name for fullName, with the reason for each. stored says
// whether name is the server's stored name. In a named agent a conflict refuses the run, so it is returned as the
// error. In a detected agent it fails that agent for this server only.
func nameFailures(ctx context.Context, in planInput, fullName, name string, stored bool) (map[*agents.Agent]error, error) {
	failed := map[*agents.Agent]error{}
	for _, a := range in.targets {
		conflict, err := holderConflict(ctx, in, a, fullName, name, stored)
		if err != nil {
			return nil, err
		}
		if conflict == nil {
			continue
		}
		if in.explicit {
			return nil, conflict
		}
		failed[a] = conflict
	}
	return failed, nil
}

// holderConflict returns why agent a cannot take name for fullName, or nil when it can. An entry of another binding
// is a conflict. So is an entry this CLI did not write, unless force is set: the agent's write would refuse it.
func holderConflict(ctx context.Context, in planInput, a *agents.Agent, fullName, name string, stored bool) (conflict, err error) {
	entry, ok := in.held[a][name]
	if !ok {
		return nil, nil
	}
	if !mcpconfig.IsDatabricksEntry(entry) {
		if in.force {
			return nil, nil
		}
		return failure(categoryNameConflict, agentconfig.ErrNotOwned, "%s already has a server named %s that this CLI did not write; pass --force to replace it", a.DisplayName, name), nil
	}
	serves, err := mcp.EntryServes(ctx, entry, fullName, in.binding.profile, in.binding.configFile)
	if err != nil {
		return nil, err
	}
	switch {
	case serves:
		return nil, nil
	case stored:
		return failure(categoryNameConflict, nil, "%s is installed as %s, but %s holds %s for another server; remove that entry first", fullName, name, a.Name, name), nil
	default:
		return failure(protos.AitoolsErrorCategoryInvalidFlags, nil, "--name %s is used by another server in %s; pick a different name", name, a.Name), nil
	}
}

// planningRefusals returns the failures of the plans when no agent can take any of the servers, which ends the run
// before any request. It returns nil when some agent can take some server.
func planningRefusals(plans []addPlan, targets []*agents.Agent) error {
	var errs []error
	for _, plan := range plans {
		for _, a := range targets {
			if plan.failed[a] == nil {
				return nil
			}
			errs = appendDistinct(errs, plan.failed[a])
		}
	}
	return errors.Join(errs...)
}

// appendDistinct appends err unless errs already holds that same error value. A failure that repeats for each server,
// such as a refusal, is reported once.
func appendDistinct(errs []error, err error) []error {
	for _, e := range errs {
		if errors.Is(err, e) {
			return errs
		}
	}
	return append(errs, err)
}

// installServer writes one server into every target agent. Each agent's result is reported; one
// agent's failure does not stop the others. An agent in req.refused, or that failed planning for the server, is not written.
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
		err := req.refused[a]
		if err == nil {
			err = plan.failed[a]
		}
		if err == nil {
			args := []string{"aitools", "mcp", "serve", mcpconfig.ProfileFlag, req.profile, mcpconfig.ServiceFlag, plan.service.FullName, mcpconfig.AgentFlag, a.Name}
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
			result.Path = path
		default:
			result.Status = statusInstalled
			result.Path = path
		}
		out.Agents = append(out.Agents, result)
	}
	return out
}

// refusedAgents returns the agents named with --agents that are not installed, with the reason for each.
// It runs once, before any request, so each service reports the same refusal without another check.
func refusedAgents(ctx context.Context, targets []*agents.Agent) map[*agents.Agent]error {
	refused := map[*agents.Agent]error{}
	for _, a := range targets {
		if err := requireInstalled(ctx, a); err != nil {
			refused[a] = err
		}
	}
	return refused
}

// requireInstalled fails when an agent named with --agents is not installed. Writing its config would create
// the agent's directory, so the agent is refused instead.
func requireInstalled(ctx context.Context, a *agents.Agent) error {
	installed, err := a.InstalledForMCP(ctx)
	if err != nil {
		return err
	}
	if installed {
		return nil
	}
	return failure(protos.AitoolsErrorCategoryUnknownAgent, nil, "%s is not installed; install it first or leave it out of --agents", a.DisplayName)
}

// recordInstalls adds each server that reached at least one agent to the state file. The file is
// re-read first, so entries written since planning are kept. Each server is recorded on its own, so that one
// failure does not keep the others from being saved: their agent files are already written.
func recordInstalls(ctx context.Context, req addRequest, plans []addPlan, servers []addServerOutput) error {
	state, err := mcp.LoadState(ctx)
	if err != nil {
		return err
	}
	var errs []error
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
		if err := recordServer(ctx, state, req, plans[i], paths); err != nil {
			errs = append(errs, err)
			continue
		}
		recorded = true
	}
	if recorded {
		if err := mcp.SaveState(ctx, state); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// recordServer stores the install of plan's server for req.profile in configFile, with the agents that succeeded
// and their config paths.
func recordServer(ctx context.Context, state *mcp.State, req addRequest, plan addPlan, paths map[string]string) error {
	entry, err := mcp.FindEntry(ctx, state, plan.service.FullName, req.profile, req.configFile)
	if err != nil {
		return err
	}
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
	entry.ConfigFile = req.pinned
	entry.CLIVersion = build.GetInfo().Version
	if entry.Agents == nil {
		// A hand-edited state file can hold "agents": null.
		entry.Agents = map[string]mcp.AgentRecord{}
	}
	for agent, path := range paths {
		entry.Agents[agent] = mcp.AgentRecord{Path: path, Scope: agentScopeUser}
	}
	return nil
}

// agentFailureError returns the agents' failures as an agentErrors. The command fails when an agent was named
// with --agents, or when a requested service reached no agent at all. Otherwise detected agents only warn, as
// in install, and the results already show them. Every distinct failure is kept; one that repeats for each
// service is reported once.
func agentFailureError(req addRequest, servers []addServerOutput) error {
	var errs []error
	fail := req.explicit
	for _, server := range servers {
		reached := false
		for _, a := range server.Agents {
			if a.err == nil {
				reached = true
				continue
			}
			errs = appendDistinct(errs, a.err)
		}
		if !reached {
			fail = true
		}
	}
	if len(errs) == 0 || !fail {
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
			if a.Name == agents.NameCursor && a.Status == statusInstalled {
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
	path := filepath.ToSlash(a.Path)
	switch a.Status {
	case statusFailed:
		return cmdio.Yellow(ctx, fmt.Sprintf("%-12s failed: %s", a.Name, a.Message))
	case statusReasserted:
		return fmt.Sprintf("%-12s already installed, re-asserted  %s", a.Name, path)
	default:
		return fmt.Sprintf("%-12s installed  %s", a.Name, path)
	}
}

// cursorApprovalHint explains the approval step Cursor asks for on a new server. The Cursor docs
// (https://cursor.com/docs/cli/mcp) call the CLI "agent" and list "agent mcp enable <identifier>";
// the installed binary is cursor-agent, so the hint names that.
func cursorApprovalHint(name string) string {
	return fmt.Sprintf("Cursor may ask you to approve %s. Run: cursor-agent mcp enable %s (the Cursor docs call the CLI agent), or approve it in Cursor's MCP settings.", name, name)
}
