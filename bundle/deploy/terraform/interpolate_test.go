package terraform

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInterpolate(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Name: "example",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my_job": {
						JobSettings: jobs.JobSettings{
							Tags: map[string]string{
								"other_pipeline":         "${resources.pipelines.other_pipeline.id}",
								"other_job":              "${resources.jobs.other_job.id}",
								"other_model":            "${resources.models.other_model.id}",
								"other_experiment":       "${resources.experiments.other_experiment.id}",
								"other_model_serving":    "${resources.model_serving_endpoints.other_model_serving.id}",
								"other_registered_model": "${resources.registered_models.other_registered_model.id}",
								"other_schema":           "${resources.schemas.other_schema.id}",
								"other_volume":           "${resources.volumes.other_volume.id}",
								"other_cluster":          "${resources.clusters.other_cluster.id}",
								"other_dashboard":        "${resources.dashboards.other_dashboard.id}",
								"other_app":              "${resources.apps.other_app.id}",
								"other_sql_warehouse":    "${resources.sql_warehouses.other_sql_warehouse.id}",
							},
							Tasks: []jobs.Task{
								{
									TaskKey: "my_task",
									NotebookTask: &jobs.NotebookTask{
										BaseParameters: map[string]string{
											"model_name": "${resources.models.my_model.name}",
										},
									},
								},
							},
						},
					},
				},
				Models: map[string]*resources.MlflowModel{
					"my_model": {
						CreateModelRequest: ml.CreateModelRequest{
							Name: "my_model",
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(context.Background(), b, Interpolate())
	require.NoError(t, diags.Error())

	j := b.Config.Resources.Jobs["my_job"]
	assert.Equal(t, "${databricks_pipeline.other_pipeline.id}", j.Tags["other_pipeline"])
	assert.Equal(t, "${databricks_job.other_job.id}", j.Tags["other_job"])
	assert.Equal(t, "${databricks_mlflow_model.other_model.id}", j.Tags["other_model"])
	assert.Equal(t, "${databricks_mlflow_experiment.other_experiment.id}", j.Tags["other_experiment"])
	assert.Equal(t, "${databricks_model_serving.other_model_serving.id}", j.Tags["other_model_serving"])
	assert.Equal(t, "${databricks_registered_model.other_registered_model.id}", j.Tags["other_registered_model"])
	assert.Equal(t, "${databricks_schema.other_schema.id}", j.Tags["other_schema"])
	assert.Equal(t, "${databricks_volume.other_volume.id}", j.Tags["other_volume"])
	assert.Equal(t, "${databricks_cluster.other_cluster.id}", j.Tags["other_cluster"])
	assert.Equal(t, "${databricks_dashboard.other_dashboard.id}", j.Tags["other_dashboard"])
	assert.Equal(t, "${databricks_app.other_app.id}", j.Tags["other_app"])
	assert.Equal(t, "${databricks_sql_endpoint.other_sql_warehouse.id}", j.Tags["other_sql_warehouse"])

	m := b.Config.Resources.Models["my_model"]
	assert.Equal(t, "my_model", m.CreateModelRequest.Name)
}

func TestInterpolateUnknownResourceType(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my_job": {
						JobSettings: jobs.JobSettings{
							Tags: map[string]string{
								"other_unknown": "${resources.unknown.other_unknown.id}",
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(context.Background(), b, Interpolate())
	assert.ErrorContains(t, diags.Error(), `reference does not exist: ${resources.unknown.other_unknown.id}`)
}
