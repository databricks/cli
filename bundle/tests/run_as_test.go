package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
)

func TestRunAsDefault(t *testing.T) {
	b := load(t, "./run_as")

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

	assert.NotNil(t, jobs["job_one"].RunAs)
	assert.Equal(t, "my_service_principal", jobs["job_one"].RunAs.ServicePrincipalName)
	assert.Equal(t, "", jobs["job_one"].RunAs.UserName)

	assert.NotNil(t, jobs["job_two"].RunAs)
	assert.Equal(t, "my_service_principal", jobs["job_two"].RunAs.ServicePrincipalName)
	assert.Equal(t, "", jobs["job_two"].RunAs.UserName)

	assert.NotNil(t, jobs["job_three"].RunAs)
	assert.Equal(t, "my_service_principal_for_job", jobs["job_three"].RunAs.ServicePrincipalName)
	assert.Equal(t, "", jobs["job_three"].RunAs.UserName)

	pipelines := b.Config.Resources.Pipelines
	assert.Len(t, pipelines["nyc_taxi_pipeline"].Permissions, 2)
	assert.Equal(t, "CAN_VIEW", pipelines["nyc_taxi_pipeline"].Permissions[0].Level)
	assert.Equal(t, "my_user_name", pipelines["nyc_taxi_pipeline"].Permissions[0].UserName)

	assert.Equal(t, "IS_OWNER", pipelines["nyc_taxi_pipeline"].Permissions[1].Level)
	assert.Equal(t, "my_service_principal", pipelines["nyc_taxi_pipeline"].Permissions[1].ServicePrincipalName)
}

func TestRunAsDevelopment(t *testing.T) {
	b := loadTarget(t, "./run_as", "development")

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

	assert.NotNil(t, jobs["job_one"].RunAs)
	assert.Equal(t, "", jobs["job_one"].RunAs.ServicePrincipalName)
	assert.Equal(t, "my_user_name", jobs["job_one"].RunAs.UserName)

	assert.NotNil(t, jobs["job_two"].RunAs)
	assert.Equal(t, "", jobs["job_two"].RunAs.ServicePrincipalName)
	assert.Equal(t, "my_user_name", jobs["job_two"].RunAs.UserName)

	assert.NotNil(t, jobs["job_three"].RunAs)
	assert.Equal(t, "my_service_principal_for_job", jobs["job_three"].RunAs.ServicePrincipalName)
	assert.Equal(t, "", jobs["job_three"].RunAs.UserName)

	pipelines := b.Config.Resources.Pipelines
	assert.Len(t, pipelines["nyc_taxi_pipeline"].Permissions, 2)
	assert.Equal(t, "CAN_VIEW", pipelines["nyc_taxi_pipeline"].Permissions[0].Level)
	assert.Equal(t, "my_service_principal", pipelines["nyc_taxi_pipeline"].Permissions[0].ServicePrincipalName)

	assert.Equal(t, "IS_OWNER", pipelines["nyc_taxi_pipeline"].Permissions[1].Level)
	assert.Equal(t, "my_user_name", pipelines["nyc_taxi_pipeline"].Permissions[1].UserName)
}
