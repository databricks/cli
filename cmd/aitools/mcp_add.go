package aitools

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
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
	"github.com/databricks/cli/libs/browser"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

// Per-agent results of `mcp add`. installed means the agent's entry was added or changed; reasserted
// means the file already held the same entry.
const (
	statusInstalled  = "installed"
	statusReasserted = "reasserted"
	statusFailed     = "failed"
)

// serverNameRules is the rule that a server name must meet, in the messages that refuse a name.
const serverNameRules = "use lowercase letters, digits, '-' and '_', with no \"__\" and no leading or trailing '-' or '_'"

type mcpAddFlags struct {
	names    string
	location string
	agents   string
	name     string
	force    bool
	login    bool
	noLogin  bool
}

// addOutput is the `mcp add --output json` document.
type addOutput struct {
	Profile       string                      `json:"profile"`
	Host          string                      `json:"host"`
	Scope         string                      `json:"scope"`
	Servers       []addServerOutput           `json:"servers"`
	ErrorCategory protos.AitoolsErrorCategory `json:"error_category,omitempty"`
	Error         string                      `json:"error,omitempty"`
}

type addServerOutput struct {
	Name           string           `json:"name"`
	Service        string           `json:"service"`
	SignInRequired bool             `json:"sign_in_required"`
	Agents         []addAgentOutput `json:"agents"`
	// SignIn is the sign-in of the server under --login, with the state that login reports. It is absent without --login.
	SignIn *addSignInOutput `json:"sign_in,omitempty"`

	// notes explain the name and the entries that were replaced. The text output prints them; the JSON does not.
	notes []string
	// note is the service's prerequisite, printed after the agent results. The text output prints it; the JSON does not.
	note string
}

// addSignInOutput is the sign-in of one server under --login: its state, as login reports it, and the failure when the
// sign-in failed.
type addSignInOutput struct {
	State         string                      `json:"state,omitempty"`
	ErrorCategory protos.AitoolsErrorCategory `json:"error_category,omitempty"`
	Error         string                      `json:"error,omitempty"`
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
	// note explains the name when it got a suffix.
	note string
	// reasserted holds the agents whose config already has this server as the write would leave it.
	reasserted map[*agents.Agent]bool
	// updated holds the agents whose config has a Databricks entry for this server under its name that the write changes.
	updated map[*agents.Agent]bool
	// replaced holds the agents whose config has an entry under this name that this CLI did not write. Under --force
	// the write replaces it.
	replaced map[*agents.Agent]bool
}

// agentSource says where the agents of an add came from. It is shown in the header.
type agentSource string

const (
	agentsNamed    agentSource = "named with --agents"
	agentsDetected agentSource = "detected"
	agentsPicked   agentSource = "selected"
)

// addRequest is what one `mcp add` run applies to every server.
type addRequest struct {
	// jsonMode suppresses the text output; the JSON document is the only output.
	jsonMode    bool
	profile     string
	host        string
	workspaceID string
	agents      []*agents.Agent
	agentSource agentSource
	force       bool
	binary      string
	// configFile is the resolved config file that the servers bind to.
	configFile string
	// pinned is the config file pinned into the entries and the state, "" when DATABRICKS_CONFIG_FILE is unset.
	pinned string
	// env is the entry env: DATABRICKS_CONFIG_FILE when it was set.
	env map[string]string
	// refused holds the agents that are not installed or whose config cannot be read, with the reason. Their config is not written.
	refused map[*agents.Agent]error
	// login signs in to the sign-in-required servers after they are installed.
	login bool
	// askLogin asks, per sign-in-required server, whether to sign in after it is installed.
	askLogin bool
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

Services come from --names: full names such as system.ai.github, or leaf names such as github. Leaf names
resolve in --location, which defaults to system.ai, or to the one schema that the full names in --names share when there are no leaf names.
Agents come from --agents.

In a terminal, the command asks for whatever is not given: the profile, the servers and the agents (detected
agents are pre-checked). It confirms the plan when it asked for any of them, and it asks before each sign-in unless
--login or --no-login is given. Without a terminal, --names is required and the agents default to every detected
MCP-capable agent. With --output json, --names and --agents are required.

Supported agents: ` + strings.Join(mcpAgentNames(), ", "),
		Example: `  # Install GitHub and web search for Claude Code and Cursor, bound to the dogfood profile
  databricks aitools mcp add --names github,web_search --agents claude-code,cursor --profile dogfood`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			jsonMode := installOutputIsJSON(cmd)

			out := &addOutput{
				Scope:   installer.ScopeGlobal,
				Servers: []addServerOutput{},
			}
			runErr := runMcpAdd(ctx, cmd, f, jsonMode, out)
			logMcpEvent(ctx, protos.AitoolsMcpEvent{
				CallingAgent: root.CallingAgentType(),
				Operation:    protos.AitoolsMcpOperationAdd,
				Servers:      addServerResults(out.Servers),
			})
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
	cmd.Flags().StringVar(&f.location, "location", "", "Schema to resolve leaf names in, as <catalog>.<schema> (default: system.ai, or the one schema that the full names in --names share when there are no leaf names)")
	cmd.Flags().StringVar(&f.agents, "agents", "", "Agents to install for, comma-separated (default: a picker in a terminal; otherwise every detected MCP-capable agent)")
	cmd.Flags().StringVar(&f.name, "name", "", "Server name to use instead of the generated one; only with exactly one service")
	cmd.Flags().BoolVar(&f.force, "force", false, "Overwrite an existing server with the same name that this CLI did not write")
	cmd.Flags().BoolVar(&f.login, "login", false, "Sign in to each sign-in-required server that was installed, after the installs")
	cmd.Flags().BoolVar(&f.noLogin, "no-login", false, "Do not sign in; print the login commands for sign-in-required servers (default without a terminal; in a terminal, the default asks for each one)")
	return cmd
}

// runMcpAdd installs the requested servers and fills out. Every refusal that needs no request comes before
// the first one: the flags, the agents named with --agents, the binary name, the binding of each service that a
// recorded entry holds, and the server names. A server that no agent can take is refused the same way, unless a
// detected or picked agent can take it. A failure of an agent named with --agents is returned as an agentErrors;
// detected and picked agents only report theirs. A server that reaches no agent fails the run.
//
// Without --names, in a terminal, the services are picked from the workspace, so the workspace is read before the
// plan. The services named with --names are planned first, and the workspace is read after that.
func runMcpAdd(ctx context.Context, cmd *cobra.Command, f mcpAddFlags, jsonMode bool, out *addOutput) error {
	names := splitAndTrim(f.names)
	agentNames := splitAndTrim(f.agents)
	if err := validateMCPAddFlags(ctx, f, names, agentNames, jsonMode); err != nil {
		return err
	}
	interactive := !jsonMode && promptsSupported(ctx)
	location, fullNames, err := resolveServices(f.location, names)
	if err != nil {
		return err
	}
	if f.name != "" && len(fullNames) != 1 {
		return failure(protos.AitoolsErrorCategoryInvalidFlags, "--name needs exactly one service in --names")
	}
	var named []*agents.Agent
	if len(agentNames) > 0 {
		if named, err = resolveMCPAgents(agentNames); err != nil {
			return err
		}
	}
	// Without a terminal the agents are chosen before the binary and the profile are checked, so that a run with no
	// agent reports that first. A terminal chooses them once the profile is known.
	var targets []*agents.Agent
	var source agentSource
	var detectErrs map[*agents.Agent]error
	if !interactive {
		if targets, source, detectErrs, err = chooseAgents(ctx, named, false); err != nil {
			return err
		}
	}
	binary, absolute := mcpBinaryPath()
	if !mcpconfig.IsDatabricksBinary(binary) {
		return failure(protos.AitoolsErrorCategoryUncategorized, "%s is not the databricks binary; coding agents start MCP servers with databricks or databricks.exe", filepath.ToSlash(binary))
	}
	// A named agent that is not installed, and a detected agent whose detection failed, are refused before any request.
	// Nothing can be written when every agent is refused, so the refusals end the run. A terminal that picks its agents
	// checks them once the pick is made.
	refused := map[*agents.Agent]error{}
	if len(named) > 0 {
		refused = refusedAgents(ctx, named)
	}
	maps.Copy(refused, detectErrs)
	checked := targets
	if len(named) > 0 {
		checked = named
	}
	if len(checked) > 0 && len(refused) == len(checked) {
		var errs []error
		for _, a := range checked {
			errs = append(errs, refused[a])
		}
		return withCategory(errors.Join(errs...), errorCategory(errs[0]))
	}

	// The config file is resolved once, as Bind resolves it. The profile is read from that same file, so the
	// profile lookups and the bindings below all name one file.
	configFile, pinned, err := addConfigFile(ctx)
	if err != nil {
		return err
	}
	ctx = env.Set(ctx, mcpconfig.ConfigFileEnv, configFile)
	profileName, profilePicked, err := chooseProfile(ctx, profileFlag(cmd), interactive)
	if err != nil {
		return err
	}
	out.Profile = profileName

	state, err := mcp.LoadState(ctx)
	if err != nil {
		return err
	}
	// Until the services are known, nothing reaches the workspace. A run that picks its services checks the profile here,
	// and the agents are chosen before the first request.
	pickingServices := len(names) == 0
	var w *databricks.WorkspaceClient
	var workspaceID, authorization string
	var picked []mcp.Service
	if pickingServices {
		if err := mcp.CheckProfile(profileName, configFile); err != nil {
			return withExistingProfiles(ctx, state, profileName, err)
		}
	} else if err := checkBindings(ctx, state, fullNames, profileName, configFile); err != nil {
		return withExistingProfiles(ctx, state, profileName, err)
	}
	if interactive {
		if targets, source, _, err = chooseAgents(ctx, named, true); err != nil {
			return err
		}
	}
	held, unreadable := readAgentEntries(ctx, targets)
	maps.Copy(refused, unreadable)
	// Nothing can be written when every agent is refused, so the refusals end the run before any request.
	if len(refused) == len(targets) {
		var errs []error
		for _, a := range targets {
			errs = append(errs, refused[a])
		}
		return withCategory(errors.Join(errs...), errorCategory(errs[0]))
	}
	// Without --names the services are picked from the workspace, and then checked like the named ones.
	if pickingServices {
		w, workspaceID, authorization, err = connectWorkspace(ctx, out, profileName, configFile, pinned)
		if err != nil {
			return err
		}
		if picked, err = chooseServices(ctx, w, addBinding{profile: profileName, configFile: configFile}, pinned, location, state); err != nil {
			return err
		}
		if len(picked) == 0 {
			cmdio.LogString(ctx, "No servers selected.")
			return nil
		}
		fullNames = serviceFullNames(picked)
		if err := checkBindings(ctx, state, fullNames, profileName, configFile); err != nil {
			return err
		}
	}
	in := planInput{
		binding:  addBinding{profile: profileName, configFile: configFile},
		nameFlag: f.name,
		force:    f.force,
		explicit: source == agentsNamed,
		targets:  targets,
		held:     held,
		refused:  refused,
		binary:   binary,
		env:      pinnedEnv(pinned),
	}
	plans, err := planAdds(ctx, in, state, fullNames)
	if err != nil {
		return err
	}
	if err := planningRefusals(plans, targets, refused); err != nil {
		return err
	}
	if !pickingServices {
		w, workspaceID, authorization, err = connectWorkspace(ctx, out, profileName, configFile, pinned)
		if err != nil {
			return err
		}
		services, err := mcp.Discover(ctx, w, location)
		if err != nil {
			return describeRequestError(profileName, pinned, err)
		}
		if picked, err = pickServices(location, fullNames, services); err != nil {
			return err
		}
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
		agentSource: source,
		force:       f.force,
		binary:      binary,
		configFile:  configFile,
		pinned:      pinned,
		refused:     refused,
		login:       f.login,
		askLogin:    interactive && !f.login && !f.noLogin,
		env:         pinnedEnv(pinned),
	}
	if !absolute && !jsonMode {
		cmdio.LogString(ctx, "Note: databricks is not an absolute path; coding agents must find it on PATH to start the MCP servers.")
	}

	printAddHeader(ctx, req)
	// A run that names everything asks no question, except the sign-in question of a sign-in-required server.
	// The plan is confirmed only when a picker made one of the choices.
	if interactive && (profilePicked || pickingServices || source == agentsPicked) {
		printAddPlan(ctx, req, plans)
		proceed, err := promptMCPProceed()
		if err != nil {
			return err
		}
		if !proceed {
			cmdio.LogString(ctx, cancelledMessage)
			return nil
		}
	}
	servers := make([]addServerOutput, 0, len(plans))
	for _, plan := range plans {
		servers = append(servers, installServer(ctx, req, plan))
	}
	out.Servers = servers

	// The results are printed even when the state cannot be saved: the agent files were written. The sign-in does not
	// depend on the state, so it still runs, and every failure is returned.
	recordErr := recordInstalls(ctx, req, plans, servers)
	printAddResults(ctx, req, servers)
	var signErr error
	if req.login || req.askLogin {
		signErr = signInInstalled(ctx, cmd.ErrOrStderr(), w, req, plans, servers)
	}
	setUploadConfig(ctx, profileName, configFile, w.Config, authorization)
	return errors.Join(recordErr, signErr, agentFailureError(req, servers))
}

// serviceFullNames returns the full names of services, in order.
func serviceFullNames(services []mcp.Service) []string {
	names := make([]string, 0, len(services))
	for _, s := range services {
		names = append(names, s.FullName)
	}
	return names
}

// connectWorkspace returns the bound client, workspace ID and Authorization header, and sets the host in out.
func connectWorkspace(ctx context.Context, out *addOutput, profileName, configFile, pinned string) (*databricks.WorkspaceClient, string, string, error) {
	w, authorization, err := profileClient(ctx, profileName, configFile, pinned)
	if err != nil {
		return nil, "", "", err
	}
	out.Host = w.Config.Host
	workspaceID, err := auth.ResolveWorkspaceID(ctx, w)
	if err != nil {
		return nil, "", "", describeRequestError(profileName, pinned, fmt.Errorf("cannot read the workspace ID of %s: %w", w.Config.Host, err))
	}
	return w, workspaceID, authorization, nil
}

// checkBindings returns the first failure of CheckBinding for the services, which makes no request.
func checkBindings(ctx context.Context, state *mcp.State, fullNames []string, profileName, configFile string) error {
	for _, name := range fullNames {
		if _, err := mcp.CheckBinding(ctx, state, name, profileName, configFile); err != nil {
			return err
		}
	}
	return nil
}

// chooseServices returns the services the user checks in the picker, from the services of location. The services
// the profile already has are rows marked installed, unchecked, so that they can be added to another agent.
func chooseServices(ctx context.Context, w *databricks.WorkspaceClient, b addBinding, pinned, location string, state *mcp.State) ([]mcp.Service, error) {
	list := func(ctx context.Context, location string) ([]mcp.Service, error) {
		services, err := mcp.Discover(ctx, w, location)
		if err != nil {
			return nil, describeRequestError(b.profile, pinned, err)
		}
		return services, nil
	}
	return pickServicesInteractively(ctx, b, location, state, list)
}

// validateMCPAddFlags rejects flag combinations that cannot run, before anything is read or written.
func validateMCPAddFlags(ctx context.Context, f mcpAddFlags, names, agentNames []string, jsonMode bool) error {
	// A value that names no agent, such as ",", must not fall back to every detected agent.
	if f.agents != "" && len(agentNames) == 0 {
		return failure(protos.AitoolsErrorCategoryInvalidFlags, "--agents names no agent; pass one or more of: %s", strings.Join(mcpAgentNames(), ", "))
	}
	if f.login && f.noLogin {
		return failure(protos.AitoolsErrorCategoryInvalidFlags, "--login and --no-login cannot be used together")
	}
	if f.name != "" {
		// A name must come out of mcp.Sanitize unchanged, so that mcp__<name>__<tool> splits the same way as
		// a generated name.
		if mcp.Sanitize(f.name) != f.name {
			return failure(protos.AitoolsErrorCategoryInvalidFlags, "--name %q is not a valid server name: "+serverNameRules, f.name)
		}
		// The name is ASCII once it comes out of Sanitize unchanged, so its length in bytes is its length in characters.
		if len(f.name) > mcp.MaxServerNameLen {
			return failure(protos.AitoolsErrorCategoryInvalidFlags, "--name must be at most %d characters, got %d; pick a shorter name", mcp.MaxServerNameLen, len(f.name))
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
			return failure(protos.AitoolsErrorCategoryInvalidFlags, "--output json requires %s so the command runs without interactive prompts", strings.Join(missing, " and "))
		}
		return nil
	}
	if len(names) == 0 && !promptsSupported(ctx) {
		return failure(protos.AitoolsErrorCategoryInvalidFlags, "--names is required so the command runs without interactive prompts")
	}
	// The picker chooses services in a terminal, and --name needs exactly one of them, which only --names can give.
	if f.name != "" && len(names) == 0 {
		return failure(protos.AitoolsErrorCategoryInvalidFlags, "--name needs exactly one service; leave out --name to pick services in the picker")
	}
	return nil
}

// validateLocation checks that location is <catalog>.<schema>.
func validateLocation(location string) error {
	catalog, schema, ok := strings.Cut(location, ".")
	if !ok || catalog == "" || schema == "" || strings.Contains(schema, ".") {
		return failure(protos.AitoolsErrorCategoryInvalidFlags, "--location must be <catalog>.<schema>, got %q", location)
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

// withExistingProfiles adds the workspace profiles that exist to a profile-not-found failure, when no server is
// installed for the profile. A mistyped --profile is the usual cause then. Other failures are returned unchanged, and so
// is the failure of an installed server, whose recreate hint is the one to follow.
func withExistingProfiles(ctx context.Context, state *mcp.State, profileName string, err error) error {
	if be, ok := errors.AsType[*mcp.BindError](err); !ok || be.Category != mcp.BindProfileNotFound || !errors.Is(err, mcp.ErrProfileMissing) {
		return err
	}
	if slices.ContainsFunc(state.Servers, func(e mcp.Entry) bool { return e.Profile == profileName }) {
		return err
	}
	profiles, loadErr := profile.GetProfiler(ctx).LoadProfiles(ctx, mcpWorkspaceProfile)
	if loadErr != nil && !errors.Is(loadErr, profile.ErrNoConfiguration) {
		return err
	}
	if len(profiles) == 0 {
		return err
	}
	return fmt.Errorf("%w\nExisting profiles: %s", err, strings.Join(profiles.Names(), ", "))
}

// profileClient returns a workspace client bound to exactly the named profile
// in configFile, alongside the Authorization header obtained during binding.
func profileClient(ctx context.Context, name, configFile, pinnedFile string) (*databricks.WorkspaceClient, string, error) {
	cfg, err := mcp.BoundConfig(name, configFile)
	if err != nil {
		return nil, "", err
	}
	if err := checkWorkspaceProfile(name, configFile, cfg); err != nil {
		return nil, "", err
	}
	return boundClient(ctx, cfg, name, pinnedFile)
}

// checkWorkspaceProfile refuses an account-only profile, before the credential is sent. The workspace commands refuse it
// after the SDK has resolved its workspace ID, so this checks the resolved config, which includes a workspace given in the
// host or found by discovery.
func checkWorkspaceProfile(name, configFile string, cfg *config.Config) error {
	if cfg.AccountID != "" && (cfg.WorkspaceID == "" || cfg.WorkspaceID == profile.WorkspaceIDNone) {
		return failure(protos.AitoolsErrorCategoryUncategorized, "profile %q is account-level, with no workspace_id in %s; MCP servers need a profile for one workspace", name, filepath.ToSlash(configFile))
	}
	return nil
}

// boundClient returns the workspace client and Authorization header for the bound config.
func boundClient(ctx context.Context, cfg *config.Config, name, pinnedFile string) (*databricks.WorkspaceClient, string, error) {
	w, err := databricks.NewWorkspaceClient((*databricks.Config)(cfg))
	if err != nil {
		return nil, "", err
	}
	authorization, err := mcp.AuthenticateBound(ctx, cfg, name, pinnedFile)
	if err != nil {
		return nil, "", err
	}
	return w, authorization, nil
}

// describeRequestError classifies a failed request to the profile's workspace. A credential the workspace
// rejects is AUTH_FAILED, with the login command for the profile. Other failures are described by
// describeListError.
func describeRequestError(profileName, pinnedFile string, err error) error {
	if errors.Is(err, apierr.ErrUnauthenticated) {
		return mcp.AuthErrorIn(profileName, pinnedFile, err)
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
		category = protos.AitoolsErrorCategoryMCPLocationNotFound
	case mcp.ListErrPermissionDenied:
		category = protos.AitoolsErrorCategoryMCPPermissionDenied
	case mcp.ListErrRateLimited:
		category = protos.AitoolsErrorCategoryRateLimited
	default:
		category = protos.AitoolsErrorCategoryUncategorized
	}
	return withCategory(fmt.Errorf("profile %s: %w", profileName, listErr), category)
}

// chooseAgents returns the agents to install for: the ones named with --agents, else the ones the user checks in the
// picker when the session is interactive, else every detected MCP-capable agent. In a terminal with no agent installed
// for MCP, the run is refused here, with the agents that were not found, before any request. detectErrs holds the agents
// whose detection failed. They are not written, and each reports its failure.
func chooseAgents(ctx context.Context, named []*agents.Agent, interactive bool) ([]*agents.Agent, agentSource, map[*agents.Agent]error, error) {
	if len(named) > 0 {
		return named, agentsNamed, nil, nil
	}
	if interactive {
		choices, notFound := mcpAgentChoices(ctx)
		if len(notFound) > 0 {
			cmdio.LogString(ctx, "Not found: "+strings.Join(notFound, ", "))
		}
		if len(choices) == 0 {
			return nil, "", nil, noAgentsError()
		}
		picked, err := promptMCPAgents(ctx, installer.ScopeGlobal, choices)
		return picked, agentsPicked, nil, err
	}
	var detected []*agents.Agent
	var detectErrs map[*agents.Agent]error
	for _, a := range agents.MCPAgents() {
		installed, err := a.InstalledForMCP(ctx)
		if err != nil {
			// The agent is kept, so that its failure is reported in its result rather than the agent being skipped.
			if detectErrs == nil {
				detectErrs = map[*agents.Agent]error{}
			}
			detectErrs[a] = err
			detected = append(detected, a)
		} else if installed {
			detected = append(detected, a)
		}
	}
	if len(detected) == 0 {
		return nil, "", nil, noAgentsError()
	}
	return detected, agentsDetected, detectErrs, nil
}

// noAgentsError is the refusal of a run that finds no MCP-capable agent to install for.
func noAgentsError() error {
	return failure(protos.AitoolsErrorCategoryUnknownAgent, "no MCP-capable coding agents detected; pass --agents with one of: %s", strings.Join(mcpAgentNames(), ", "))
}

// resolveMCPAgents maps --agents names to MCP-capable agents. Unknown and non-MCP agents are errors.
func resolveMCPAgents(names []string) ([]*agents.Agent, error) {
	supported := strings.Join(mcpAgentNames(), ", ")
	var result []*agents.Agent
	for _, name := range names {
		a := agents.ByName(name)
		switch {
		case a == nil:
			if suggestion := agentSuggestion(name); suggestion != "" {
				return nil, failure(protos.AitoolsErrorCategoryUnknownAgent, "unknown agent %q; did you mean %s? Supported: %s", name, suggestion, supported)
			}
			return nil, failure(protos.AitoolsErrorCategoryUnknownAgent, "unknown agent %q. Supported: %s", name, supported)
		case !a.SupportsMCP():
			return nil, failure(protos.AitoolsErrorCategoryUnknownAgent, "%s does not support MCP servers. Supported: %s", name, supported)
		}
		if !slices.Contains(result, a) {
			result = append(result, a)
		}
	}
	return result, nil
}

// agentSuggestion returns the MCP-capable agent that an unknown name most likely means: one whose binary
// is the name, or whose name starts with it. It returns "" when no agent or several agents match.
func agentSuggestion(name string) string {
	var match string
	for _, a := range agents.MCPAgents() {
		if a.Binary != name && !strings.HasPrefix(a.Name, name) {
			continue
		}
		if match != "" {
			return ""
		}
		match = a.Name
	}
	return match
}

// mcpAgentNames returns the names of the agents that take MCP servers.
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
		return "", failure(protos.AitoolsErrorCategoryInvalidFlags, "--names are in different schemas (%s); run add once for each schema", strings.Join(schemas, ", "))
	case leaves && schemas[0] != mcp.SystemAILocation:
		return "", failure(protos.AitoolsErrorCategoryInvalidFlags, "leaf names resolve in %s, but the full names are in %s; pass --location %s to resolve the leaves there, or use full names only", mcp.SystemAILocation, schemas[0], schemas[0])
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
				return nil, failure(protos.AitoolsErrorCategoryInvalidFlags, "%s is not in %s; pass --location to match its schema", name, location)
			}
			fullName = name
		default:
			return nil, failure(protos.AitoolsErrorCategoryInvalidFlags, "%s is not a service name: use a leaf such as github or a full name such as system.ai.github", name)
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
				return nil, failure(protos.AitoolsErrorCategoryMCPServiceNotFound, "%s is not an MCP service in %s, which has none", name, location)
			}
			available := make([]string, 0, len(services))
			for _, s := range services {
				available = append(available, s.FullName)
			}
			return nil, failure(protos.AitoolsErrorCategoryMCPServiceNotFound, "%s is not an MCP service in %s. Available: %s", name, location, strings.Join(available, ", "))
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

// readAgentEntries returns the server entries of each target agent, and the agents whose config could not be read, with
// the reason.
func readAgentEntries(ctx context.Context, targets []*agents.Agent) (agentEntries, map[*agents.Agent]error) {
	held := make(agentEntries, len(targets))
	unreadable := map[*agents.Agent]error{}
	files := newAgentFiles()
	for _, a := range targets {
		cfg := files.current(ctx, a)
		if cfg.err != nil {
			unreadable[a] = cfg.err
			continue
		}
		held[a] = cfg.entries
	}
	return held, unreadable
}

// pinnedEnv returns the entry env that pins the config file, or nil when it is not pinned.
func pinnedEnv(pinned string) map[string]string {
	if pinned == "" {
		return nil
	}
	return map[string]string{mcpconfig.ConfigFileEnv: pinned}
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
	// refused are the agents that will not be written, so a conflict in them does not refuse the run.
	refused map[*agents.Agent]error
	// binary and env are the command and env of the entries the write makes.
	binary string
	env    map[string]string
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
	chosen := map[string]bool{}
	for _, fullName := range fullNames {
		entry, err := mcp.FindEntry(ctx, state, fullName, in.binding.profile, in.binding.configFile)
		if err != nil {
			return nil, err
		}
		name, failed, note, err := chooseName(ctx, in, state, fullName, entry, taken)
		if err != nil {
			return nil, err
		}
		// A name recovered from an agent entry can repeat across services only through a hand edit; two plans must not share it.
		if chosen[name] {
			return nil, failure(protos.AitoolsErrorCategoryMCPNameConflict, "%s and another requested server both use %s; remove one of the entries first", fullName, name)
		}
		chosen[name] = true
		taken[name] = true
		plan := addPlan{
			service:    mcp.Service{FullName: fullName},
			name:       name,
			failed:     failed,
			note:       note,
			reasserted: map[*agents.Agent]bool{},
			updated:    map[*agents.Agent]bool{},
			replaced:   map[*agents.Agent]bool{},
		}
		for _, a := range in.targets {
			entry, held := in.held[a][name]
			if !held || failed[a] != nil {
				continue
			}
			// A foreign entry is replaced. A Databricks entry is re-asserted when the write leaves it as it is, and updated
			// when the write changes it.
			if !mcpconfig.IsDatabricksEntry(entry) {
				plan.replaced[a] = true
				continue
			}
			changed, err := agentconfig.Changed(a, entry, in.binary, mcpconfig.ServeArgs(in.binding.profile, fullName, a.Name), in.env)
			if err != nil {
				// The write reports a bad entry with its file, so the plan only has to show that it changes.
				plan.updated[a] = true
				continue
			}
			if !changed {
				plan.reasserted[a] = true
			} else {
				plan.updated[a] = true
			}
		}
		plans = append(plans, plan)
	}
	return plans, nil
}

// chooseName returns the server name for fullName, the target agents that cannot take it, and a note when the name
// was generated with a suffix. entry is the server's state entry, nil for a new install. The name it holds is kept.
// Otherwise --name is used when no other server holds it, and the name is generated.
func chooseName(ctx context.Context, in planInput, state *mcp.State, fullName string, entry *mcp.Entry, taken map[string]bool) (string, map[*agents.Agent]error, string, error) {
	b := in.binding
	owned, err := ownedNames(ctx, in.held, in.targets, fullName, b)
	if err != nil {
		return "", nil, "", err
	}
	stored := ""
	if entry != nil {
		stored = entry.Name
		if stored == "" {
			return "", nil, "", failure(protos.AitoolsErrorCategoryUncategorized, "%s is installed on profile %s without a server name in the state; remove that entry and add it again", fullName, b.profile)
		}
		// The agents must hold the state's name, or the run would add a second name for the same server beside theirs.
		if slices.ContainsFunc(owned, func(name string) bool { return name != stored }) {
			return "", nil, "", failure(protos.AitoolsErrorCategoryMCPNameConflict, "%s is installed on profile %s as %s, but agents hold it as %s; remove the extra entries", fullName, b.profile, stored, strings.Join(owned, ", "))
		}
	} else {
		switch len(owned) {
		case 0:
		case 1:
			stored = owned[0]
		default:
			return "", nil, "", failure(protos.AitoolsErrorCategoryMCPNameConflict, "%s is installed on profile %s under more than one name in the agents (%s); remove the extra entries", fullName, b.profile, strings.Join(owned, ", "))
		}
	}

	if stored != "" {
		// A stored name gets the checks that --name gets, since a hand edit can leave a name that does not split as mcp__<name>__<tool> does.
		if mcp.Sanitize(stored) != stored {
			return "", nil, "", failure(protos.AitoolsErrorCategoryUncategorized, "%s is installed as %q, which is not a valid server name: "+serverNameRules+". Run: %s and add it again", fullName, stored, mcp.RemoveNamesCommand(stored))
		}
		if len(stored) > mcp.MaxServerNameLen {
			return "", nil, "", failure(protos.AitoolsErrorCategoryUncategorized, "%s is installed as %q, which is longer than the %d-character server name limit. Run: %s and add it again", fullName, stored, mcp.MaxServerNameLen, mcp.RemoveNamesCommand(stored))
		}
		if in.nameFlag != "" && in.nameFlag != stored {
			return "", nil, "", failure(protos.AitoolsErrorCategoryInvalidFlags, "%s is already installed on profile %s as %s; remove it before using --name %s", fullName, b.profile, stored, in.nameFlag)
		}
		if stateHolds(state, stored, entry) {
			return "", nil, "", failure(protos.AitoolsErrorCategoryMCPNameConflict, "%s is installed as %s, but another server in the state holds %s; remove that entry first", fullName, stored, stored)
		}
		failed, err := nameFailures(ctx, in, fullName, stored, true)
		return stored, failed, "", err
	}

	if in.nameFlag != "" {
		if stateHolds(state, in.nameFlag, nil) {
			return "", nil, "", failure(protos.AitoolsErrorCategoryInvalidFlags, "--name %s is used by another installed server; pick a different name", in.nameFlag)
		}
		failed, err := nameFailures(ctx, in, fullName, in.nameFlag, false)
		return in.nameFlag, failed, "", err
	}

	base, err := mcp.ServerName(fullName, b.profile)
	if err != nil {
		return "", nil, "", withCategory(fmt.Errorf("%w; pass --name", err), protos.AitoolsErrorCategoryInvalidFlags)
	}
	// No entry of this binding exists, so every name that an agent holds belongs to another server.
	reserved := map[string]bool{}
	for _, a := range in.targets {
		for name := range in.held[a] {
			reserved[name] = true
		}
	}
	name := mcp.UniqueName(base, func(n string) bool { return taken[n] || reserved[n] })
	return name, nil, suffixNote(in, base, name), nil
}

// suffixNote explains a generated name that got a suffix: the target agents that hold the base name, or else another
// installed server. It is "" when the base name is used as it is.
func suffixNote(in planInput, base, name string) string {
	if name == base {
		return ""
	}
	var holders []string
	for _, a := range in.targets {
		if _, ok := in.held[a][base]; ok {
			holders = append(holders, a.DisplayName)
		}
	}
	if len(holders) == 0 {
		return fmt.Sprintf("%s is taken by another installed server; using %s", base, name)
	}
	return fmt.Sprintf("%s is taken in %s; using %s", base, strings.Join(holders, ", "), name)
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
		if in.refused[a] != nil {
			continue
		}
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
		return failure(protos.AitoolsErrorCategoryMCPNameConflict, "%s already has a server named %s that this CLI did not write; pass --force to replace it", a.DisplayName, name), nil
	}
	serves, err := mcp.EntryServes(ctx, entry, fullName, in.binding.profile, in.binding.configFile)
	if err != nil {
		return nil, err
	}
	switch {
	case serves:
		return nil, nil
	case stored:
		return failure(protos.AitoolsErrorCategoryMCPNameConflict, "%s is installed as %s, but %s holds %s for another server; remove that entry first", fullName, name, a.Name, name), nil
	default:
		return failure(protos.AitoolsErrorCategoryInvalidFlags, "--name %s is used by another server in %s; pick a different name", name, a.Name), nil
	}
}

// planningRefusals returns the failures of the plans when no agent can take any of the servers, which ends the run
// before any request. An agent in refused cannot take any server. It returns nil when some agent can take some server.
func planningRefusals(plans []addPlan, targets []*agents.Agent, refused map[*agents.Agent]error) error {
	var errs []error
	for _, plan := range plans {
		for _, a := range targets {
			err := refused[a]
			if err == nil {
				err = plan.failed[a]
			}
			if err == nil {
				return nil
			}
			errs = appendDistinct(errs, err)
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
		SignInRequired: plan.service.SignInRequired(),
		Agents:         make([]addAgentOutput, 0, len(req.agents)),
	}
	var replaced []string
	for _, a := range req.agents {
		var path string
		var status agentconfig.Status
		err := req.refused[a]
		if err == nil {
			err = plan.failed[a]
		}
		if err == nil {
			args := mcpconfig.ServeArgs(req.profile, plan.service.FullName, a.Name)
			path, status, err = agentconfig.Upsert(ctx, a, plan.name, req.binary, args, req.env, req.force)
			// A detected agent was not named, so --agents is no way out for it.
			if req.agentSource == agentsNamed {
				err = withAgentsHint(err, a.Name)
			}
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
			if plan.replaced[a] {
				replaced = append(replaced, a.DisplayName)
			}
		}
		out.Agents = append(out.Agents, result)
	}
	// Known is nil for a service outside system.ai, so only a known service can have a note.
	if plan.service.Known != nil {
		out.note = plan.service.Known.Note
	}
	if plan.note != "" {
		out.notes = append(out.notes, plan.note)
	}
	if len(replaced) > 0 {
		out.notes = append(out.notes, fmt.Sprintf("replaced %s in %s (not written by this CLI)", plan.name, strings.Join(replaced, ", ")))
	}
	return out
}

// refusedAgents returns the agents of named, the ones given with --agents, that are not installed, with the reason for
// each. It runs once, before any request, so each service reports the same refusal without another check.
func refusedAgents(ctx context.Context, named []*agents.Agent) map[*agents.Agent]error {
	refused := map[*agents.Agent]error{}
	for _, a := range named {
		if err := requireInstalled(ctx, a); err != nil {
			refused[a] = err
		}
	}
	return refused
}

// requireInstalled fails when an agent is not installed. Writing its config would create the agent's
// directory, so the agent is refused instead.
func requireInstalled(ctx context.Context, a *agents.Agent) error {
	installed, err := a.InstalledForMCP(ctx)
	if err != nil {
		return err
	}
	if installed {
		return nil
	}
	return failureWithCause(protos.AitoolsErrorCategoryUnknownAgent, errNotInstalled, "%s is not installed; install it first or leave it out of --agents", a.DisplayName)
}

// errNotInstalled is the cause of the refusal of an agent that is not installed. A failed check has another cause.
var errNotInstalled = errors.New("agent is not installed")

// signInInstalled signs in to each sign-in-required server that reached an agent, in order, and records the sign-in of
// each server that reached an agent in servers. A sign-in failure does not stop the others: the failures are joined, and
// the servers are installed either way. An abort at a question (Ctrl+C) stops the rest. The servers from the one asked
// about on get their login hint, which the results did not print, and the abort is returned.
//
// In a terminal without --login, each server that is not already signed in is asked about first; a declined one gets
// the login command.
func signInInstalled(ctx context.Context, errOut io.Writer, w *databricks.WorkspaceClient, req addRequest, plans []addPlan, servers []addServerOutput) error {
	var errs []error
	for i := range servers {
		server := &servers[i]
		if !installedAnywhere(*server) {
			continue
		}
		if !server.SignInRequired {
			server.SignIn = &addSignInOutput{State: signInStateNotRequired}
			continue
		}
		// A cancelled command ends the sign-ins that have not started, rather than failing each one with the same error.
		// The sign-in that the cancellation interrupted has already reported it.
		if err := ctx.Err(); err != nil {
			if len(errs) == 0 {
				errs = append(errs, err)
			}
			break
		}
		if req.askLogin {
			// A caller already signed in is recorded as such, with no question and no second read of the credential.
			if signedIn(ctx, w, plans[i].service.FullName) {
				server.SignIn = &addSignInOutput{State: signInStateActive}
				reportSignedIn(errOut, plans[i].service, req.profile, true)
				logMcpEvent(ctx, signInEventOf(server.Service, signInStateActive, true, nil))
				continue
			}
			yes, err := promptSignInNow(server.Service)
			if err != nil {
				errs = append(errs, err)
				printSignInHints(ctx, req, servers[i:])
				break
			}
			if !yes {
				cmdio.LogString(ctx, signInHint(server.Service, req.profile, req.pinned))
				continue
			}
		}
		var state string
		var alreadySignedIn bool
		// A sign-in whose URL cannot be built is logged as a failed sign-in. The install record keeps no state for it.
		eventState := signInStateFailed
		url, err := mcp.SignInURL(w.Config.Host, plans[i].service.FullName, req.workspaceID)
		if err == nil {
			state, alreadySignedIn, _, err = signIn(ctx, errOut, w, req.pinned, url, plans[i].service, signInOptions{Profile: req.profile, Open: browser.OpenOutputToStderr})
			eventState = state
		}
		server.SignIn = &addSignInOutput{State: state}
		logMcpEvent(ctx, signInEventOf(server.Service, eventState, alreadySignedIn, err))
		if err != nil {
			server.SignIn.ErrorCategory = errorCategory(err)
			server.SignIn.Error = err.Error()
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// signedIn reports whether the caller's sign-in to service is ACTIVE, so that no question is asked about it. The read is
// bounded by profileCheckTimeout, as the --check reads are. A read that fails or times out does not count as signed in;
// the sign-in reads it again and reports the failure.
func signedIn(ctx context.Context, w *databricks.WorkspaceClient, service string) bool {
	ctx, cancel := context.WithTimeout(ctx, profileCheckTimeout)
	defer cancel()
	state, err := mcp.CredentialState(ctx, w, service)
	return err == nil && state == catalog.ProvisioningInfoStateActive
}

// printSignInHints prints the login hint of each server from the given one on that needs sign-in and reached an agent.
// It runs when an abort ends the sign-ins. It makes no request, since a credential read can wait out the SDK's retries
// and hold up the abort. So a server that is already signed in can get the hint too.
func printSignInHints(ctx context.Context, req addRequest, servers []addServerOutput) {
	for i := range servers {
		if servers[i].SignInRequired && installedAnywhere(servers[i]) {
			cmdio.LogString(ctx, signInHint(servers[i].Service, req.profile, req.pinned))
		}
	}
}

// signInHint returns the sentence that says a sign-in-required service needs sign-in, with the command that signs the
// profile in.
func signInHint(service, profileName, pinnedFile string) string {
	return fmt.Sprintf("%s needs sign-in. Run: %s", service, mcpLoginHint(service, profileName, pinnedFile))
}

// installedAnywhere reports whether the server reached at least one agent.
func installedAnywhere(server addServerOutput) bool {
	for _, a := range server.Agents {
		if a.Status != statusFailed {
			return true
		}
	}
	return false
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
	entry.SignInRequired = plan.service.SignInRequired()
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
	fail := req.agentSource == agentsNamed
	for _, server := range servers {
		for _, a := range server.Agents {
			if a.err != nil {
				errs = appendDistinct(errs, a.err)
			}
		}
		if !installedAnywhere(server) {
			fail = true
		}
	}
	if len(errs) == 0 || !fail {
		return nil
	}
	return &agentErrors{err: errors.Join(errs...)}
}

// printAddHeader prints the profile and host that the servers bind to, unless the output is JSON.
func printAddHeader(ctx context.Context, req addRequest) {
	if req.jsonMode {
		return
	}
	cmdio.LogString(ctx, fmt.Sprintf("Profile %s (%s)", req.profile, req.host))
	names := make([]string, 0, len(req.agents))
	for _, a := range req.agents {
		names = append(names, a.Name)
	}
	cmdio.LogString(ctx, fmt.Sprintf("Agents (%s): %s", req.agentSource, strings.Join(names, ", ")))
}

// printAddPlan lists what each agent gets, before the confirm of an interactive add.
func printAddPlan(ctx context.Context, req addRequest, plans []addPlan) {
	cmdio.LogString(ctx, "")
	for _, line := range addPlanLines(req, plans) {
		cmdio.LogString(ctx, line)
	}
	cmdio.LogString(ctx, "")
}

// addPlanLines returns the plan: one line per agent, with the servers it gets as new entries, the servers it already
// has as the write would leave them (re-asserted), the Databricks entries the write changes (updated), the foreign
// entries it replaces, and the servers it cannot take. An agent whose config cannot be read gets one line saying so.
func addPlanLines(req addRequest, plans []addPlan) []string {
	lines := []string{"Plan (profile " + req.profile + "):"}
	for _, a := range req.agents {
		if err := req.refused[a]; err != nil {
			reason := "not installed"
			if !errors.Is(err, errNotInstalled) {
				reason = "cannot be checked: " + err.Error()
			}
			lines = append(lines, fmt.Sprintf("  %-14s %s; skipped", a.DisplayName, reason))
			continue
		}
		var added, reasserted, updated, replaced, skipped []string
		for _, p := range plans {
			switch {
			case p.failed[a] != nil:
				skipped = append(skipped, p.name)
			case p.reasserted[a]:
				reasserted = append(reasserted, p.name)
			case p.updated[a]:
				updated = append(updated, p.name)
			case p.replaced[a]:
				replaced = append(replaced, p.name)
			default:
				added = append(added, p.name)
			}
		}
		var parts []string
		if len(added) > 0 {
			parts = append(parts, "add "+strings.Join(added, ", "))
		}
		if len(reasserted) > 0 {
			parts = append(parts, "re-assert "+strings.Join(reasserted, ", "))
		}
		if len(updated) > 0 {
			parts = append(parts, "update "+strings.Join(updated, ", "))
		}
		if len(replaced) > 0 {
			parts = append(parts, "replace "+strings.Join(replaced, ", "))
		}
		if len(skipped) > 0 {
			parts = append(parts, "skip "+strings.Join(skipped, ", "))
		}
		lines = append(lines, fmt.Sprintf("  %-14s %s", a.DisplayName, strings.Join(parts, "; ")))
	}
	return lines
}

// printAddResults prints each server, its agents, and the follow-up steps the server needs. The Cursor approval
// hint comes once, after the servers.
func printAddResults(ctx context.Context, req addRequest, servers []addServerOutput) {
	if req.jsonMode {
		return
	}
	var cursorServers []string
	for _, server := range servers {
		cmdio.LogString(ctx, "")
		cmdio.LogString(ctx, server.Name+"  "+server.Service)
		for _, note := range server.notes {
			cmdio.LogString(ctx, "  "+note)
		}
		for _, a := range server.Agents {
			cmdio.LogString(ctx, "  "+agentResultLine(ctx, a))
			if a.Name == agents.NameCursor && a.Status == statusInstalled {
				cursorServers = append(cursorServers, server.Name)
			}
		}
		if server.SignInRequired && installedAnywhere(server) && !req.login && !req.askLogin {
			cmdio.LogString(ctx, "  "+signInHint(server.Service, req.profile, req.pinned))
		}
		if server.note != "" && installedAnywhere(server) {
			cmdio.LogString(ctx, "  "+server.note)
		}
	}
	if len(cursorServers) > 0 {
		cmdio.LogString(ctx, "")
		cmdio.LogString(ctx, cursorApprovalHint(cursorServers))
	}
}

// agentResultLine returns the text line that reports one agent's result for a server.
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

// cursorApprovalHint explains the approval step Cursor asks for on new servers. The Cursor docs
// (https://cursor.com/docs/cli/mcp) call the CLI "agent" and list "agent mcp enable <identifier>";
// the installed binary is cursor-agent, so the hint names that. enable takes one identifier.
func cursorApprovalHint(names []string) string {
	if len(names) == 1 {
		return fmt.Sprintf("Cursor may ask you to approve %s. Run: cursor-agent mcp enable %s (`agent` in the Cursor docs), or approve it in Cursor's MCP settings.", names[0], names[0])
	}
	return fmt.Sprintf("Cursor may ask you to approve %s. Run cursor-agent mcp enable <name> for each (`agent` in the Cursor docs), or approve them in Cursor's MCP settings.", strings.Join(names, ", "))
}
