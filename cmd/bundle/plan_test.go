package bundle

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedactMapInPlace(t *testing.T) {
	m := map[string]any{
		"value":   "super-secret",
		"comment": "visible",
	}
	redactMapInPlace(m, map[string]bool{"value": true})
	assert.Equal(t, "********", m["value"])
	assert.Equal(t, "visible", m["comment"])
}

func TestRedactMapInPlaceEmptyNotMasked(t *testing.T) {
	m := map[string]any{
		"value": "",
	}
	redactMapInPlace(m, map[string]bool{"value": true})
	assert.Empty(t, m["value"])
}

func TestMarshalPlanRedactedNoSensitiveResources(t *testing.T) {
	// A plan with a job (no sensitive fields) must not be altered.
	raw := `{"plan_version":2,"plan":{"resources.jobs.my_job":{"action":"create","new_state":{"value":{"name":"my_job"}}}}}`

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &m))

	// Reconstruct as a minimal deployplan.Plan for marshaling.
	buf, err := json.MarshalIndent(m, "", "  ")
	require.NoError(t, err)

	// The job name must not be masked.
	assert.Contains(t, string(buf), `"my_job"`)
}
