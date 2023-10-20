package bundle

import (
	"testing"

	"github.com/databricks/cli/internal"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func runPythonWheelTest(t *testing.T, sparkVersion string, pythonWheelWrapper bool) {
	env := internal.GetEnvOrSkipTest(t, "CLOUD_ENV")
	t.Log(env)

	var nodeTypeId string
	if env == "gcp" {
		nodeTypeId = "n1-standard-4"
	} else if env == "aws" {
		nodeTypeId = "i3.xlarge"
	} else {
		nodeTypeId = "Standard_DS4_v2"
	}

	bundleRoot, err := initTestTemplate(t, "python_wheel_task", map[string]any{
		"node_type_id":         nodeTypeId,
		"unique_id":            uuid.New().String(),
		"spark_version":        sparkVersion,
		"python_wheel_wrapper": pythonWheelWrapper,
	})
	require.NoError(t, err)

	err = deployBundle(t, bundleRoot)
	require.NoError(t, err)

	t.Cleanup(func() {
		destroyBundle(t, bundleRoot)
	})

	out, err := runResource(t, bundleRoot, "some_other_job")
	require.NoError(t, err)
	require.Contains(t, out, "Hello from my func")
	require.Contains(t, out, "Got arguments:")
	require.Contains(t, out, "['my_test_code', 'one', 'two']")
}

func TestAccPythonWheelTaskDeployAndRunWithoutWrapper(t *testing.T) {
	runPythonWheelTest(t, "13.2.x-snapshot-scala2.12", false)
}

func TestAccPythonWheelTaskDeployAndRunWithWrapper(t *testing.T) {
	runPythonWheelTest(t, "12.2.x-scala2.12", true)
}
