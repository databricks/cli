package direct

import (
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/stretchr/testify/require"
)

func TestPipelineANSIBackendDefaultProducesNoOpPlan(t *testing.T) {
	adapters, err := dresources.InitAll(nil)
	require.NoError(t, err)
	adapter := adapters["pipelines"]
	path := structpath.MustParsePath("configuration['spark.sql.ansi.enabled']")

	changes := deployplan.Changes{
		path.String(): &deployplan.ChangeDesc{
			Old:    nil,
			New:    nil,
			Remote: "true",
		},
	}
	require.NoError(t, addPerFieldActions(t.Context(), adapter, changes, nil, nil))
	change := changes[path.String()]
	require.Equal(t, deployplan.Skip, change.Action)
	require.Equal(t, deployplan.ReasonBackendDefault, change.Reason)

	// A second planner pass has the same classification and therefore remains a no-op.
	require.NoError(t, addPerFieldActions(t.Context(), adapter, changes, nil, nil))
	require.Equal(t, deployplan.Skip, changes[path.String()].Action)
}
