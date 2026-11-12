package protos

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The lumberjack proto keys fields in snake_case; a wrong tag is dropped
// silently on ingestion. This pins the wire keys of the aitools hint event.
func TestAitoolsEventsWireKeys(t *testing.T) {
	b, err := json.Marshal(DatabricksCliLog{
		AitoolsHintEvent: &AitoolsHintEvent{HintKind: AitoolsHintKindInstallText},
	})
	require.NoError(t, err)
	assert.JSONEq(t, `{"aitools_hint_event":{"hint_kind":"INSTALL_TEXT"}}`, string(b))
}
