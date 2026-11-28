package aitools

import (
	"context"
	"strings"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/config"
)

// setUploadConfig prepares telemetry after the command has resolved its workspace.
// For offline removal, bound is nil and only a token in the profile itself can be used.
func setUploadConfig(ctx context.Context, profileName, configFile string, bound *config.Config, authorization string, recorded ...mcp.Entry) {
	if telemetry.Disabled(ctx) {
		return
	}
	if bound != nil {
		if bound.AuthType == "" {
			telemetry.SkipUpload(ctx)
			return
		}
		if err := mcp.CheckUnrecordedBinding("", profileName, configFile, bound); err != nil {
			log.Debugf(ctx, "telemetry: profile %s no longer matches the bound workspace: %s", profileName, err)
			telemetry.SkipUpload(ctx)
			return
		}
	}
	cfg, err := mcp.BoundUploadConfig(ctx, profileName, configFile, telemetryAuthBudget)
	if err != nil {
		log.Debugf(ctx, "telemetry: cannot resolve profile %s to upload with: %s", profileName, err)
		telemetry.SkipUpload(ctx)
		return
	}
	if bound == nil {
		pat := config.PatCredentials{}
		if cfg.Token == "" || (cfg.AuthType != "" && cfg.AuthType != pat.Name()) {
			telemetry.SkipUpload(ctx)
			return
		}
		cfg.Credentials = pat
		for _, entry := range recorded {
			target := &config.Config{Host: entry.Host, WorkspaceID: entry.WorkspaceID}
			target.Host = target.CanonicalHostName()
			if !sameUploadWorkspace(cfg, target) {
				telemetry.SkipUpload(ctx)
				return
			}
		}
	} else {
		if !sameUploadWorkspace(cfg, bound) || (cfg.AuthType != "" && cfg.AuthType != bound.AuthType) {
			telemetry.SkipUpload(ctx)
			return
		}
		if bound.AuthType == (auth.CLICredentials{}).Name() {
			// The CLI OAuth cache is keyed by profile name across config files.
			// Reuse the exact bearer token obtained during binding, with a
			// static strategy that cannot look up or refresh an OAuth token.
			token, ok := strings.CutPrefix(authorization, "Bearer ")
			if !ok || token == "" {
				telemetry.SkipUpload(ctx)
				return
			}
			cfg.Token = token
			cfg.Credentials = config.PatCredentials{}
			cfg.AuthType = cfg.Credentials.Name()
		} else {
			cfg.AuthType = bound.AuthType
		}
	}
	telemetry.SetUploadConfig(ctx, cfg)
}

func sameUploadWorkspace(a, b *config.Config) bool {
	return mcp.NormalizeHost(a.Host) == mcp.NormalizeHost(b.Host) &&
		auth.WorkspaceIDHeaders(a)[auth.WorkspaceIDHeader] == auth.WorkspaceIDHeaders(b)[auth.WorkspaceIDHeader]
}

// mcpServiceTypes maps the system.ai leaves in the CLI's service table to their telemetry enum.
var mcpServiceTypes = map[string]protos.AitoolsMcpService{
	"genie_one_mcp":   protos.AitoolsMcpServiceGenieOne,
	"dbsql":           protos.AitoolsMcpServiceDBSQL,
	"web_search":      protos.AitoolsMcpServiceWebSearch,
	"sandbox":         protos.AitoolsMcpServiceSandbox,
	"slack":           protos.AitoolsMcpServiceSlack,
	"github":          protos.AitoolsMcpServiceGitHub,
	"atlassian":       protos.AitoolsMcpServiceAtlassian,
	"google_drive":    protos.AitoolsMcpServiceGoogleDrive,
	"google_calendar": protos.AitoolsMcpServiceGoogleCalendar,
	"gmail":           protos.AitoolsMcpServiceGmail,
	"microsoft_365":   protos.AitoolsMcpServiceMicrosoft365,
}

// mcpServiceType maps a service full name to its telemetry enum. The name itself is never sent:
// a system.ai leaf in the table has its own value, other system.ai leaves are OTHER_SYSTEM, and
// services outside system.ai are CUSTOM.
func mcpServiceType(fullName string) protos.AitoolsMcpService {
	parts := strings.Split(fullName, ".")
	if len(parts) != 3 || !strings.EqualFold(parts[0]+"."+parts[1], mcp.SystemAILocation) {
		return protos.AitoolsMcpServiceCustom
	}
	if t, ok := mcpServiceTypes[strings.ToLower(parts[2])]; ok {
		return t
	}
	return protos.AitoolsMcpServiceOtherSystem
}

// CallingAgentType returns the agent that drives the CLI, Unspecified when none is detected. The skills commands
// report it as their calling agent too.
func CallingAgentType() protos.AitoolsAgentType {
	return callingAgentType()
}

// callingAgentType returns the agent that drives the CLI, Unspecified when none is detected.
func callingAgentType() protos.AitoolsAgentType {
	a := agents.CallingAgent()
	if a == nil {
		return protos.AitoolsAgentTypeUnspecified
	}
	return agentType(a.Name)
}

// proxyAgentType returns the agent that started the proxy: the --agent flag when given, else the
// detected agent.
func proxyAgentType(flag string) protos.AitoolsAgentType {
	if flag == "" {
		return callingAgentType()
	}
	return agentType(flag)
}

// logMcpEvent buffers an MCP event for the next upload and reports whether it did. An event without server
// results is not logged: a run that stopped before any server was reached has nothing to report. The bound profile count
// is read from the state file as it is after the operation. A state file that cannot be read drops the event; telemetry
// never fails a command.
func logMcpEvent(ctx context.Context, event protos.AitoolsMcpEvent) bool {
	if telemetry.Disabled(ctx) || len(event.Servers) == 0 {
		return false
	}
	state, err := mcp.LoadState(ctx)
	if err != nil {
		log.Debugf(ctx, "telemetry: cannot read the MCP state; dropping the event: %s", err)
		return false
	}
	event.BoundProfileCount = boundProfileCount(ctx, state)
	return telemetry.Log(ctx, protos.DatabricksCliLog{AitoolsMcpEvent: &event})
}

// boundProfileCount returns the number of distinct profiles that have servers installed, grouped as list groups them
// (profileGroup), so DEFAULT in two config files counts twice.
func boundProfileCount(ctx context.Context, state *mcp.State) int64 {
	profiles := map[profileKey]bool{}
	for _, e := range state.Servers {
		path, _ := mcp.ResolveConfigFile(ctx, e.ConfigFile)
		profiles[profileGroup(e, mcp.EntryCheck{Path: path})] = true
	}
	return int64(len(profiles))
}

// addSignInState reports the state add observed. A service may require sign-in
// even when add never checks whether the user has already signed in.
func addSignInState(s addServerOutput) protos.AitoolsMcpSignInState {
	if s.SignIn != nil && s.SignIn.State != "" {
		return signInStateOf(s.SignIn.State)
	}
	if s.SignInRequired {
		if s.SignIn == nil {
			return ""
		}
		return protos.AitoolsMcpSignInStatePending
	}
	return protos.AitoolsMcpSignInStateNotRequired
}

// addServerResults returns one result per (service, agent) of an add run. Installed entries are
// CHANGED, re-asserted ones UNCHANGED, and failed ones FAILED with their category.
func addServerResults(servers []addServerOutput) []protos.AitoolsMcpServerResult {
	var results []protos.AitoolsMcpServerResult
	for _, s := range servers {
		for _, a := range s.Agents {
			r := protos.AitoolsMcpServerResult{
				Service:     mcpServiceType(s.Service),
				Agent:       agentType(a.Name),
				SignInState: addSignInState(s),
			}
			switch a.Status {
			case statusInstalled:
				r.Outcome = protos.AitoolsTargetOutcomeChanged
			case statusReasserted:
				r.Outcome = protos.AitoolsTargetOutcomeUnchanged
			default:
				r.Outcome = protos.AitoolsTargetOutcomeFailed
				r.ErrorCategory = a.ErrorCategory
			}
			results = append(results, r)
		}
	}
	return results
}

// removeServerResult returns the result of removing one server from one agent config. An entry that was deleted, or
// pruned (its record dropped), is CHANGED. An entry that was changed outside this CLI is SKIPPED: it is left in place.
func removeServerResult(service, agent string, outcome removeOutcome, err error) protos.AitoolsMcpServerResult {
	r := protos.AitoolsMcpServerResult{Service: mcpServiceType(service), Agent: agentType(agent)}
	switch {
	case err != nil:
		r.Outcome = protos.AitoolsTargetOutcomeFailed
		r.ErrorCategory = errorCategory(err)
	case outcome == removeKept:
		r.Outcome = protos.AitoolsTargetOutcomeSkipped
	default:
		r.Outcome = protos.AitoolsTargetOutcomeChanged
	}
	return r
}

// signInStateOf maps a sign-in state name of login to its telemetry state. A run that never learned the state leaves it
// unset. Anything but ACTIVE or NOT_REQUIRED is still PENDING.
func signInStateOf(state string) protos.AitoolsMcpSignInState {
	switch state {
	case "":
		return ""
	case signInStateActive:
		return protos.AitoolsMcpSignInStateActive
	case signInStateNotRequired:
		return protos.AitoolsMcpSignInStateNotRequired
	}
	return protos.AitoolsMcpSignInStatePending
}

// signInServerResult returns the result of one provider sign-in. The sign-in was already active
// (alreadySignedIn) or the service needed none, so nothing changed: UNCHANGED.
func signInServerResult(service, state string, alreadySignedIn bool, err error) protos.AitoolsMcpServerResult {
	r := protos.AitoolsMcpServerResult{Service: mcpServiceType(service), SignInState: signInStateOf(state)}
	switch {
	case err != nil:
		r.Outcome = protos.AitoolsTargetOutcomeFailed
		r.ErrorCategory = errorCategory(err)
	case alreadySignedIn || state == signInStateNotRequired:
		r.Outcome = protos.AitoolsTargetOutcomeUnchanged
	default:
		r.Outcome = protos.AitoolsTargetOutcomeChanged
	}
	return r
}

// signInEventOf is the SIGN_IN event of one provider sign-in, under add --login or login. A sign-in that found the caller
// already signed in is UNCHANGED.
func signInEventOf(service, state string, alreadySignedIn bool, err error) protos.AitoolsMcpEvent {
	return protos.AitoolsMcpEvent{
		CallingAgent: callingAgentType(),
		Operation:    protos.AitoolsMcpOperationSignIn,
		Servers:      []protos.AitoolsMcpServerResult{signInServerResult(service, state, alreadySignedIn, err)},
	}
}

// signOutServerResult returns the result of a logout. Logout does not read whether the service needs
// sign-in, so the sign-in state is not set.
func signOutServerResult(service, state string, err error) protos.AitoolsMcpServerResult {
	r := protos.AitoolsMcpServerResult{Service: mcpServiceType(service)}
	switch {
	case err != nil:
		r.Outcome = protos.AitoolsTargetOutcomeFailed
		r.ErrorCategory = errorCategory(err)
	case state == signOutStateDeleted:
		r.Outcome = protos.AitoolsTargetOutcomeChanged
	default:
		r.Outcome = protos.AitoolsTargetOutcomeUnchanged
	}
	return r
}

// proxyStartedEvent is the PROXY_STARTED event of serve, logged once the startup checks have passed and the proxy serves.
// A start that fails its checks, whether at the binding or on a rejected credential, logs nothing, so its run has nothing
// to upload.
func proxyStartedEvent(agent protos.AitoolsAgentType, service string) protos.AitoolsMcpEvent {
	return protos.AitoolsMcpEvent{
		CallingAgent: agent,
		Operation:    protos.AitoolsMcpOperationProxyStarted,
		Servers:      []protos.AitoolsMcpServerResult{{Service: mcpServiceType(service)}},
	}
}

// mcpProxyTelemetry is the proxy's telemetry for one serve session. It implements proxy.Recorder.
type mcpProxyTelemetry struct {
	agent   protos.AitoolsAgentType
	service string
}

// SignInPrompted logs SIGN_IN_PROMPTED once the sign-in page the authenticate tool opened has been
// waited on. signedIn reports whether the credential became ACTIVE.
func (t mcpProxyTelemetry) SignInPrompted(ctx context.Context, signedIn bool) {
	logMcpEvent(ctx, signInPromptedEvent(t.agent, t.service, signedIn))
}

// signInPromptedEvent is the SIGN_IN_PROMPTED event of serve: the sign-in is required, and the
// sign-in state is ACTIVE when the wait ended with the credential, PENDING otherwise.
func signInPromptedEvent(agent protos.AitoolsAgentType, service string, signedIn bool) protos.AitoolsMcpEvent {
	state := protos.AitoolsMcpSignInStatePending
	if signedIn {
		state = protos.AitoolsMcpSignInStateActive
	}
	return protos.AitoolsMcpEvent{
		CallingAgent: agent,
		Operation:    protos.AitoolsMcpOperationSignInPrompted,
		Servers: []protos.AitoolsMcpServerResult{{
			Service:       mcpServiceType(service),
			ErrorCategory: protos.AitoolsErrorCategorySignInRequired,
			SignInState:   state,
		}},
	}
}

// recordedUploadTargets returns the entries under key after checking their
// bindings against the config file. A missing profile or drift skips the upload
// before resolving credentials or host metadata.
func recordedUploadTargets(ctx context.Context, state *mcp.State, key profileKey) []mcp.Entry {
	checks := mcp.CheckEntries(ctx, state)
	var entries []mcp.Entry
	for i, e := range state.Servers {
		if profileGroup(e, checks[i]) != key {
			continue
		}
		if checks[i].Err != nil || checks[i].Drift != nil {
			return nil
		}
		entries = append(entries, e)
	}
	return entries
}

// sharedProfile returns the profile and config file that all entries are bound to, grouped as list groups them, or the
// zero key when they are bound to none or to several. checks are the checks of entries, in the same order.
func sharedProfile(entries []mcp.Entry, checks []mcp.EntryCheck) profileKey {
	if len(entries) == 0 {
		return profileKey{}
	}
	first := profileGroup(entries[0], checks[0])
	for i, e := range entries[1:] {
		if profileGroup(e, checks[i+1]) != first {
			return profileKey{}
		}
	}
	return first
}
