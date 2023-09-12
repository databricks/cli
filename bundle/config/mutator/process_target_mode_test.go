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
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mockBundle(mode config.Mode) *bundle.Bundle {
	return &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Mode: mode,
				Git: config.Git{
					OriginURL: "http://origin",
					Branch:    "main",
				},
			},
			Workspace: config.Workspace{
				CurrentUser: &config.User{
					ShortName: "lennart",
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
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"servingendpoint1": {CreateServingEndpoint: &serving.CreateServingEndpoint{Name: "servingendpoint1"}},
				},
			},
		},
	}
}

func TestProcessTargetModeDevelopment(t *testing.T) {
	bundle := mockBundle(config.Development)

	m := ProcessTargetMode()
	err := m.Apply(context.Background(), bundle)
	require.NoError(t, err)
	assert.Equal(t, "[dev lennart] job1", bundle.Config.Resources.Jobs["job1"].Name)
	assert.Equal(t, "[dev lennart] pipeline1", bundle.Config.Resources.Pipelines["pipeline1"].Name)
	assert.Equal(t, "/Users/lennart.kats@databricks.com/[dev lennart] experiment1", bundle.Config.Resources.Experiments["experiment1"].Name)
	assert.Equal(t, "[dev lennart] experiment2", bundle.Config.Resources.Experiments["experiment2"].Name)
	assert.Equal(t, "[dev lennart] model1", bundle.Config.Resources.Models["model1"].Name)
	assert.Equal(t, "dev_lennart_servingendpoint1", bundle.Config.Resources.ModelServingEndpoints["servingendpoint1"].Name)
	assert.Equal(t, "dev", bundle.Config.Resources.Experiments["experiment1"].Experiment.Tags[0].Key)
	assert.True(t, bundle.Config.Resources.Pipelines["pipeline1"].PipelineSpec.Development)
}

func TestProcessTargetModeDefault(t *testing.T) {
	bundle := mockBundle("")

	m := ProcessTargetMode()
	err := m.Apply(context.Background(), bundle)
	require.NoError(t, err)
	assert.Equal(t, "job1", bundle.Config.Resources.Jobs["job1"].Name)
	assert.Equal(t, "pipeline1", bundle.Config.Resources.Pipelines["pipeline1"].Name)
	assert.False(t, bundle.Config.Resources.Pipelines["pipeline1"].PipelineSpec.Development)
	assert.Equal(t, "servingendpoint1", bundle.Config.Resources.ModelServingEndpoints["servingendpoint1"].Name)
}

func TestProcessTargetModeProduction(t *testing.T) {
	bundle := mockBundle(config.Production)

	err := validateProductionMode(context.Background(), bundle, false)
	require.ErrorContains(t, err, "state_path")

	bundle.Config.Workspace.StatePath = "/Shared/.bundle/x/y/state"
	bundle.Config.Workspace.ArtifactsPath = "/Shared/.bundle/x/y/artifacts"
	bundle.Config.Workspace.FilesPath = "/Shared/.bundle/x/y/files"

	err = validateProductionMode(context.Background(), bundle, false)
	require.ErrorContains(t, err, "production")

	permissions := []resources.Permission{
		{
			Level:    "CAN_MANAGE",
			UserName: "user@company.com",
		},
	}
	bundle.Config.Resources.Jobs["job1"].Permissions = permissions
	bundle.Config.Resources.Jobs["job1"].RunAs = &jobs.JobRunAs{UserName: "user@company.com"}
	bundle.Config.Resources.Pipelines["pipeline1"].Permissions = permissions
	bundle.Config.Resources.Experiments["experiment1"].Permissions = permissions
	bundle.Config.Resources.Experiments["experiment2"].Permissions = permissions
	bundle.Config.Resources.Models["model1"].Permissions = permissions
	bundle.Config.Resources.ModelServingEndpoints["servingendpoint1"].Permissions = permissions

	err = validateProductionMode(context.Background(), bundle, false)
	require.NoError(t, err)

	assert.Equal(t, "job1", bundle.Config.Resources.Jobs["job1"].Name)
	assert.Equal(t, "pipeline1", bundle.Config.Resources.Pipelines["pipeline1"].Name)
	assert.False(t, bundle.Config.Resources.Pipelines["pipeline1"].PipelineSpec.Development)
	assert.Equal(t, "servingendpoint1", bundle.Config.Resources.ModelServingEndpoints["servingendpoint1"].Name)
}

func TestProcessTargetModeProductionOkForPrincipal(t *testing.T) {
	bundle := mockBundle(config.Production)

	// Our target has all kinds of problems when not using service principals ...
	err := validateProductionMode(context.Background(), bundle, false)
	require.Error(t, err)

	// ... but we're much less strict when a principal is used
	err = validateProductionMode(context.Background(), bundle, true)
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
				"process_target_mode should support '%s' (please add it to process_target_mode.go and extend the test suite)",
				resources.Type().Field(i).Name,
			)
		}
	}
}

// Make sure that we at least rename all resources
func TestAllResourcesRenamed(t *testing.T) {
	bundle := mockBundle(config.Development)
	resources := reflect.ValueOf(bundle.Config.Resources)

	m := ProcessTargetMode()
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
						"process_target_mode should rename '%s' in '%s'",
						key,
						resources.Type().Field(i).Name,
					)
				}
			}
		}
	}
}
