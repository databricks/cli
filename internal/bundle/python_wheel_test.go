package bundle

import (
	"testing"

	"github.com/databricks/cli/internal"
	"github.com/databricks/cli/internal/acc"
	"github.com/databricks/cli/libs/env"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func runPythonWheelTest(t *testing.T, sparkVersion string, pythonWheelWrapper bool) {
	ctx, _ := acc.WorkspaceTest(t)

	nodeTypeId := internal.GetNodeTypeId(env.Get(ctx, "CLOUD_ENV"))
	bundleRoot, err := initTestTemplate(t, ctx, "python_wheel_task", map[string]any{
		"node_type_id":         nodeTypeId,
		"unique_id":            uuid.New().String(),
		"spark_version":        sparkVersion,
		"python_wheel_wrapper": pythonWheelWrapper,
	})
	require.NoError(t, err)

	err = deployBundle(t, ctx, bundleRoot)
	require.NoError(t, err)

	t.Cleanup(func() {
		destroyBundle(t, ctx, bundleRoot)
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
	runPythonWheelTest(t, "13.2.x-snapshot-scala2.12", false)
}

func TestAccPythonWheelTaskDeployAndRunWithWrapper(t *testing.T) {
	runPythonWheelTest(t, "12.2.x-scala2.12", true)
}
