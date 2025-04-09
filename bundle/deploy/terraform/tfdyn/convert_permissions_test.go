package tfdyn

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertPermissions(t *testing.T) {
	src := resources.Job{
		Permissions: []resources.JobPermission{
			{
				Level:    "CAN_VIEW",
				UserName: "jane@doe.com",
			},
			{
				Level:     "CAN_MANAGE",
				GroupName: "special admins",
			},
			{
				Level:                "CAN_MANAGE_RUN",
				ServicePrincipalName: "spn",
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	resource := convertPermissionsResource(ctx, vin)
	require.NotNil(t, resource)
	assert.Equal(t, []schema.ResourcePermissionsAccessControl{
		{
			PermissionLevel:      "CAN_VIEW",
			UserName:             "jane@doe.com",
			GroupName:            "",
			ServicePrincipalName: "",
		},
		{
			PermissionLevel:      "CAN_MANAGE",
			UserName:             "",
			GroupName:            "special admins",
			ServicePrincipalName: "",
		},
		{
			PermissionLevel:      "CAN_MANAGE_RUN",
			UserName:             "",
			GroupName:            "",
			ServicePrincipalName: "spn",
		},
	}, resource.AccessControl)
}

func TestConvertPermissionsNil(t *testing.T) {
	src := resources.Job{
		Permissions: nil,
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	resource := convertPermissionsResource(ctx, vin)
	assert.Nil(t, resource)
}

func TestConvertPermissionsEmpty(t *testing.T) {
	src := resources.Job{
		Permissions: []resources.JobPermission{},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	resource := convertPermissionsResource(ctx, vin)
	assert.Nil(t, resource)
}
