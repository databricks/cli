package bundle

import (
	"context"
	"testing"

	"github.com/databricks/cli/internal"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccLocalStateStaleness(t *testing.T) {
	env := internal.GetEnvOrSkipTest(t, "CLOUD_ENV")
	t.Log(env)

	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	// The approach for this test is as follows:
	// 1) First deploy of bundle instance A
	// 2) First deploy of bundle instance B
	// 3) Second deploy of bundle instance A
	// Because of deploy (2), the locally cached state of bundle instance A should be stale.
	// Then for deploy (3), it must use the remote state over the stale local state.

	nodeTypeId := internal.GetNodeTypeId(env)
	uniqueId := uuid.New().String()
	initialize := func() string {
		root, err := initTestTemplate(t, "basic", map[string]any{
			"unique_id":     uniqueId,
			"node_type_id":  nodeTypeId,
			"spark_version": "13.2.x-snapshot-scala2.12",
		})
		require.NoError(t, err)

		t.Cleanup(func() {
			err = destroyBundle(t, root)
			require.NoError(t, err)
		})

		return root
	}

	bundleA := initialize()
	bundleB := initialize()

	// 1) Deploy bundle A
	err = deployBundle(t, bundleA)
	require.NoError(t, err)

	// 2) Deploy bundle B
	err = deployBundle(t, bundleB)
	require.NoError(t, err)

	// 3) Deploy bundle A again
	err = deployBundle(t, bundleA)
	require.NoError(t, err)

	// Assert that there is only a single job in the workspace corresponding to this bundle.
	iter := w.Jobs.List(context.Background(), jobs.ListJobsRequest{
		Name: "test-job-basic-" + uniqueId,
	})
	jobs, err := listing.ToSlice(context.Background(), iter)
	require.NoError(t, err)
	assert.Len(t, jobs, 1)
}
