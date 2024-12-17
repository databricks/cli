package bundle_test

import (
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPythonWheelTaskWithEnvironmentsDeployAndRun(t *testing.T) {
	t.Skip("Skipping test until serveless is enabled")

	ctx, _ := acc.WorkspaceTest(t)

	bundleRoot := initTestTemplate(t, ctx, "python_wheel_task_with_environments", map[string]any{
		"unique_id": uuid.New().String(),
	})

	deployBundle(t, ctx, bundleRoot)

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
