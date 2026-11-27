package aitools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"path/filepath"
	"slices"
	"testing"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMcpServiceType(t *testing.T) {
	// Every leaf in the CLI's known table has its own enum, so a known service never reports OTHER_SYSTEM.
	cases := map[string]protos.AitoolsMcpService{
		"system.ai.genie_one_mcp":   protos.AitoolsMcpServiceGenieOne,
		"system.ai.dbsql":           protos.AitoolsMcpServiceDBSQL,
		"system.ai.web_search":      protos.AitoolsMcpServiceWebSearch,
		"system.ai.sandbox":         protos.AitoolsMcpServiceSandbox,
		"system.ai.slack":           protos.AitoolsMcpServiceSlack,
		"system.ai.github":          protos.AitoolsMcpServiceGitHub,
		"system.ai.atlassian":       protos.AitoolsMcpServiceAtlassian,
		"system.ai.google_drive":    protos.AitoolsMcpServiceGoogleDrive,
		"system.ai.google_calendar": protos.AitoolsMcpServiceGoogleCalendar,
		"system.ai.gmail":           protos.AitoolsMcpServiceGmail,
		"system.ai.microsoft_365":   protos.AitoolsMcpServiceMicrosoft365,
		// The schema name is case-insensitive in Unity Catalog.
		"System.AI.GitHub": protos.AitoolsMcpServiceGitHub,
		// A system.ai leaf the CLI does not know yet.
		"system.ai.future_tool": protos.AitoolsMcpServiceOtherSystem,
		// Anything outside system.ai, whatever its name.
		"main.tools.github":  protos.AitoolsMcpServiceCustom,
		"system.ai":          protos.AitoolsMcpServiceCustom,
		"system.ai.a.github": protos.AitoolsMcpServiceCustom,
	}
	for name, want := range cases {
		assert.Equal(t, want, mcpServiceType(name), name)
	}
}

// TestMcpServiceTypeCoversKnownServices checks that every service in the CLI's known table has its own enum, so a known
// service never reports OTHER_SYSTEM. A new table entry needs a new value in mcpServiceTypes.
func TestMcpServiceTypeCoversKnownServices(t *testing.T) {
	for _, leaf := range mcp.KnownLeaves() {
		assert.Contains(t, mcpServiceTypes, leaf, "system.ai.%s has no telemetry enum: add it to mcpServiceTypes", leaf)
		assert.NotEqualf(t, protos.AitoolsMcpServiceOtherSystem, mcpServiceType(mcp.SystemAILocation+"."+leaf),
			"system.ai.%s has no telemetry enum", leaf)
	}
}

func TestAddServerResults(t *testing.T) {
	servers := []addServerOutput{
		{
			Service:        "system.ai.github",
			SignInRequired: true,
			SignIn:         &addSignInOutput{State: signInStateActive},
			Agents: []addAgentOutput{
				{Name: "claude-code", Status: statusInstalled},
				{Name: "codex", Status: statusReasserted},
				{Name: "cursor", Status: statusFailed, ErrorCategory: protos.AitoolsErrorCategoryAgentConfigInvalid},
			},
		},
		{
			Service: "main.tools.custom",
			Agents:  []addAgentOutput{{Name: "claude-code", Status: statusInstalled}},
		},
	}

	assert.Equal(t, []protos.AitoolsMcpServerResult{
		{
			Service:     protos.AitoolsMcpServiceGitHub,
			Agent:       protos.AitoolsAgentTypeClaudeCode,
			Outcome:     protos.AitoolsTargetOutcomeChanged,
			SignInState: protos.AitoolsMcpSignInStateActive,
		},
		{
			Service:     protos.AitoolsMcpServiceGitHub,
			Agent:       protos.AitoolsAgentTypeCodex,
			Outcome:     protos.AitoolsTargetOutcomeUnchanged,
			SignInState: protos.AitoolsMcpSignInStateActive,
		},
		{
			Service:       protos.AitoolsMcpServiceGitHub,
			Agent:         protos.AitoolsAgentTypeCursor,
			Outcome:       protos.AitoolsTargetOutcomeFailed,
			ErrorCategory: protos.AitoolsErrorCategoryAgentConfigInvalid,
			SignInState:   protos.AitoolsMcpSignInStateActive,
		},
		{
			Service:     protos.AitoolsMcpServiceCustom,
			Agent:       protos.AitoolsAgentTypeClaudeCode,
			Outcome:     protos.AitoolsTargetOutcomeChanged,
			SignInState: protos.AitoolsMcpSignInStateNotRequired,
		},
	}, addServerResults(servers))
}

func TestRemoveServerResult(t *testing.T) {
	assert.Equal(t, protos.AitoolsTargetOutcomeChanged,
		removeServerResult("system.ai.github", "claude-code", removeDeleted, nil).Outcome)
	// A pruned entry changes the state, so it is CHANGED too. An entry changed outside this CLI is left in place: SKIPPED.
	assert.Equal(t, protos.AitoolsTargetOutcomeChanged,
		removeServerResult("system.ai.github", "claude-code", removePruned, nil).Outcome)
	assert.Equal(t, protos.AitoolsTargetOutcomeSkipped,
		removeServerResult("system.ai.github", "claude-code", removeKept, nil).Outcome)

	failed := removeServerResult("system.ai.github", "codex", removeDeleted, errors.New("config is not JSON"))
	assert.Equal(t, protos.AitoolsMcpServerResult{
		Service:       protos.AitoolsMcpServiceGitHub,
		Agent:         protos.AitoolsAgentTypeCodex,
		Outcome:       protos.AitoolsTargetOutcomeFailed,
		ErrorCategory: protos.AitoolsErrorCategoryUncategorized,
	}, failed)
}

func TestSignInServerResult(t *testing.T) {
	cases := []struct {
		name            string
		state           string
		alreadySignedIn bool
		err             error
		want            protos.AitoolsMcpServerResult
	}{
		{
			name:  "signed in now",
			state: signInStateActive,
			want:  protos.AitoolsMcpServerResult{Service: protos.AitoolsMcpServiceGitHub, Outcome: protos.AitoolsTargetOutcomeChanged, SignInState: protos.AitoolsMcpSignInStateActive},
		},
		{
			name:            "already active",
			state:           signInStateActive,
			alreadySignedIn: true,
			want:            protos.AitoolsMcpServerResult{Service: protos.AitoolsMcpServiceGitHub, Outcome: protos.AitoolsTargetOutcomeUnchanged, SignInState: protos.AitoolsMcpSignInStateActive},
		},
		{
			name:  "no sign-in needed",
			state: signInStateNotRequired,
			want:  protos.AitoolsMcpServerResult{Service: protos.AitoolsMcpServiceGitHub, Outcome: protos.AitoolsTargetOutcomeUnchanged, SignInState: protos.AitoolsMcpSignInStateNotRequired},
		},
		{
			// A run that never learned the state, such as a read that failed, reports none.
			name: "state not learned",
			err:  failure(protos.AitoolsErrorCategoryMCPServiceNotFound, "not found"),
			want: protos.AitoolsMcpServerResult{
				Service: protos.AitoolsMcpServiceGitHub, Outcome: protos.AitoolsTargetOutcomeFailed,
				ErrorCategory: protos.AitoolsErrorCategoryMCPServiceNotFound,
			},
		},
		{
			name:  "timed out",
			state: signInStateTimeout,
			err:   failure(protos.AitoolsErrorCategorySignInTimeout, "timed out"),
			want: protos.AitoolsMcpServerResult{
				Service: protos.AitoolsMcpServiceGitHub, Outcome: protos.AitoolsTargetOutcomeFailed,
				ErrorCategory: protos.AitoolsErrorCategorySignInTimeout, SignInState: protos.AitoolsMcpSignInStatePending,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, signInServerResult("system.ai.github", tc.state, tc.alreadySignedIn, tc.err))
		})
	}
}

func TestSignOutServerResult(t *testing.T) {
	// Logout does not read the sign-in state, so the result has none.
	assert.Equal(t, protos.AitoolsMcpServerResult{Service: protos.AitoolsMcpServiceGitHub, Outcome: protos.AitoolsTargetOutcomeChanged},
		signOutServerResult("system.ai.github", signOutStateDeleted, nil))
	assert.Equal(t, protos.AitoolsMcpServerResult{Service: protos.AitoolsMcpServiceGitHub, Outcome: protos.AitoolsTargetOutcomeUnchanged},
		signOutServerResult("system.ai.github", signOutStateNotSignedIn, nil))
}

func TestProxyStartedEvent(t *testing.T) {
	assert.Equal(t, protos.AitoolsMcpEvent{
		CallingAgent: protos.AitoolsAgentTypeClaudeCode,
		Operation:    protos.AitoolsMcpOperationProxyStarted,
		Servers:      []protos.AitoolsMcpServerResult{{Service: protos.AitoolsMcpServiceGenieOne}},
	}, proxyStartedEvent(protos.AitoolsAgentTypeClaudeCode, "system.ai.genie_one_mcp"))
}

// TestLogMcpEventLogsRejectedCredential checks that an event that names a rejected credential is logged, and that the run
// uploads with it like any other run: the upload authenticates with the profile the run bound, which never refreshes.
func TestLogMcpEventLogsRejectedCredential(t *testing.T) {
	withHome(t)
	ctx := env.Set(telemetry.WithNewLogger(t.Context()), "DATABRICKS_CLI_DISABLE_TELEMETRY", "")
	telemetry.SetUploadConfig(ctx, &config.Config{Profile: "logfood", Host: "https://adb-1.test"})
	assert.True(t, logMcpEvent(ctx, protos.AitoolsMcpEvent{
		Operation: protos.AitoolsMcpOperationSignIn,
		Servers: []protos.AitoolsMcpServerResult{{
			Service:       protos.AitoolsMcpServiceGitHub,
			Outcome:       protos.AitoolsTargetOutcomeFailed,
			ErrorCategory: protos.AitoolsErrorCategoryAuthFailed,
		}},
	}))
	assert.Len(t, telemetry.BufferedEvents(ctx), 1)
	_, uploads := telemetry.UploadConfig(ctx)
	assert.True(t, uploads)
}

// With DATABRICKS_NOT_AN_AGENT set, the event is logged without the calling agent.
func TestLogMcpEventOmitsCallingAgentUnderNotAnAgent(t *testing.T) {
	withHome(t)
	ctx := env.Set(telemetry.WithNewLogger(t.Context()), "DATABRICKS_CLI_DISABLE_TELEMETRY", "")
	telemetry.SetUploadConfig(ctx, &config.Config{Profile: "logfood", Host: "https://adb-1.test"})
	ctx = env.Set(ctx, "DATABRICKS_NOT_AN_AGENT", "1")
	event := protos.AitoolsMcpEvent{
		CallingAgent: protos.AitoolsAgentTypeClaudeCode,
		Operation:    protos.AitoolsMcpOperationSignIn,
		Servers: []protos.AitoolsMcpServerResult{{
			Service: protos.AitoolsMcpServiceGitHub,
			Outcome: protos.AitoolsTargetOutcomeFailed,
		}},
	}
	require.True(t, logMcpEvent(ctx, event))
	buffered := telemetry.BufferedEvents(ctx)
	require.Len(t, buffered, 1)
	assert.Equal(t, protos.AitoolsAgentTypeUnspecified, buffered[0].AitoolsMcpEvent.CallingAgent)
}

func TestLogMcpEventWithoutLogger(t *testing.T) {
	withHome(t)
	assert.False(t, logMcpEvent(t.Context(), proxyStartedEvent(protos.AitoolsAgentTypeCodex, "system.ai.github")))
}

func TestSetUploadConfigSkipsUnverifiedCredentials(t *testing.T) {
	for _, tc := range []struct {
		name  string
		cfg   string
		bound *config.Config
	}{
		{
			name: "offline OAuth",
			cfg:  "[logfood]\nhost = https://adb-1.test\nauth_type = databricks-cli\n",
		},
		{
			name:  "profile re-pointed",
			cfg:   "[logfood]\nhost = https://adb-2.test\nauth_type = pat\ntoken = test\n",
			bound: &config.Config{Host: "https://adb-1.test", AuthType: "pat"},
		},
		{
			name:  "workspace removed from profile",
			cfg:   "[logfood]\nhost = https://adb-1.test\nauth_type = pat\ntoken = test\n",
			bound: &config.Config{Host: "https://adb-1.test", WorkspaceID: "222", AuthType: "pat"},
		},
		{
			name:  "missing bound OAuth token",
			cfg:   "[logfood]\nhost = https://adb-1.test\nauth_type = databricks-cli\n",
			bound: &config.Config{Host: "https://adb-1.test", AuthType: "databricks-cli"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := withHome(t)
			writeHomeConfig(t, home, tc.cfg)
			ctx := telemetry.WithNewLogger(t.Context())
			setUploadConfig(ctx, "logfood", filepath.Join(home, ".databrickscfg"), tc.bound, "")
			_, uploads := telemetry.UploadConfig(ctx)
			assert.False(t, uploads)
		})
	}
}

func TestSetUploadConfigUsesBoundOAuthToken(t *testing.T) {
	home := withHome(t)
	writeHomeConfig(t, home, "[logfood]\nhost = https://adb-1.test\nworkspace_id = 222\nauth_type = databricks-cli\n")
	ctx := env.Set(telemetry.WithNewLogger(t.Context()), "DATABRICKS_CLI_DISABLE_TELEMETRY", "")
	bound := &config.Config{Host: "https://adb-1.test", WorkspaceID: "222", AuthType: "databricks-cli"}
	setUploadConfig(ctx, "logfood", filepath.Join(home, ".databrickscfg"), bound, "Bearer test-bound-token")

	cfg, uploads := telemetry.UploadConfig(ctx)
	require.True(t, uploads)
	assert.Equal(t, bound.Host, cfg.Host)
	assert.Equal(t, bound.WorkspaceID, cfg.WorkspaceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.Host+"/telemetry-ext", nil)
	require.NoError(t, err)
	require.NoError(t, cfg.Authenticate(req))
	assert.Equal(t, "Bearer test-bound-token", req.Header.Get("Authorization"))
}

// TestSetUploadConfigLogsNoWarningForFailedHostMetadata checks that an upload whose host metadata lookup fails warns
// nothing through the global logger, since the SDK would log that failure there. The command has succeeded, and the
// upload is best-effort.
func TestSetUploadConfigLogsNoWarningForFailedHostMetadata(t *testing.T) {
	home := withHome(t)
	writeHomeConfig(t, home, "[logfood]\nhost = https://adb-1.test\nauth_type = pat\ntoken = t\n")
	factory := config.DefaultHostMetadataResolverFactory
	// Like the registered factory (libs/hostmetadata), the resolver logs its failure as a warning and returns no error.
	config.DefaultHostMetadataResolverFactory = func(*config.Config) config.HostMetadataResolver {
		return func(ctx context.Context, host string) (*config.HostMetadata, error) {
			log.Warnf(ctx, "[hostmetadata] failed to fetch host metadata for %s, will skip for 1m0s", host)
			return nil, nil
		}
	}
	t.Cleanup(func() { config.DefaultHostMetadataResolverFactory = factory })
	var warnings bytes.Buffer
	defaultLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&warnings, &slog.HandlerOptions{Level: slog.LevelWarn})))
	t.Cleanup(func() { slog.SetDefault(defaultLogger) })

	ctx := env.Set(telemetry.WithNewLogger(t.Context()), "DATABRICKS_CLI_DISABLE_TELEMETRY", "")
	setUploadConfig(ctx, "logfood", filepath.Join(home, ".databrickscfg"), nil, "")
	_, uploads := telemetry.UploadConfig(ctx)
	assert.True(t, uploads)
	assert.Empty(t, warnings.String())
}

func TestSignInPromptedEvent(t *testing.T) {
	for signedIn, state := range map[bool]protos.AitoolsMcpSignInState{
		true:  protos.AitoolsMcpSignInStateActive,
		false: protos.AitoolsMcpSignInStatePending,
	} {
		event := signInPromptedEvent(protos.AitoolsAgentTypeCursor, "system.ai.github", signedIn)
		assert.Equal(t, protos.AitoolsMcpEvent{
			CallingAgent: protos.AitoolsAgentTypeCursor,
			Operation:    protos.AitoolsMcpOperationSignInPrompted,
			Servers: []protos.AitoolsMcpServerResult{{
				Service:       protos.AitoolsMcpServiceGitHub,
				ErrorCategory: protos.AitoolsErrorCategorySignInRequired,
				SignInState:   state,
			}},
		}, event)
	}
}

func TestBoundProfileCount(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.cfg")
	second := filepath.Join(dir, "second.cfg")
	state := &mcp.State{Servers: []mcp.Entry{
		{Profile: "logfood", Service: "system.ai.github", ConfigFile: first},
		{Profile: "logfood", Service: "system.ai.genie_one_mcp", ConfigFile: first},
		// DEFAULT in two config files is two profiles, as list groups them.
		{Profile: "DEFAULT", Service: "system.ai.github", ConfigFile: first},
		{Profile: "DEFAULT", Service: "system.ai.github", ConfigFile: second},
	}}
	assert.Equal(t, int64(3), boundProfileCount(t.Context(), state))
	assert.Equal(t, int64(0), boundProfileCount(t.Context(), &mcp.State{}))
}

func TestSignInEventOf(t *testing.T) {
	// An ACTIVE credential that the add found signed in is UNCHANGED, as add --login reports it.
	assert.Equal(t, protos.AitoolsMcpEvent{
		CallingAgent: callingAgentType(),
		Operation:    protos.AitoolsMcpOperationSignIn,
		Servers: []protos.AitoolsMcpServerResult{{
			Service:     protos.AitoolsMcpServiceGitHub,
			Outcome:     protos.AitoolsTargetOutcomeUnchanged,
			SignInState: protos.AitoolsMcpSignInStateActive,
		}},
	}, signInEventOf("system.ai.github", signInStateActive, true, nil))
}

// TestSignInInstalledURLFailureLogsFailedSignIn checks that an add --login whose sign-in URL cannot be built logs the
// sign-in as FAILED with the PENDING state of a failed sign-in, and records no state in the install record.
func TestSignInInstalledURLFailureLogsFailedSignIn(t *testing.T) {
	withHome(t)
	ctx := env.Set(telemetry.WithNewLogger(t.Context()), "DATABRICKS_CLI_DISABLE_TELEMETRY", "")
	w := &databricks.WorkspaceClient{Config: &config.Config{Host: "https://adb-1.test"}}
	servers := []addServerOutput{{
		Service: "system.ai.github", SignInRequired: true,
		Agents: []addAgentOutput{{Name: agents.NameClaudeCode, Status: statusInstalled}},
	}}
	plans := []addPlan{{service: mcp.Service{FullName: "system.ai.github", Leaf: "github"}}}

	require.Error(t, signInInstalled(ctx, io.Discard, w, addRequest{}, plans, servers))
	assert.Empty(t, servers[0].SignIn.State)
	events := telemetry.BufferedEvents(ctx)
	require.Len(t, events, 1)
	require.NotNil(t, events[0].AitoolsMcpEvent)
	require.Len(t, events[0].AitoolsMcpEvent.Servers, 1)
	assert.Equal(t, protos.AitoolsTargetOutcomeFailed, events[0].AitoolsMcpEvent.Servers[0].Outcome)
	assert.Equal(t, protos.AitoolsMcpSignInStatePending, events[0].AitoolsMcpEvent.Servers[0].SignInState)
}

// TestMcpEventOmitsUserContent checks that the event carries enums and counts only: a custom service's name must not
// appear anywhere in the payload.
func TestMcpEventOmitsUserContent(t *testing.T) {
	event := protos.AitoolsMcpEvent{
		CallingAgent:      protos.AitoolsAgentTypeClaudeCode,
		Operation:         protos.AitoolsMcpOperationAdd,
		BoundProfileCount: 1,
		Servers: addServerResults([]addServerOutput{{
			Service: "acme_catalog.secret_schema.internal_tool",
			Agents:  []addAgentOutput{{Name: "claude-code", Status: statusInstalled}},
		}}),
	}
	b, err := json.Marshal(event)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(b, &got))
	assert.ElementsMatch(t, []string{"calling_agent", "operation", "servers", "bound_profile_count"}, slices.Collect(maps.Keys(got)))
	assert.NotContains(t, string(b), "acme_catalog")
	assert.NotContains(t, string(b), "secret_schema")
	assert.NotContains(t, string(b), "internal_tool")
	assert.Contains(t, string(b), `"service":"CUSTOM"`)
}

// TestAddSignInStateWithoutAState checks that a sign-in that failed before it read a state is PENDING, as a server that
// needs sign-in is until its state is known. A server that needs no sign-in is NOT_REQUIRED.
func TestAddSignInStateWithoutAState(t *testing.T) {
	failed := addServerOutput{Service: "system.ai.github", SignInRequired: true, SignIn: &addSignInOutput{ErrorCategory: protos.AitoolsErrorCategoryUncategorized}}
	assert.Equal(t, protos.AitoolsMcpSignInStatePending, addSignInState(failed))
	assert.Empty(t, addSignInState(addServerOutput{Service: "system.ai.github", SignInRequired: true}))
	assert.Equal(t, protos.AitoolsMcpSignInStateNotRequired, addSignInState(addServerOutput{Service: "system.ai.dbsql"}))
}
