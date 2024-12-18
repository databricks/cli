package bundle_test

import (
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/env"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func runPythonWheelTest(t *testing.T, templateName, sparkVersion string, pythonWheelWrapper bool) {
	ctx, _ := acc.WorkspaceTest(t)

	nodeTypeId := testutil.GetCloud(t).NodeTypeID()
	instancePoolId := env.Get(ctx, "TEST_INSTANCE_POOL_ID")
	bundleRoot := initTestTemplate(t, ctx, templateName, map[string]any{
		"node_type_id":         nodeTypeId,
		"unique_id":            uuid.New().String(),
		"spark_version":        sparkVersion,
		"python_wheel_wrapper": pythonWheelWrapper,
		"instance_pool_id":     instancePoolId,
	})

	deployBundle(t, ctx, bundleRoot)

	t.Cleanup(func() {
		destroyBundle(t, ctx, bundleRoot)
	})

	if testing.Short() {
		t.Log("Skip the job run in short mode")
		return
	}

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

func TestPythonWheelTaskDeployAndRunWithoutWrapper(t *testing.T) {
	runPythonWheelTest(t, "python_wheel_task", "13.3.x-snapshot-scala2.12", false)
}

func TestPythonWheelTaskDeployAndRunWithWrapper(t *testing.T) {
	runPythonWheelTest(t, "python_wheel_task", "12.2.x-scala2.12", true)
}

func TestPythonWheelTaskDeployAndRunOnInteractiveCluster(t *testing.T) {
	if testutil.GetCloud(t) == testutil.AWS {
		t.Skip("Skipping test for AWS cloud because it is not permitted to create clusters")
	}

	runPythonWheelTest(t, "python_wheel_task_with_cluster", defaultSparkVersion, false)
}
