package tfdyn

import (
	"context"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
)

func convertPermissionsResource(ctx context.Context, vin dyn.Value) *schema.ResourcePermissions {
	permissions, ok := vin.GetTODO("permissions").AsSequence()
	if !ok || len(permissions) == 0 {
		return nil
	}

	resource := &schema.ResourcePermissions{}
	for _, permission := range permissions {
		level, _ := permission.GetTODO("level").AsString()
		userName, _ := permission.GetTODO("user_name").AsString()
		groupName, _ := permission.GetTODO("group_name").AsString()
		servicePrincipalName, _ := permission.GetTODO("service_principal_name").AsString()

		resource.AccessControl = append(resource.AccessControl, schema.ResourcePermissionsAccessControl{
			PermissionLevel:      level,
			UserName:             userName,
			GroupName:            groupName,
			ServicePrincipalName: servicePrincipalName,
		})
	}

	return resource
}
