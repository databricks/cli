package protos

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The lumberjack proto keys these fields as snake_case error_category, both on
// the event and on each agent_results entry. A hand-edit to a camelCase tag
// marshals under the wrong key and is dropped silently on ingestion, so this
// pins the wire payload for an event with one per-agent result.
func TestAitoolsInstallEventEncodesErrorCategoryKeys(t *testing.T) {
	b, err := json.Marshal(AitoolsInstallEvent{
		Agents:        []AitoolsAgentType{AitoolsAgentTypeCodex},
		Scope:         AitoolsInstallScopeGlobal,
		ErrorCategory: AitoolsErrorCategoryPluginInstallFailed,
		AgentResults: []AitoolsAgentResult{
			{Agent: AitoolsAgentTypeCodex, ErrorCategory: AitoolsErrorCategoryPluginInstallFailed},
		},
	})
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(b, &got))

	assert.Equal(t, "PLUGIN_INSTALL_FAILED", got["error_category"])

	results, ok := got["agent_results"].([]any)
	require.True(t, ok, "agent_results must be present")
	require.Len(t, results, 1)
	entry := results[0].(map[string]any)
	assert.Equal(t, "CODEX", entry["agent"])
	assert.Equal(t, "PLUGIN_INSTALL_FAILED", entry["error_category"])
}
