<<<<<<< HEAD
package mutator

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockBundle(mode config.Mode) *bundle.Bundle {
	return &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Mode: mode,
			},
			Workspace: config.Workspace{
				CurrentUser: &config.User{
					ShortName: "Lennart",
					User: &iam.User{
						UserName: "lennart@company.com",
						Id:       "1",
					},
				},
				StatePath:     "/Users/lennart@company.com/.bundle/x/y/state",
				ArtifactsPath: "/Users/lennart@company.com/.bundle/x/y/artifacts",
				FilesPath:     "/Users/lennart@company.com/.bundle/x/y/files",
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
}

func TestProcessEnvironmentModeDevelopment(t *testing.T) {
	bundle := mockBundle(config.Development)

	m := ProcessEnvironmentMode()
	err := m.Apply(context.Background(), bundle)
	require.NoError(t, err)
	assert.Equal(t, "[dev Lennart] job1", bundle.Config.Resources.Jobs["job1"].Name)
	assert.Equal(t, "[dev Lennart] pipeline1", bundle.Config.Resources.Pipelines["pipeline1"].Name)
	assert.Equal(t, "/Users/lennart.kats@databricks.com/[dev Lennart] experiment1", bundle.Config.Resources.Experiments["experiment1"].Name)
	assert.Equal(t, "[dev Lennart] experiment2", bundle.Config.Resources.Experiments["experiment2"].Name)
	assert.Equal(t, "[dev Lennart] model1", bundle.Config.Resources.Models["model1"].Name)
	assert.Equal(t, "dev", bundle.Config.Resources.Experiments["experiment1"].Experiment.Tags[0].Key)
	assert.True(t, bundle.Config.Resources.Pipelines["pipeline1"].PipelineSpec.Development)
}

func TestProcessEnvironmentModeDefault(t *testing.T) {
	bundle := mockBundle("")

	m := ProcessEnvironmentMode()
	err := m.Apply(context.Background(), bundle)
	require.NoError(t, err)
	assert.Equal(t, "job1", bundle.Config.Resources.Jobs["job1"].Name)
	assert.Equal(t, "pipeline1", bundle.Config.Resources.Pipelines["pipeline1"].Name)
	assert.False(t, bundle.Config.Resources.Pipelines["pipeline1"].PipelineSpec.Development)
}

func TestProcessEnvironmentModeProduction(t *testing.T) {
	bundle := mockBundle(config.Production)
	bundle.Config.Workspace.StatePath = "/Shared/.bundle/x/y/state"
	bundle.Config.Workspace.ArtifactsPath = "/Shared/.bundle/x/y/artifacts"
	bundle.Config.Workspace.FilesPath = "/Shared/.bundle/x/y/files"

	err := validateProductionMode(context.Background(), bundle, false)

	require.NoError(t, err)
	assert.Equal(t, "job1", bundle.Config.Resources.Jobs["job1"].Name)
	assert.Equal(t, "pipeline1", bundle.Config.Resources.Pipelines["pipeline1"].Name)
	assert.False(t, bundle.Config.Resources.Pipelines["pipeline1"].PipelineSpec.Development)
}

func TestProcessEnvironmentModeProductionFails(t *testing.T) {
	bundle := mockBundle(config.Production)

	err := validateProductionMode(context.Background(), bundle, false)

	require.Error(t, err)
}

func TestProcessEnvironmentModeProductionOkForPrincipal(t *testing.T) {
	bundle := mockBundle(config.Production)

	err := validateProductionMode(context.Background(), bundle, false)

	require.NoError(t, err)
}

// Make sure that we have test coverage for all resource types
func TestAllResourcesMocked(t *testing.T) {
	bundle := mockBundle(config.Development)
	resources := reflect.ValueOf(bundle.Config.Resources)

	for i := 0; i < resources.NumField(); i++ {
		field := resources.Field(i)
		if field.Kind() == reflect.Map {
			assert.True(
				t,
				!field.IsNil() && field.Len() > 0,
				"process_environment_mode should support '%s' (please add it to process_environment_mode.go and extend the test suite)",
				resources.Type().Field(i).Name,
			)
		}
	}
}

// Make sure that we at least rename all resources
func TestAllResourcesRenamed(t *testing.T) {
	bundle := mockBundle(config.Development)
	resources := reflect.ValueOf(bundle.Config.Resources)

	m := ProcessEnvironmentMode()
	err := m.Apply(context.Background(), bundle)
	require.NoError(t, err)

	for i := 0; i < resources.NumField(); i++ {
		field := resources.Field(i)

		if field.Kind() == reflect.Map {
			for _, key := range field.MapKeys() {
				resource := field.MapIndex(key)
				nameField := resource.Elem().FieldByName("Name")
				if nameField.IsValid() && nameField.Kind() == reflect.String {
					assert.True(
						t,
						strings.Contains(nameField.String(), "dev"),
						"process_environment_mode should rename '%s' in '%s'",
						key,
						resources.Type().Field(i).Name,
					)
				}
			}
		}
	}
}
||||||| 3354750
=======
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
>>>>>>> databricks/main
