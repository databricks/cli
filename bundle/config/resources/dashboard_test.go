package resources

import (
	"encoding/json"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/stretchr/testify/assert"
)

func TestDashboardConfigMarshalJSON(t *testing.T) {
	config := DashboardConfig{
		Dashboard: dashboards.Dashboard{
			DisplayName:         "test",
			SerializedDashboard: `{"a": "b"}`,
		},
		SerializedDashboard: map[string]any{"c": "d"},
		EmbedCredentials:    true,
	}

	b, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal dashboard config: %v", err)
	}

	assert.Contains(t, string(b), `"display_name":"test"`)
	assert.Contains(t, string(b), `"embed_credentials":true`)
	assert.Contains(t, string(b), `"serialized_dashboard":{"c":"d"}`)
}

func TestDashboardConfigUnmarshalJSON(t *testing.T) {
	b := []byte(`{"display_name":"test","warehouse_id":"","serialized_dashboard":{"a": "b"},"embed_credentials":true}`)

	expectedConfig := DashboardConfig{
		Dashboard: dashboards.Dashboard{
			DisplayName:     "test",
			ForceSendFields: nil,
		},
		SerializedDashboard: map[string]any{"a": "b"},
		EmbedCredentials:    true,
		ForceSendFields:     []string{"EmbedCredentials"},
	}

	var config DashboardConfig
	err := json.Unmarshal(b, &config)
	assert.NoError(t, err)
	assert.Equal(t, expectedConfig, config)
}
