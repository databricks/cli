package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/iamutil"
)

type secretScopeFixups struct {
	engine engine.EngineType
}

func SecretScopeFixups(engine engine.EngineType) bundle.Mutator {
	return &secretScopeFixups{engine: engine}
}

func (m *secretScopeFixups) Name() string {
	return "SecretScopeFixups"
}

func (m *secretScopeFixups) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	// Secret scopes by default have the current user as a MANAGE ACL. We need to add it to the client ACL list
	// to prevent a phantom persistent diff.
	// We do not need to do this in terraform because terraform naively always applies the config during ACL
	// creation without checking if the ACL already exists.
	// https://github.com/databricks/terraform-provider-databricks/blob/5cb5d3fa46bc4843be1a4c4bce89296eaa2e14fc/secrets/resource_secret_acl.go#L43
	if !m.engine.IsDirect() {
		return nil
	}

	// Secret scopes assigns the create MANAGE ACL on it by default. So we always add it to
	// the client ACL list as a default.
	for _, scope := range b.Config.Resources.SecretScopes {
		if scope == nil {
			continue
		}

		currentUser := b.Config.Workspace.CurrentUser.User

		acl := resources.SecretScopePermission{
			Level: resources.SecretScopePermissionLevelManage,
		}
		if iamutil.IsServicePrincipal(currentUser) {
			acl.ServicePrincipalName = currentUser.UserName
		} else {
			acl.UserName = currentUser.UserName
		}

		scope.Permissions = append(scope.Permissions, acl)
	}

	return nil
}
