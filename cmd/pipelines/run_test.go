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
			expected: `Update for pipeline test-pipeline completed successfully.
Pipeline ID: pipeline-789
Update start time: 2022-01-01T00:00:00Z
Update end time: 2022-01-01T01:00:00Z.
Pipeline configurations for this update:
• All tables are fully refreshed
• Update cause: Manual trigger
• Serverless compute
• Channel: CURRENT
• Continuous pipeline
• Development mode
• Catalog: test_catalog
• Schema: test_schema
`,
		},
		{
			name: "minimal pipeline update with default refresh",
			update: pipelines.UpdateInfo{
				UpdateId: "update-456",
			},
			pipelineID: "pipeline-789",
			events:     []pipelines.PipelineEvent{},
			expected: `Update for pipeline completed successfully.
Pipeline configurations for this update:
• All tables are refreshed
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
			expected: `Update for pipeline completed successfully.
Pipeline configurations for this update:
• Refreshed [table1, table2]
• Full refreshed [table3]
`,
		},
		{
			name: "pipeline update with storage instead of catalog/schema",
			update: pipelines.UpdateInfo{
				UpdateId: "update-storage",
				Config: &pipelines.PipelineSpec{
					Name:    "test-pipeline",
					Id:      "pipeline-789",
					Storage: "test_storage",
				},
			},
			pipelineID: "pipeline-456",
			events:     []pipelines.PipelineEvent{},
			expected: `Update for pipeline test-pipeline completed successfully.
Pipeline ID: pipeline-789
Pipeline configurations for this update:
• All tables are refreshed
• Storage: test_storage
`,
		},
		{
			name: "pipeline update with classic compute and no config",
			update: pipelines.UpdateInfo{
				UpdateId:  "update-classic",
				ClusterId: "cluster-123",
			},
			pipelineID: "pipeline-456",
			events:     []pipelines.PipelineEvent{},
			expected: `Update for pipeline completed successfully.
Pipeline configurations for this update:
• All tables are refreshed
• Classic compute: cluster-123
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
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, buf.String())
		})
	}
}

func TestDisplayProgressEvents(t *testing.T) {
	tests := []struct {
		name     string
		events   []pipelines.PipelineEvent
		expected string
		wantErr  bool
	}{
		{
			name: "pipeline completed with multiple phases and different duration formats",
			events: []pipelines.PipelineEvent{
				{
					Timestamp: "2022-01-01T00:00:00.000Z",
					EventType: "update_progress",
					Message:   "Update test-update-123 is WAITING_FOR_RESOURCES.",
				},
				{
					Timestamp: "2022-01-01T00:00:01.500Z",
					EventType: "update_progress",
					Message:   "Update test-update-123 is RUNNING.",
				},
				{
					Timestamp: "2022-01-01T00:01:30Z",
					EventType: "update_progress",
					Message:   "Update test-update-123 is COMPLETED.",
				},
			},
			expected: `Run Phase                 Duration
---------                 --------
WAITING_FOR_RESOURCES     1.5s
RUNNING                   1m 28s
`,
		},
		{
			name: "pipeline with millisecond and decimal second durations",
			events: []pipelines.PipelineEvent{
				{
					Timestamp: "2022-01-01T00:00:00.000Z",
					EventType: "update_progress",
					Message:   "Update test-update-ms is WAITING_FOR_RESOURCES.",
				},
				{
					Timestamp: "2022-01-01T00:00:00.500Z",
					EventType: "update_progress",
					Message:   "Update test-update-ms is RUNNING.",
				},
				{
					Timestamp: "2022-01-01T00:00:01.250Z",
					EventType: "update_progress",
					Message:   "Update test-update-ms is COMPLETED.",
				},
			},
			expected: `Run Phase                 Duration
---------                 --------
WAITING_FOR_RESOURCES     500ms
RUNNING                   750ms
`,
		},
		{
			name: "edge cases - empty events and single event",
			events: []pipelines.PipelineEvent{
				{
					Timestamp: "2022-01-01T00:00:00Z",
					EventType: "update_progress",
					Message:   "Update test-update-single is COMPLETED.",
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			ctx := context.Background()
			cmdIO := cmdio.NewIO(ctx, flags.OutputText, nil, &buf, &buf, "", "")
			ctx = cmdio.InContext(ctx, cmdIO)

			err := displayProgressEvents(ctx, tt.events)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, buf.String())
			}
		})
	}
}
