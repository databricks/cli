package pipelines

import (
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/tableview"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	sdkpipelines "github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestLooksLikeUUID(t *testing.T) {
	assert.True(t, looksLikeUUID("a12cd3e4-0ab1-1abc-1a2b-1a2bcd3e4f05"))
}

func TestLooksLikeUUID_resourceName(t *testing.T) {
	assert.False(t, looksLikeUUID("my-pipeline-key"))
}

func TestListPipelinesTableConfig(t *testing.T) {
	cmd := newListPipelines()

	cfg := tableview.GetTableConfigForCmd(cmd)
	require.NotNil(t, cfg)
	require.Len(t, cfg.Columns, 3)
	require.NotNil(t, cfg.Search)

	pipeline := sdkpipelines.PipelineStateInfo{
		PipelineId: "pipeline-id",
		Name:       "pipeline-name",
		State:      sdkpipelines.PipelineStateIdle,
	}

	assert.Equal(t, "pipeline-id", cfg.Columns[0].Extract(pipeline))
	assert.Equal(t, "pipeline-name", cfg.Columns[1].Extract(pipeline))
	assert.Equal(t, "IDLE", cfg.Columns[2].Extract(pipeline))
}

func TestListPipelinesSearchEscapesLikeWildcards(t *testing.T) {
	cmd := newListPipelines()

	cfg := tableview.GetTableConfigForCmd(cmd)
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.Search)

	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockPipelinesAPI().EXPECT().
		ListPipelines(mock.Anything, sdkpipelines.ListPipelinesRequest{
			Filter: "name LIKE '%foo''\\%\\_bar%'",
		}).
		Return(nil)

	ctx := cmdctx.SetWorkspaceClient(t.Context(), m.WorkspaceClient)
	assert.NotNil(t, cfg.Search.NewIterator(ctx, "foo'%_bar"))
}

func TestListPipelinesSearchDisabledWhenFilterSet(t *testing.T) {
	cmd := newListPipelines()

	err := cmd.Flags().Set("filter", "state = 'RUNNING'")
	require.NoError(t, err)

	cfg := tableview.GetTableConfigForCmd(cmd)
	require.NotNil(t, cfg)
	require.NotNil(t, cfg.Search)

	// Simulate context setup so disableSearchIfFilterSet can read the config.
	cmd.SetContext(tableview.SetTableConfig(t.Context(), cfg))
	disableSearchIfFilterSet(cmd)
	assert.Nil(t, cfg.Search)
}

func TestListPipelinesSearchNotDisabledWithoutFilter(t *testing.T) {
	cmd := newListPipelines()

	cfg := tableview.GetTableConfigForCmd(cmd)
	require.NotNil(t, cfg)

	// Simulate context setup so disableSearchIfFilterSet can read the config.
	cmd.SetContext(tableview.SetTableConfig(t.Context(), cfg))
	disableSearchIfFilterSet(cmd)
	assert.NotNil(t, cfg.Search)
}
