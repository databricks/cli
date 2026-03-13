package apps

import (
	"testing"

	"github.com/databricks/cli/libs/tableview"
	sdkapps "github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListTableConfig(t *testing.T) {
	cmd := newList()

	cfg := tableview.GetConfig(cmd)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Columns, 4)

	tests := []struct {
		name        string
		app         sdkapps.App
		wantName    string
		wantURL     string
		wantCompute string
		wantDeploy  string
	}{
		{
			name: "with nested fields",
			app: sdkapps.App{
				Name: "test-app",
				Url:  "https://example.com",
				ComputeStatus: &sdkapps.ComputeStatus{
					State: sdkapps.ComputeStateActive,
				},
				ActiveDeployment: &sdkapps.AppDeployment{
					Status: &sdkapps.AppDeploymentStatus{
						State: sdkapps.AppDeploymentStateSucceeded,
					},
				},
			},
			wantName:    "test-app",
			wantURL:     "https://example.com",
			wantCompute: "ACTIVE",
			wantDeploy:  "SUCCEEDED",
		},
		{
			name: "nil nested fields",
			app: sdkapps.App{
				Name:             "test-app",
				Url:              "https://example.com",
				ActiveDeployment: &sdkapps.AppDeployment{},
			},
			wantName:    "test-app",
			wantURL:     "https://example.com",
			wantCompute: "",
			wantDeploy:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantName, cfg.Columns[0].Extract(tt.app))
			assert.Equal(t, tt.wantURL, cfg.Columns[1].Extract(tt.app))
			assert.Equal(t, tt.wantCompute, cfg.Columns[2].Extract(tt.app))
			assert.Equal(t, tt.wantDeploy, cfg.Columns[3].Extract(tt.app))
		})
	}
}
