package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundlePermissions(t *testing.T) {
	b := load(t, "./bundle_permissions")
	assert.Contains(t, b.Config.Permissions, resources.Permission{Level: "CAN_RUN", UserName: "test@company.com"})
	assert.NotContains(t, b.Config.Permissions, resources.Permission{Level: "CAN_MANAGE", GroupName: "devs"})
	assert.NotContains(t, b.Config.Permissions, resources.Permission{Level: "CAN_VIEW", ServicePrincipalName: "1234-abcd"})
	assert.NotContains(t, b.Config.Permissions, resources.Permission{Level: "CAN_RUN", UserName: "bot@company.com"})

	diags := bundle.Apply(context.Background(), b, resourcemutator.ApplyBundlePermissions())
	require.NoError(t, diags.Error())

	pipelinePermissions := b.Config.Resources.Pipelines["nyc_taxi_pipeline"].Permissions
	assert.Contains(t, pipelinePermissions, resources.PipelinePermission{Level: "CAN_RUN", UserName: "test@company.com"})
	assert.NotContains(t, pipelinePermissions, resources.PipelinePermission{Level: "CAN_MANAGE", GroupName: "devs"})
	assert.NotContains(t, pipelinePermissions, resources.PipelinePermission{Level: "CAN_VIEW", ServicePrincipalName: "1234-abcd"})
	assert.NotContains(t, pipelinePermissions, resources.PipelinePermission{Level: "CAN_RUN", UserName: "bot@company.com"})

	jobsPermissions := b.Config.Resources.Jobs["pipeline_schedule"].Permissions
	assert.Contains(t, jobsPermissions, resources.JobPermission{Level: "CAN_MANAGE_RUN", UserName: "test@company.com"})
	assert.NotContains(t, jobsPermissions, resources.JobPermission{Level: "CAN_MANAGE", GroupName: "devs"})
	assert.NotContains(t, jobsPermissions, resources.JobPermission{Level: "CAN_VIEW", ServicePrincipalName: "1234-abcd"})
	assert.NotContains(t, jobsPermissions, resources.JobPermission{Level: "CAN_MANAGE_RUN", UserName: "bot@company.com"})
}

func TestBundlePermissionsDevTarget(t *testing.T) {
	b := loadTarget(t, "./bundle_permissions", "development")
	assert.Contains(t, b.Config.Permissions, resources.Permission{Level: "CAN_RUN", UserName: "test@company.com"})
	assert.Contains(t, b.Config.Permissions, resources.Permission{Level: "CAN_MANAGE", GroupName: "devs"})
	assert.Contains(t, b.Config.Permissions, resources.Permission{Level: "CAN_VIEW", ServicePrincipalName: "1234-abcd"})
	assert.Contains(t, b.Config.Permissions, resources.Permission{Level: "CAN_RUN", UserName: "bot@company.com"})

	diags := bundle.Apply(context.Background(), b, resourcemutator.ApplyBundlePermissions())
	require.NoError(t, diags.Error())

	pipelinePermissions := b.Config.Resources.Pipelines["nyc_taxi_pipeline"].Permissions
	assert.Contains(t, pipelinePermissions, resources.PipelinePermission{Level: "CAN_RUN", UserName: "test@company.com"})
	assert.Contains(t, pipelinePermissions, resources.PipelinePermission{Level: "CAN_MANAGE", GroupName: "devs"})
	assert.Contains(t, pipelinePermissions, resources.PipelinePermission{Level: "CAN_VIEW", ServicePrincipalName: "1234-abcd"})
	assert.Contains(t, pipelinePermissions, resources.PipelinePermission{Level: "CAN_RUN", UserName: "bot@company.com"})

	jobsPermissions := b.Config.Resources.Jobs["pipeline_schedule"].Permissions
	assert.Contains(t, jobsPermissions, resources.JobPermission{Level: "CAN_MANAGE_RUN", UserName: "test@company.com"})
	assert.Contains(t, jobsPermissions, resources.JobPermission{Level: "CAN_MANAGE", GroupName: "devs"})
	assert.Contains(t, jobsPermissions, resources.JobPermission{Level: "CAN_VIEW", ServicePrincipalName: "1234-abcd"})
	assert.Contains(t, jobsPermissions, resources.JobPermission{Level: "CAN_MANAGE_RUN", UserName: "bot@company.com"})
}
