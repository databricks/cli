package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const reservedFieldMsg = " must not be set in bundle configuration; it is managed by Declarative Automation Bundles"

func jobBundle(d *jobs.JobDeployment) *bundle.Bundle {
	return &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my_job": {JobSettings: jobs.JobSettings{Name: "my_job", Deployment: d}},
				},
			},
		},
	}
}

func pipelineBundle(d *pipelines.PipelineDeployment) *bundle.Bundle {
	return &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"my_pipeline": {CreatePipeline: pipelines.CreatePipeline{Name: "my_pipeline", Deployment: d}},
				},
			},
		},
	}
}

func TestValidateDeploymentFieldsRejectsReservedFields(t *testing.T) {
	tests := []struct {
		name string
		b    *bundle.Bundle
		want string
	}{
		{"job deployment_id", jobBundle(&jobs.JobDeployment{DeploymentId: "x"}), "deployment_id" + reservedFieldMsg},
		{"job version_id", jobBundle(&jobs.JobDeployment{VersionId: "x"}), "version_id" + reservedFieldMsg},
		{"pipeline deployment_id", pipelineBundle(&pipelines.PipelineDeployment{DeploymentId: "x"}), "deployment_id" + reservedFieldMsg},
		{"pipeline version_id", pipelineBundle(&pipelines.PipelineDeployment{VersionId: "x"}), "version_id" + reservedFieldMsg},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := bundle.Apply(t.Context(), tt.b, ValidateDeploymentFields())
			require.Len(t, diags, 1)
			assert.Equal(t, diag.Error, diags[0].Severity)
			assert.Equal(t, tt.want, diags[0].Summary)
		})
	}
}

func TestValidateDeploymentFieldsReportsAllOffenders(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my_job": {JobSettings: jobs.JobSettings{Deployment: &jobs.JobDeployment{DeploymentId: "a"}}},
				},
				Pipelines: map[string]*resources.Pipeline{
					"my_pipeline": {CreatePipeline: pipelines.CreatePipeline{Deployment: &pipelines.PipelineDeployment{VersionId: "b"}}},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, ValidateDeploymentFields())
	require.Len(t, diags, 2)
}
