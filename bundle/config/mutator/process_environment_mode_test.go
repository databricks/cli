package mutator_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessEnvironmentModeApplyDebug(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Mode: config.Development,
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {JobSettings: &jobs.JobSettings{Name: "job1"}},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline1": {PipelineSpec: &pipelines.PipelineSpec{Name: "pipeline1"}},
				},
				Experiments: map[string]*resources.MlflowExperiment{
					"experiment1": {Experiment: &ml.Experiment{Name: "/Users/lennart.kats@databricks.com/experiment1"}},
					"experiment2": {Experiment: &ml.Experiment{Name: "experiment2"}},
				},
				Models: map[string]*resources.MlflowModel{
					"model1": {Model: &ml.Model{Name: "model1"}},
				},
			},
		},
	}

	m := mutator.ProcessEnvironmentMode()
	err := m.Apply(context.Background(), bundle)
	require.NoError(t, err)
	assert.Equal(t, "[dev] job1", bundle.Config.Resources.Jobs["job1"].Name)
	assert.Equal(t, "[dev] pipeline1", bundle.Config.Resources.Pipelines["pipeline1"].Name)
	assert.Equal(t, "/Users/lennart.kats@databricks.com/[dev] experiment1", bundle.Config.Resources.Experiments["experiment1"].Name)
	assert.Equal(t, "[dev] experiment2", bundle.Config.Resources.Experiments["experiment2"].Name)
	assert.Equal(t, "[dev] model1", bundle.Config.Resources.Models["model1"].Name)
	assert.Equal(t, "dev", bundle.Config.Resources.Experiments["experiment1"].Experiment.Tags[0].Key)
	assert.True(t, bundle.Config.Resources.Pipelines["pipeline1"].PipelineSpec.Development)
}

func TestProcessEnvironmentModeApplyDefault(t *testing.T) {
	bundle := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Mode: "",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {JobSettings: &jobs.JobSettings{Name: "job1"}},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline1": {PipelineSpec: &pipelines.PipelineSpec{Name: "pipeline1"}},
				},
			},
		},
	}

	m := mutator.ProcessEnvironmentMode()
	err := m.Apply(context.Background(), bundle)
	require.NoError(t, err)
	assert.Equal(t, "job1", bundle.Config.Resources.Jobs["job1"].Name)
	assert.Equal(t, "pipeline1", bundle.Config.Resources.Pipelines["pipeline1"].Name)
	assert.False(t, bundle.Config.Resources.Pipelines["pipeline1"].PipelineSpec.Development)
}
