package jobs

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobSettingsRoundTrip(t *testing.T) {
	input := `{"name":"x","max_concurrent_runs":0,"tasks":[{"task_key":"a","notebook_task":{"notebook_path":"/p"},"existing_cluster_id":"c1","environment_key":"e"}],"run_as":{"service_principal_name":"sp"},"format":"MULTI_TASK"}`
	var m JobSettings
	require.NoError(t, json.Unmarshal([]byte(input), &m))
	s, err := JobSettingsToSDK(&m)
	require.NoError(t, err)
	back, err := JobSettingsFromSDK(s)
	require.NoError(t, err)
	out, err := json.Marshal(back)
	require.NoError(t, err)
	assert.JSONEq(t, input, string(out))
}
