package prompt_test

import (
	"testing"

	"github.com/databricks/cli/libs/apps/prompt"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListModelServicesInSchema(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), m.WorkspaceClient)

	m.GetMockAiGatewayAPI().EXPECT().
		ListModelServicesAll(ctx, catalog.ListModelServicesRequest{Parent: "schemas/main.default"}).
		Return([]catalog.ModelService{
			{Name: "model-services/main.default.chat"},
			{Name: "model-services/main.default.embed"},
		}, nil)

	items, err := prompt.ListModelServicesInSchema(ctx, "main", "default")
	require.NoError(t, err)
	assert.Equal(t, []prompt.ListItem{
		{ID: "main.default.chat", Label: "main.default.chat"},
		{ID: "main.default.embed", Label: "main.default.embed"},
	}, items)
}
