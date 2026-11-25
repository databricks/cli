package dresources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRejectsFieldOfWrongKind(t *testing.T) {
	tests := []struct {
		name    string
		declare func(rc *ResourceLifecycleConfig)
		wantErr string
	}{
		{
			name:    "etag_field names an integer",
			declare: func(rc *ResourceLifecycleConfig) { rc.EtagField = "max_concurrent_runs" },
			wantErr: `etag_field: field "max_concurrent_runs" in *jobs.JobSettings is int, must be a string`,
		},
		{
			name:    "deployment_id_field names a struct",
			declare: func(rc *ResourceLifecycleConfig) { rc.DeploymentIDField = "deployment" },
			wantErr: `deployment_id_field: field "deployment" in *jobs.JobSettings is *jobs.JobDeployment, must be a string`,
		},
		{
			name:    "job_id_field names a string",
			declare: func(rc *ResourceLifecycleConfig) { rc.JobIDField = "name" },
			wantErr: `job_id_field: field "name" in *jobs.JobSettings is string, must be an integer`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter, err := NewAdapter(SupportedResources()["jobs"], "jobs", nil)
			require.NoError(t, err)

			rc := *adapter.resourceConfig
			tt.declare(&rc)
			adapter.resourceConfig = &rc

			assert.EqualError(t, adapter.validate(), tt.wantErr)
		})
	}
}
