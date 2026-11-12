package protos

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The lumberjack proto keys fields in snake_case; a wrong tag is dropped
// silently on ingestion. This pins the wire keys of the new aitools events,
// including the measurable zeros that must not be omitted.
func TestAitoolsEventsWireKeys(t *testing.T) {
	b, err := json.Marshal(DatabricksCliLog{
		AitoolsHintEvent: &AitoolsHintEvent{HintKind: AitoolsHintKindInstallText},
	})
	require.NoError(t, err)
	assert.JSONEq(t, `{"aitools_hint_event":{"hint_kind":"INSTALL_TEXT"}}`, string(b))

	b, err = json.Marshal(DatabricksCliLog{
		AitoolsCommandEvent: &AitoolsCommandEvent{
			Command: AitoolsCommandList,
			Stage:   AitoolsCommandStageStarted,
		},
	})
	require.NoError(t, err)
	assert.JSONEq(t, `{"aitools_command_event":{"command":"LIST","stage":"STARTED","interactive":false}}`, string(b))

	b, err = json.Marshal(AitoolsCommandEvent{
		Command:       AitoolsCommandInstall,
		Stage:         AitoolsCommandStageFinished,
		Outcome:       AitoolsCommandOutcomeFailure,
		ErrorCategory: AitoolsErrorCategoryNetwork,
		Interactive:   true,
		InstallMode:   AitoolsInstallModeDefault,
		Agents:        []AitoolsAgentType{AitoolsAgentTypeCodex},
		Scope:         AitoolsInstallScopeGlobal,
	})
	require.NoError(t, err)
	assert.JSONEq(t, `{"command":"INSTALL","stage":"FINISHED","outcome":"FAILURE","error_category":"NETWORK",
		"interactive":true,"install_mode":"DEFAULT","agents":["CODEX"],"scope":"GLOBAL"}`, string(b))
}
