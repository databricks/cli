package bundle

import (
	"testing"

	"github.com/databricks/cli/internal"
	"github.com/databricks/cli/internal/acc"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/env"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func runPythonWheelTest(t *testing.T, templateName, sparkVersion string, pythonWheelWrapper bool) {
	ctx, _ := acc.WorkspaceTest(t)

	nodeTypeId := internal.GetNodeTypeId(env.Get(ctx, "CLOUD_ENV"))
	instancePoolId := env.Get(ctx, "TEST_INSTANCE_POOL_ID")
	bundleRoot, err := initTestTemplate(t, ctx, templateName, map[string]any{
		"node_type_id":         nodeTypeId,
		"unique_id":            uuid.New().String(),
		"spark_version":        sparkVersion,
		"python_wheel_wrapper": pythonWheelWrapper,
		"instance_pool_id":     instancePoolId,
	})
	require.NoError(t, err)

	err = deployBundle(t, ctx, bundleRoot)
	require.NoError(t, err)

	t.Cleanup(func() {
		err := destroyBundle(t, ctx, bundleRoot)
		require.NoError(t, err)
	})

	out, err := runResource(t, ctx, bundleRoot, "some_other_job")
	require.NoError(t, err)
	require.Contains(t, out, "Hello from my func")
	require.Contains(t, out, "Got arguments:")
	require.Contains(t, out, "['my_test_code', 'one', 'two']")

	out, err = runResourceWithParams(t, ctx, bundleRoot, "some_other_job", "--python-params=param1,param2")
	require.NoError(t, err)
	require.Contains(t, out, "Hello from my func")
	require.Contains(t, out, "Got arguments:")
	require.Contains(t, out, "['my_test_code', 'param1', 'param2']")
}

func TestAccPythonWheelTaskDeployAndRunWithoutWrapper(t *testing.T) {
	runPythonWheelTest(t, "python_wheel_task", "13.3.x-snapshot-scala2.12", false)
}

func TestAccPythonWheelTaskDeployAndRunWithWrapper(t *testing.T) {
	runPythonWheelTest(t, "python_wheel_task", "12.2.x-scala2.12", true)
}

func TestAccPythonWheelTaskDeployAndRunOnInteractiveCluster(t *testing.T) {
	_, wt := acc.WorkspaceTest(t)

	if testutil.IsAWSCloud(wt.T) {
		t.Skip("Skipping test for AWS cloud because it is not permitted to create clusters")
	}

	runPythonWheelTest(t, "python_wheel_task_with_cluster", defaultSparkVersion, false)
}
