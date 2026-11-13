package aitools

import (
	"encoding/json"
	"errors"
	"maps"
	"path/filepath"
	"slices"
	"testing"

	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
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
			err:  failure(protos.AitoolsErrorCategoryMCPServiceNotFound, nil, "not found"),
			want: protos.AitoolsMcpServerResult{
				Service: protos.AitoolsMcpServiceGitHub, Outcome: protos.AitoolsTargetOutcomeFailed,
				ErrorCategory: protos.AitoolsErrorCategoryMCPServiceNotFound,
			},
		},
		{
			name:  "timed out",
			state: signInStateTimeout,
			err:   failure(protos.AitoolsErrorCategorySignInTimeout, nil, "timed out"),
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

// TestLogMcpEventDropsRejectedCredential checks that an event that names a rejected credential is not logged, and that the
// run then uploads nothing: its upload would authenticate with that credential again. The events of the run that were
// logged before stay buffered, and a later event of the run is logged, but not uploaded.
func TestLogMcpEventDropsRejectedCredential(t *testing.T) {
	withHome(t)
	ctx := env.Set(telemetry.WithNewLogger(t.Context()), "DATABRICKS_CLI_DISABLE_TELEMETRY", "")
	telemetry.SetUploadConfig(ctx, &config.Config{Profile: "logfood", Host: "https://adb-1.test"})
	assert.False(t, logMcpEvent(ctx, protos.AitoolsMcpEvent{
		Operation: protos.AitoolsMcpOperationSignIn,
		Servers: []protos.AitoolsMcpServerResult{{
			Service:       protos.AitoolsMcpServiceGitHub,
			Outcome:       protos.AitoolsTargetOutcomeFailed,
			ErrorCategory: protos.AitoolsErrorCategoryAuthFailed,
		}},
	}))
	assert.Empty(t, telemetry.BufferedEvents(ctx))
	_, uploads := telemetry.UploadConfig(ctx)
	assert.False(t, uploads)

	assert.True(t, logMcpEvent(ctx, protos.AitoolsMcpEvent{
		Operation: protos.AitoolsMcpOperationAdd,
		Servers: []protos.AitoolsMcpServerResult{{
			Service:     protos.AitoolsMcpServiceGitHub,
			Outcome:     protos.AitoolsTargetOutcomeChanged,
			SignInState: protos.AitoolsMcpSignInStatePending,
		}},
	}))
	assert.Len(t, telemetry.BufferedEvents(ctx), 1)
	_, uploads = telemetry.UploadConfig(ctx)
	assert.False(t, uploads, "the add event is logged, and it is not uploaded with the rejected run")
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
	assert.Equal(t, protos.AitoolsMcpSignInStateNotRequired, addSignInState(addServerOutput{Service: "system.ai.dbsql"}))
}
