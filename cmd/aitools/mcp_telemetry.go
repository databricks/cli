package aitools

import (
	"context"
	"strings"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

// setUploadConfig sets the config of profileName in configFile for the exit-time upload, resolved within
// telemetryAuthBudget, with credentials that never refresh (mcp.BoundUploadConfig). A profile that does not resolve
// uploads nothing.
func setUploadConfig(cmd *cobra.Command, profileName, configFile string) {
	ctx := cmd.Context()
	cfg, err := mcp.BoundUploadConfig(profileName, configFile, telemetryAuthBudget)
	if err != nil {
		log.Debugf(ctx, "telemetry: cannot resolve profile %s to upload with: %s", profileName, err)
		return
	}
	cmd.SetContext(cmdctx.SetConfigUsed(ctx, cfg))
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
	if mcp.KnownOf(fullName) == nil {
		return protos.AitoolsMcpServiceOtherSystem
	}
	return mcpServiceTypes[strings.ToLower(parts[2])]
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

// logMcpEvent buffers an MCP event; cmd/root uploads it at exit. An event without server results
// is not logged: a run that stopped before any server was reached has nothing to report. The bound
// profile count is read from the state file as it is after the operation. A state file that cannot
// be read drops the event; telemetry never fails a command.
func logMcpEvent(ctx context.Context, event protos.AitoolsMcpEvent) {
	if len(event.Servers) == 0 {
		return
	}
	state, err := mcp.LoadState(ctx)
	if err != nil {
		log.Debugf(ctx, "telemetry: cannot read the MCP state; dropping the event: %s", err)
		return
	}
	event.BoundProfileCount = boundProfileCount(ctx, state)
	telemetry.Log(ctx, protos.DatabricksCliLog{AitoolsMcpEvent: &event})
}

// boundProfileCount returns the number of distinct profiles that have servers installed. A profile is its name in its
// config file, as list groups them, so DEFAULT in two config files counts twice.
func boundProfileCount(ctx context.Context, state *mcp.State) int64 {
	profiles := map[profileKey]bool{}
	for _, e := range state.Servers {
		path, err := mcp.ResolveConfigFile(ctx, e.ConfigFile)
		if err != nil {
			path = e.ConfigFile
		}
		profiles[profileKey{profile: e.Profile, path: path}] = true
	}
	return int64(len(profiles))
}

// addSignInState is the sign-in state of a server after an add: the state that --login reports, else PENDING for a
// server that needs sign-in, else NOT_REQUIRED.
func addSignInState(s addServerOutput) protos.AitoolsMcpSignInState {
	if s.SignIn != nil {
		return signInStateOf(s.SignIn.State)
	}
	if s.SignInRequired {
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

// removeServerResult returns the result of removing one server from one agent config. An entry that
// is gone already, or that was changed outside this CLI, is SKIPPED: nothing was changed there.
func removeServerResult(service, agent string, outcome removeOutcome, err error) protos.AitoolsMcpServerResult {
	r := protos.AitoolsMcpServerResult{Service: mcpServiceType(service), Agent: agentType(agent)}
	switch {
	case err != nil:
		r.Outcome = protos.AitoolsTargetOutcomeFailed
		r.ErrorCategory = errorCategory(err)
	case outcome == removeDeleted:
		r.Outcome = protos.AitoolsTargetOutcomeChanged
	default:
		r.Outcome = protos.AitoolsTargetOutcomeSkipped
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

// addSignInEvent is the SIGN_IN event of one provider sign-in under add --login. A sign-in that found the caller
// already signed in is UNCHANGED.
func addSignInEvent(service, state string, alreadySignedIn bool, err error) protos.AitoolsMcpEvent {
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

// proxyStartedEvent is the PROXY_STARTED event of serve, logged once the startup checks have run. err is the failure
// of a check, or nil when the proxy is about to serve. The event uploads only when its profile resolved: a failed
// binding with no profile to upload with leaves it unsent.
func proxyStartedEvent(agent protos.AitoolsAgentType, service string, err error) protos.AitoolsMcpEvent {
	r := protos.AitoolsMcpServerResult{Service: mcpServiceType(service)}
	if err != nil {
		r.ErrorCategory = errorCategory(err)
	}
	return protos.AitoolsMcpEvent{
		CallingAgent: agent,
		Operation:    protos.AitoolsMcpOperationProxyStarted,
		Servers:      []protos.AitoolsMcpServerResult{r},
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

// bindingDrifted reports whether a server recorded for cfg's profile in state no longer matches the profile: its host or
// workspace changed, or its config file moved. The upload would send the profile's token to the host the profile names
// now, which the server was not set up against. It reads the config files only.
func bindingDrifted(ctx context.Context, state *mcp.State, cfg *config.Config) bool {
	for _, e := range state.Servers {
		if e.Profile != cfg.Profile {
			continue
		}
		if _, err := mcp.CheckBinding(ctx, state, e.Service, cfg.Profile, cfg.ConfigFile); err != nil {
			return true
		}
	}
	return false
}
