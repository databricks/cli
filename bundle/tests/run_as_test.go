package config_tests

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/stretchr/testify/assert"
)

func TestRunAsForAllowed(t *testing.T) {
	b := load(t, "./run_as/allowed")

	ctx := context.Background()
	bundle.ApplyFunc(ctx, b, func(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
		b.Config.Workspace.CurrentUser = &config.User{
			User: &iam.User{
				UserName: "jane@doe.com",
			},
		}
		return nil
	})

	diags := bundle.Apply(ctx, b, mutator.SetRunAs())
	assert.NoError(t, diags.Error())

	assert.Len(t, b.Config.Resources.Jobs, 3)
	jobs := b.Config.Resources.Jobs

	// job_one and job_two should have the same run_as identity as the bundle.
	assert.NotNil(t, jobs["job_one"].RunAs)
	assert.Equal(t, "my_service_principal", jobs["job_one"].RunAs.ServicePrincipalName)
	assert.Equal(t, "", jobs["job_one"].RunAs.UserName)

	assert.NotNil(t, jobs["job_two"].RunAs)
	assert.Equal(t, "my_service_principal", jobs["job_two"].RunAs.ServicePrincipalName)
	assert.Equal(t, "", jobs["job_two"].RunAs.UserName)

	// job_three should retain the job level run_as identity.
	assert.NotNil(t, jobs["job_three"].RunAs)
	assert.Equal(t, "my_service_principal_for_job", jobs["job_three"].RunAs.ServicePrincipalName)
	assert.Equal(t, "", jobs["job_three"].RunAs.UserName)

	// Assert other resources are not affected.
	assert.Equal(t, ml.Model{Name: "skynet"}, *b.Config.Resources.Models["model_one"].Model)
	assert.Equal(t, catalog.CreateRegisteredModelRequest{Name: "skynet (in UC)"}, *b.Config.Resources.RegisteredModels["model_two"].CreateRegisteredModelRequest)
	assert.Equal(t, ml.Experiment{Name: "experiment_one"}, *b.Config.Resources.Experiments["experiment_one"].Experiment)
}

func TestRunAsForAllowedWithTargetOverride(t *testing.T) {
	b := loadTarget(t, "./run_as/allowed", "development")

	ctx := context.Background()
	bundle.ApplyFunc(ctx, b, func(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
		b.Config.Workspace.CurrentUser = &config.User{
			User: &iam.User{
				UserName: "jane@doe.com",
			},
		}
		return nil
	})

	diags := bundle.Apply(ctx, b, mutator.SetRunAs())
	assert.NoError(t, diags.Error())

	assert.Len(t, b.Config.Resources.Jobs, 3)
	jobs := b.Config.Resources.Jobs

	// job_one and job_two should have the same run_as identity as the bundle's
	// development target.
	assert.NotNil(t, jobs["job_one"].RunAs)
	assert.Equal(t, "", jobs["job_one"].RunAs.ServicePrincipalName)
	assert.Equal(t, "my_user_name", jobs["job_one"].RunAs.UserName)

	assert.NotNil(t, jobs["job_two"].RunAs)
	assert.Equal(t, "", jobs["job_two"].RunAs.ServicePrincipalName)
	assert.Equal(t, "my_user_name", jobs["job_two"].RunAs.UserName)

	// job_three should retain the job level run_as identity.
	assert.NotNil(t, jobs["job_three"].RunAs)
	assert.Equal(t, "my_service_principal_for_job", jobs["job_three"].RunAs.ServicePrincipalName)
	assert.Equal(t, "", jobs["job_three"].RunAs.UserName)

	// Assert other resources are not affected.
	assert.Equal(t, ml.Model{Name: "skynet"}, *b.Config.Resources.Models["model_one"].Model)
	assert.Equal(t, catalog.CreateRegisteredModelRequest{Name: "skynet (in UC)"}, *b.Config.Resources.RegisteredModels["model_two"].CreateRegisteredModelRequest)
	assert.Equal(t, ml.Experiment{Name: "experiment_one"}, *b.Config.Resources.Experiments["experiment_one"].Experiment)

}

func TestRunAsErrorForPipelines(t *testing.T) {
	b := load(t, "./run_as/not_allowed/pipelines")

	ctx := context.Background()
	bundle.ApplyFunc(ctx, b, func(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
		b.Config.Workspace.CurrentUser = &config.User{
			User: &iam.User{
				UserName: "jane@doe.com",
			},
		}
		return nil
	})

	diags := bundle.Apply(ctx, b, mutator.SetRunAs())
	err := diags.Error()

	configPath := filepath.FromSlash("run_as/not_allowed/pipelines/databricks.yml")
	assert.EqualError(t, err, fmt.Sprintf("pipelines are not supported when the current deployment user is different from the bundle's run_as identity. Please deploy as the run_as identity. Please refer to the documentation at https://docs.databricks.com/dev-tools/bundles/run-as.html for more details. Location of the unsupported resource: %s:14:5. Current identity: jane@doe.com. Run as identity: my_service_principal", configPath))
}

func TestRunAsNoErrorForPipelines(t *testing.T) {
	b := load(t, "./run_as/not_allowed/pipelines")

	// We should not error because the pipeline is being deployed with the same
	// identity as the bundle run_as identity.
	ctx := context.Background()
	bundle.ApplyFunc(ctx, b, func(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
		b.Config.Workspace.CurrentUser = &config.User{
			User: &iam.User{
				UserName: "my_service_principal",
			},
		}
		return nil
	})

	diags := bundle.Apply(ctx, b, mutator.SetRunAs())
	assert.NoError(t, diags.Error())
}

func TestRunAsErrorForModelServing(t *testing.T) {
	b := load(t, "./run_as/not_allowed/model_serving")

	ctx := context.Background()
	bundle.ApplyFunc(ctx, b, func(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
		b.Config.Workspace.CurrentUser = &config.User{
			User: &iam.User{
				UserName: "jane@doe.com",
			},
		}
		return nil
	})

	diags := bundle.Apply(ctx, b, mutator.SetRunAs())
	err := diags.Error()

	configPath := filepath.FromSlash("run_as/not_allowed/model_serving/databricks.yml")
	assert.EqualError(t, err, fmt.Sprintf("model_serving_endpoints are not supported when the current deployment user is different from the bundle's run_as identity. Please deploy as the run_as identity. Please refer to the documentation at https://docs.databricks.com/dev-tools/bundles/run-as.html for more details. Location of the unsupported resource: %s:14:5. Current identity: jane@doe.com. Run as identity: my_service_principal", configPath))
}

func TestRunAsNoErrorForModelServingEndpoints(t *testing.T) {
	b := load(t, "./run_as/not_allowed/model_serving")

	// We should not error because the model serving endpoint is being deployed
	// with the same identity as the bundle run_as identity.
	ctx := context.Background()
	bundle.ApplyFunc(ctx, b, func(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
		b.Config.Workspace.CurrentUser = &config.User{
			User: &iam.User{
				UserName: "my_service_principal",
			},
		}
		return nil
	})

	diags := bundle.Apply(ctx, b, mutator.SetRunAs())
	assert.NoError(t, diags.Error())
}

func TestRunAsErrorWhenBothUserAndSpSpecified(t *testing.T) {
	b := load(t, "./run_as/not_allowed/both_sp_and_user")

	ctx := context.Background()
	bundle.ApplyFunc(ctx, b, func(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
		b.Config.Workspace.CurrentUser = &config.User{
			User: &iam.User{
				UserName: "my_service_principal",
			},
		}
		return nil
	})

	diags := bundle.Apply(ctx, b, mutator.SetRunAs())
	err := diags.Error()

	configPath := filepath.FromSlash("run_as/not_allowed/both_sp_and_user/databricks.yml")
	assert.EqualError(t, err, fmt.Sprintf("run_as section must specify exactly one identity. A service_principal_name \"my_service_principal\" is specified at %s:6:27. A user_name \"my_user_name\" is defined at %s:7:14", configPath, configPath))
}

func TestRunAsErrorNeitherUserOrSpSpecified(t *testing.T) {
	tcases := []struct {
		name string
		err  string
	}{
		{
			name: "empty_run_as",
			err:  fmt.Sprintf("run_as section must specify exactly one identity. Neither service_principal_name nor user_name is specified at %s:4:8", filepath.FromSlash("run_as/not_allowed/neither_sp_nor_user/empty_run_as/databricks.yml")),
		},
		{
			name: "empty_sp",
			err:  fmt.Sprintf("run_as section must specify exactly one identity. Neither service_principal_name nor user_name is specified at %s:5:3", filepath.FromSlash("run_as/not_allowed/neither_sp_nor_user/empty_sp/databricks.yml")),
		},
		{
			name: "empty_user",
			err:  fmt.Sprintf("run_as section must specify exactly one identity. Neither service_principal_name nor user_name is specified at %s:5:3", filepath.FromSlash("run_as/not_allowed/neither_sp_nor_user/empty_user/databricks.yml")),
		},
		{
			name: "empty_user_and_sp",
			err:  fmt.Sprintf("run_as section must specify exactly one identity. Neither service_principal_name nor user_name is specified at %s:5:3", filepath.FromSlash("run_as/not_allowed/neither_sp_nor_user/empty_user_and_sp/databricks.yml")),
		},
	}

	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {

			bundlePath := fmt.Sprintf("./run_as/not_allowed/neither_sp_nor_user/%s", tc.name)
			b := load(t, bundlePath)

			ctx := context.Background()
			bundle.ApplyFunc(ctx, b, func(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
				b.Config.Workspace.CurrentUser = &config.User{
					User: &iam.User{
						UserName: "my_service_principal",
					},
				}
				return nil
			})

			diags := bundle.Apply(ctx, b, mutator.SetRunAs())
			err := diags.Error()
			assert.EqualError(t, err, tc.err)
		})
	}
}

func TestRunAsErrorNeitherUserOrSpSpecifiedAtTargetOverride(t *testing.T) {
	b := loadTarget(t, "./run_as/not_allowed/neither_sp_nor_user/override", "development")

	ctx := context.Background()
	bundle.ApplyFunc(ctx, b, func(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
		b.Config.Workspace.CurrentUser = &config.User{
			User: &iam.User{
				UserName: "my_service_principal",
			},
		}
		return nil
	})

	diags := bundle.Apply(ctx, b, mutator.SetRunAs())
	err := diags.Error()

	configPath := filepath.FromSlash("run_as/not_allowed/neither_sp_nor_user/override/override.yml")
	assert.EqualError(t, err, fmt.Sprintf("run_as section must specify exactly one identity. Neither service_principal_name nor user_name is specified at %s:4:12", configPath))
}

func TestLegacyRunAs(t *testing.T) {
	b := load(t, "./run_as/legacy")

	ctx := context.Background()
	bundle.ApplyFunc(ctx, b, func(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
		b.Config.Workspace.CurrentUser = &config.User{
			User: &iam.User{
				UserName: "jane@doe.com",
			},
		}
		return nil
	})

	diags := bundle.Apply(ctx, b, mutator.SetRunAs())
	assert.NoError(t, diags.Error())

	assert.Len(t, b.Config.Resources.Jobs, 3)
	jobs := b.Config.Resources.Jobs

	// job_one and job_two should have the same run_as identity as the bundle.
	assert.NotNil(t, jobs["job_one"].RunAs)
	assert.Equal(t, "my_service_principal", jobs["job_one"].RunAs.ServicePrincipalName)
	assert.Equal(t, "", jobs["job_one"].RunAs.UserName)

	assert.NotNil(t, jobs["job_two"].RunAs)
	assert.Equal(t, "my_service_principal", jobs["job_two"].RunAs.ServicePrincipalName)
	assert.Equal(t, "", jobs["job_two"].RunAs.UserName)

	// job_three should retain it's run_as identity.
	assert.NotNil(t, jobs["job_three"].RunAs)
	assert.Equal(t, "my_service_principal_for_job", jobs["job_three"].RunAs.ServicePrincipalName)
	assert.Equal(t, "", jobs["job_three"].RunAs.UserName)

	// Assert owner permissions for pipelines are set.
	pipelines := b.Config.Resources.Pipelines
	assert.Len(t, pipelines["nyc_taxi_pipeline"].Permissions, 2)

	assert.Equal(t, "CAN_VIEW", pipelines["nyc_taxi_pipeline"].Permissions[0].Level)
	assert.Equal(t, "my_user_name", pipelines["nyc_taxi_pipeline"].Permissions[0].UserName)

	assert.Equal(t, "IS_OWNER", pipelines["nyc_taxi_pipeline"].Permissions[1].Level)
	assert.Equal(t, "my_service_principal", pipelines["nyc_taxi_pipeline"].Permissions[1].ServicePrincipalName)

	// Assert other resources are not affected.
	assert.Equal(t, ml.Model{Name: "skynet"}, *b.Config.Resources.Models["model_one"].Model)
	assert.Equal(t, catalog.CreateRegisteredModelRequest{Name: "skynet (in UC)"}, *b.Config.Resources.RegisteredModels["model_two"].CreateRegisteredModelRequest)
	assert.Equal(t, ml.Experiment{Name: "experiment_one"}, *b.Config.Resources.Experiments["experiment_one"].Experiment)
	assert.Equal(t, serving.CreateServingEndpoint{Name: "skynet"}, *b.Config.Resources.ModelServingEndpoints["model_serving_one"].CreateServingEndpoint)
}
