package config_tests

import (
	"context"
	"runtime"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/stretchr/testify/assert"
)

func TestRunAsForAllowed(t *testing.T) {
	b := load(t, "./run_as/allowed")

	ctx := context.Background()
	bundle.ApplyFunc(ctx, b, func(ctx context.Context, b *bundle.Bundle) error {
		b.Config.Workspace.CurrentUser = &config.User{
			User: &iam.User{
				UserName: "jane@doe.com",
			},
		}
		return nil
	})

	err := bundle.Apply(ctx, b, mutator.SetRunAs())
	assert.NoError(t, err)

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
	bundle.ApplyFunc(ctx, b, func(ctx context.Context, b *bundle.Bundle) error {
		b.Config.Workspace.CurrentUser = &config.User{
			User: &iam.User{
				UserName: "jane@doe.com",
			},
		}
		return nil
	})

	err := bundle.Apply(ctx, b, mutator.SetRunAs())
	assert.NoError(t, err)

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
	bundle.ApplyFunc(ctx, b, func(ctx context.Context, b *bundle.Bundle) error {
		b.Config.Workspace.CurrentUser = &config.User{
			User: &iam.User{
				UserName: "jane@doe.com",
			},
		}
		return nil
	})

	err := bundle.Apply(ctx, b, mutator.SetRunAs())

	if runtime.GOOS == "windows" {
		assert.EqualError(t, err, "pipelines are not supported when the current deployment user is different from the bundle's run_as identity. Please deploy as the run_as identity. List of supported resources: [jobs, models, registered_models, experiments]. Location of the unsupported resource: run_as\\not_allowed\\pipelines\\databricks.yml:14:5. Current identity: jane@doe.com. Run as identity: my_service_principal")
	} else {
		assert.EqualError(t, err, "pipelines are not supported when the current deployment user is different from the bundle's run_as identity. Please deploy as the run_as identity. List of supported resources: [jobs, models, registered_models, experiments]. Location of the unsupported resource: run_as/not_allowed/pipelines/databricks.yml:14:5. Current identity: jane@doe.com. Run as identity: my_service_principal")
	}
}

func TestRunAsNoErrorForPipelines(t *testing.T) {
	b := load(t, "./run_as/not_allowed/pipelines")

	// We should not error because the pipeline is being deployed with the same
	// identity as the bundle run_as identity.
	ctx := context.Background()
	bundle.ApplyFunc(ctx, b, func(ctx context.Context, b *bundle.Bundle) error {
		b.Config.Workspace.CurrentUser = &config.User{
			User: &iam.User{
				UserName: "my_service_principal",
			},
		}
		return nil
	})

	err := bundle.Apply(ctx, b, mutator.SetRunAs())
	assert.NoError(t, err)
}

func TestRunAsErrorForModelServing(t *testing.T) {
	b := load(t, "./run_as/not_allowed/model_serving")

	ctx := context.Background()
	bundle.ApplyFunc(ctx, b, func(ctx context.Context, b *bundle.Bundle) error {
		b.Config.Workspace.CurrentUser = &config.User{
			User: &iam.User{
				UserName: "jane@doe.com",
			},
		}
		return nil
	})

	err := bundle.Apply(ctx, b, mutator.SetRunAs())

	if runtime.GOOS == "windows" {
		assert.EqualError(t, err, "model_serving_endpoints are not supported when the current deployment user is different from the bundle's run_as identity. Please deploy as the run_as identity. List of supported resources: [jobs, models, registered_models, experiments]. Location of the unsupported resource: run_as\\not_allowed\\model_serving\\databricks.yml:14:5. Current identity: jane@doe.com. Run as identity: my_service_principal")
	} else {
		assert.EqualError(t, err, "model_serving_endpoints are not supported when the current deployment user is different from the bundle's run_as identity. Please deploy as the run_as identity. List of supported resources: [jobs, models, registered_models, experiments]. Location of the unsupported resource: run_as/not_allowed/model_serving/databricks.yml:14:5. Current identity: jane@doe.com. Run as identity: my_service_principal")
	}
}

func TestRunAsNoErrorForModelServingEndpoints(t *testing.T) {
	b := load(t, "./run_as/not_allowed/model_serving")

	// We should not error because the model serving endpoint is being deployed
	// with the same identity as the bundle run_as identity.
	ctx := context.Background()
	bundle.ApplyFunc(ctx, b, func(ctx context.Context, b *bundle.Bundle) error {
		b.Config.Workspace.CurrentUser = &config.User{
			User: &iam.User{
				UserName: "my_service_principal",
			},
		}
		return nil
	})

	err := bundle.Apply(ctx, b, mutator.SetRunAs())
	assert.NoError(t, err)
}

func TestRunAsErrorWhenBothUserAndSpSpecified(t *testing.T) {
	b := load(t, "./run_as/not_allowed/both_sp_and_user")

	ctx := context.Background()
	bundle.ApplyFunc(ctx, b, func(ctx context.Context, b *bundle.Bundle) error {
		b.Config.Workspace.CurrentUser = &config.User{
			User: &iam.User{
				UserName: "my_service_principal",
			},
		}
		return nil
	})

	err := bundle.Apply(ctx, b, mutator.SetRunAs())

	if runtime.GOOS == "windows" {
		assert.EqualError(t, err, "run_as section must specify exactly one identity. A service_principal_name \"my_service_principal\" is specified at run_as\\not_allowed\\both_sp_and_user\\databricks.yml:6:27. A user_name \"my_user_name\" is defined at run_as\\not_allowed\\both_sp_and_user\\databricks.yml:7:14")
	} else {
		assert.EqualError(t, err, "run_as section must specify exactly one identity. A service_principal_name \"my_service_principal\" is specified at run_as/not_allowed/both_sp_and_user/databricks.yml:6:27. A user_name \"my_user_name\" is defined at run_as/not_allowed/both_sp_and_user/databricks.yml:7:14")
	}
}
