package generate_test

import (
	"testing"

	"github.com/databricks/cli/bundle/generate"
	"github.com/databricks/cli/libs/structs/structyaml"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertJobToValueOmitsDeployment(t *testing.T) {
	job := &jobs.Job{Settings: &jobs.JobSettings{
		Name: "generated-job",
		Deployment: &jobs.JobDeployment{
			Kind:             "BUNDLE",
			DeploymentId:     "123",
			VersionId:        "4",
			MetadataFilePath: "/Workspace/state/metadata.json",
		},
	}}

	value, err := generate.ConvertJobToValue(job)
	require.NoError(t, err)
	assert.Equal(t, structyaml.M("name", "generated-job"), value)
}
