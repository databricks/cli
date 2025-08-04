package pipelines

import (
	"bytes"
	"context"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
)

func TestDisplayPipelineUpdate(t *testing.T) {
	tests := []struct {
		name       string
		update     pipelines.UpdateInfo
		pipelineID string
		events     []pipelines.PipelineEvent
		expected   string
		wantErr    bool
	}{
		{
			name: "comprehensive pipeline update with all fields",
			update: pipelines.UpdateInfo{
				UpdateId:     "update-123",
				FullRefresh:  true,
				Cause:        "Manual trigger",
				CreationTime: 1640995200000, // 2022-01-01T00:00:00Z
				ClusterId:    "cluster-456",
				Config: &pipelines.PipelineSpec{
					Name:        "test-pipeline",
					Id:          "pipeline-789",
					Serverless:  true,
					Channel:     "CURRENT",
					Continuous:  true,
					Development: true,
					Catalog:     "test_catalog",
					Schema:      "test_schema",
				},
			},
			pipelineID: "pipeline-456",
			events: []pipelines.PipelineEvent{
				{
					Timestamp: "2022-01-01T00:00:00Z",
					EventType: "update_progress",
				},
				{
					Timestamp: "2022-01-01T01:00:00Z",
					EventType: "update_progress",
				},
			},
			expected: `Update update-123 for pipeline test-pipeline pipeline-789 completed successfully.
Cause: Manual trigger
Creation Time: 2022-01-01T00:00:00Z
End Time: 2022-01-01T01:00:00Z
Compute: serverless
Refresh: full-refresh-all
Channel: CURRENT
Continuous: true
Development mode: Dev
Catalog & Schema: test_catalog.test_schema
`,
		},
		{
			name: "minimal pipeline update with default refresh",
			update: pipelines.UpdateInfo{
				UpdateId: "update-456",
			},
			pipelineID: "pipeline-789",
			events:     []pipelines.PipelineEvent{},
			expected: `Update update-456 for pipeline completed successfully.
Refresh: default refresh-all
`,
		},
		{
			name: "pipeline update with mixed refresh selections",
			update: pipelines.UpdateInfo{
				UpdateId:             "update-789",
				RefreshSelection:     []string{"table1", "table2"},
				FullRefreshSelection: []string{"table3"},
			},
			pipelineID: "pipeline-101",
			events: []pipelines.PipelineEvent{
				{
					Timestamp: "2022-01-01T02:00:00Z",
					EventType: "update_progress",
				},
			},
			expected: `Update update-789 for pipeline completed successfully.
End Time: 2022-01-01T02:00:00Z
Refresh: refreshed [table1, table2] | full-refreshed [table3]
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			ctx := context.Background()
			cmdIO := cmdio.NewIO(ctx, flags.OutputText, nil, &buf, &buf, "", "")
			ctx = cmdio.InContext(ctx, cmdIO)

			err := displayPipelineUpdate(ctx, tt.update, tt.pipelineID, tt.events)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, buf.String())
			}
		})
	}
}
