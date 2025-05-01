package tfdyn

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

type resourceSecretAcl struct {
	schema.ResourceSecretAcl
	DependsOn []string `json:"depends_on,omitempty"`
}

type secretScopeConverter struct{}

func convertPermissionsSecretScope(key, scopeName string, permissions []dyn.Value, out *schema.Resources) {
	for idx, permission := range permissions {
		level, _ := permission.Get("level").AsString()
		userName, _ := permission.Get("user_name").AsString()
		groupName, _ := permission.Get("group_name").AsString()
		servicePrincipalName, _ := permission.Get("service_principal_name").AsString()

		principal := ""
		if userName != "" {
			principal = userName
		} else if groupName != "" {
			principal = groupName
		} else if servicePrincipalName != "" {
			principal = servicePrincipalName
		}

		acl := &resourceSecretAcl{
			ResourceSecretAcl: schema.ResourceSecretAcl{
				Permission: level,
				Principal:  principal,
				Scope:      scopeName,
			},
			DependsOn: []string{"databricks_secret_scope." + key},
		}

		aclKey := fmt.Sprintf("secret_acl_%s_%d", key, idx)
		out.SecretAcl[aclKey] = acl
	}
}

func (s secretScopeConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	// Normalize the output value to the target schema.
	vout, diags := convert.Normalize(workspace.SecretScope{}, vin)
	for _, diag := range diags {
		log.Debugf(ctx, "secret scope normalization diagnostic: %s", diag.Summary)
	}
	out.SecretScope[key] = vout.AsAny()

	// Configure permissions for this resource
	scopeName, _ := vin.Get("name").AsString()
	permissions, ok := vin.Get("permissions").AsSequence()
	if ok && len(permissions) > 0 {
		convertPermissionsSecretScope(key, scopeName, permissions, out)
	}

	return nil
}

func init() {
	registerConverter("secret_scopes", secretScopeConverter{})
}
