package generate_test

import (
	"testing"

	"github.com/databricks/cli/bundle/generate"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertPipelineToValueOmitsDeployment(t *testing.T) {
	pipeline := &pipelines.PipelineSpec{
		Name: "generated-pipeline",
		Deployment: &pipelines.PipelineDeployment{
			Kind:             "BUNDLE",
			DeploymentId:     "123",
			VersionId:        "4",
			MetadataFilePath: "/Workspace/state/metadata.json",
		},
	}

	value, err := generate.ConvertPipelineToValue(pipeline, "", "")
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"name": "generated-pipeline"}, value.AsAny())
}
