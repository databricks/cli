package protos

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAitoolsMcpEventEncodesSnakeCaseKeys checks that the lumberjack proto keys these fields in snake_case. It pins the
// wire keys of an event and of one server result, and that unset enums are omitted rather than sent empty.
func TestAitoolsMcpEventEncodesSnakeCaseKeys(t *testing.T) {
	b, err := json.Marshal(DatabricksCliLog{AitoolsMcpEvent: &AitoolsMcpEvent{
		CallingAgent:      AitoolsAgentTypeClaudeCode,
		Operation:         AitoolsMcpOperationAdd,
		BoundProfileCount: 0,
		Servers: []AitoolsMcpServerResult{
			{
				Service:       AitoolsMcpServiceGitHub,
				Agent:         AitoolsAgentTypeCodex,
				Outcome:       AitoolsTargetOutcomeFailed,
				ErrorCategory: AitoolsErrorCategoryAgentConfigInvalid,
				SignInState:   AitoolsMcpSignInStatePending,
			},
		},
	}})
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(b, &got))
	event, ok := got["aitools_mcp_event"].(map[string]any)
	require.True(t, ok, "aitools_mcp_event must be the event key")

	assert.Equal(t, "CLAUDE_CODE", event["calling_agent"])
	assert.Equal(t, "ADD", event["operation"])
	// Zero is a real count and is sent.
	assert.EqualValues(t, 0, event["bound_profile_count"])

	servers := event["servers"].([]any)
	require.Len(t, servers, 1)
	server := servers[0].(map[string]any)
	assert.Equal(t, map[string]any{
		"service":        "GITHUB",
		"agent":          "CODEX",
		"outcome":        "FAILED",
		"error_category": "AGENT_CONFIG_INVALID",
		"sign_in_state":  "PENDING",
	}, server)
}

// TestAitoolsMcpServerResultOmitsUnsetFields checks that a proxy event has no agent on the result, no outcome and no error
// category when it succeeded.
func TestAitoolsMcpServerResultOmitsUnsetFields(t *testing.T) {
	b, err := json.Marshal(AitoolsMcpServerResult{Service: AitoolsMcpServiceOtherSystem})
	require.NoError(t, err)
	assert.JSONEq(t, `{"service":"OTHER_SYSTEM"}`, string(b))
}
