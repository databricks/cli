package serving_endpoints

import (
	"testing"

	"github.com/databricks/cli/libs/tableview"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListTableConfig(t *testing.T) {
	cmd := newList()

	cfg := tableview.GetTableConfigForCmd(cmd)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Columns, 3)

	tests := []struct {
		name        string
		endpoint    serving.ServingEndpoint
		wantName    string
		wantState   string
		wantCreator string
	}{
		{
			name: "with state",
			endpoint: serving.ServingEndpoint{
				Name:    "endpoint",
				Creator: "user@example.com",
				State: &serving.EndpointState{
					Ready: serving.EndpointStateReadyReady,
				},
			},
			wantName:    "endpoint",
			wantState:   "READY",
			wantCreator: "user@example.com",
		},
		{
			name: "nil state",
			endpoint: serving.ServingEndpoint{
				Name:    "endpoint",
				Creator: "user@example.com",
			},
			wantName:    "endpoint",
			wantState:   "",
			wantCreator: "user@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantName, cfg.Columns[0].Extract(tt.endpoint))
			assert.Equal(t, tt.wantState, cfg.Columns[1].Extract(tt.endpoint))
			assert.Equal(t, tt.wantCreator, cfg.Columns[2].Extract(tt.endpoint))
		})
	}
}
