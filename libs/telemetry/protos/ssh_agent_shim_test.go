package protos

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A failed launch is the zero value of the event, so omitempty would drop the result and
// category and make the failure indistinguishable from an unreported one. This pins the
// wire payload for that case.
func TestSshAgentShimEventEncodesFailureExplicitly(t *testing.T) {
	b, err := json.Marshal(SshAgentShimEvent{})
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(b, &got))

	assert.Equal(t, false, got["is_success"], "is_success must be sent as false, not omitted")
	// The remaining fields must appear in the payload at all: omitempty would drop the
	// zero value and make a failure indistinguishable from an unreported one.
	for _, field := range []string{"agent", "error_category", "setup_duration_ms"} {
		assert.Contains(t, got, field, "%s must be sent, not omitted", field)
	}
}
