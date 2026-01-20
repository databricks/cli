package dresources

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDashboardState_JSONSerialization_PublishedField(t *testing.T) {
	state := &DashboardState{
		DashboardConfig: resources.DashboardConfig{
			DisplayName: "test-dashboard",
			WarehouseId: "warehouse123",
		},
		Published: true,
	}

	data, err := json.Marshal(state)
	require.NoError(t, err)

	assert.Contains(t, string(data), `"published":true`)
}
