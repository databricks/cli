package bundle

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/databricks/cli/internal"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAccPythonWheelTaskDeployAndRun(t *testing.T) {
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

	id := uuid.New().String()
	bundleRoot, err := initTestTemplate(t, "python_wheel_task", map[string]any{
		"node_type_id":  nodeTypeId,
		"unique_id":     id,
		"spark_version": "13.2.x-snapshot-scala2.12",
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
	require.Contains(t, out, "['python', 'one', 'two']")
	require.Regexp(t, regexp.MustCompile(fmt.Sprintf("Directory changed successfully /Workspace/Users/.*/.bundle/%s/files", id)), out)
}
