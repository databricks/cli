package config_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunAsDefault(t *testing.T) {
	b := load(t, "./run_as")
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
	assert.NotNil(t, pipelines["nyc_taxi_pipeline"].Permissions)
	assert.Equal(t, pipelines["nyc_taxi_pipeline"].Permissions[0].Level, "IS_OWNER")
	assert.Equal(t, pipelines["nyc_taxi_pipeline"].Permissions[0].ServicePrincipalName, "my_service_principal")
}

func TestRunAsDevelopment(t *testing.T) {
	b := loadTarget(t, "./run_as", "development")
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
	assert.NotNil(t, pipelines["nyc_taxi_pipeline"].Permissions)
	assert.Equal(t, pipelines["nyc_taxi_pipeline"].Permissions[0].Level, "IS_OWNER")
	assert.Equal(t, pipelines["nyc_taxi_pipeline"].Permissions[0].UserName, "my_user_name")
}
