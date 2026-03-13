package jobs

import (
	"testing"

	"github.com/databricks/cli/libs/tableview"
	sdkjobs "github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListTableConfig(t *testing.T) {
	cmd := newList()

	cfg := tableview.GetConfig(cmd)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Columns, 2)

	tests := []struct {
		name     string
		job      sdkjobs.BaseJob
		wantID   string
		wantName string
	}{
		{
			name: "with settings",
			job: sdkjobs.BaseJob{
				JobId:    123,
				Settings: &sdkjobs.JobSettings{Name: "test-job"},
			},
			wantID:   "123",
			wantName: "test-job",
		},
		{
			name: "nil settings",
			job: sdkjobs.BaseJob{
				JobId: 456,
			},
			wantID:   "456",
			wantName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantID, cfg.Columns[0].Extract(tt.job))
			assert.Equal(t, tt.wantName, cfg.Columns[1].Extract(tt.job))
		})
	}
}
