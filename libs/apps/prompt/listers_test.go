package prompt

import (
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListModelServicesInSchema(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	ctx := cmdctx.SetWorkspaceClient(cmdio.MockDiscard(t.Context()), m.WorkspaceClient)

	m.GetMockAiGatewayAPI().EXPECT().
		ListModelServices(ctx, catalog.ListModelServicesRequest{Parent: "schemas/main.default"}).
		Return(&listing.SliceIterator[catalog.ModelService]{
			{Name: "model-services/main.default.claude"},
			{Name: "model-services/main.default.gpt"},
		})

	items, err := ListModelServicesInSchema(ctx, "main", "default")
	require.NoError(t, err)
	assert.Equal(t, []ListItem{
		{ID: "main.default.claude", Label: "claude"},
		{ID: "main.default.gpt", Label: "gpt"},
	}, items)
}
