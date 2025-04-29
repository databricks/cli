package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type filterCurrentUser struct{}

// The databricks terraform provider does not allow changing the permissions of
// current user. The current user is implied to be the owner of all deployed resources.
// This mutator removes the current user from the permissions of all resources.
func FilterCurrentUser() bundle.Mutator {
	return &filterCurrentUser{}
}

func (m *filterCurrentUser) Name() string {
	return "FilterCurrentUserFromPermissions"
}

func filter(currentUser string) dyn.WalkValueFunc {
	return func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		// Permissions are defined at top level of a resource. We can skip walking
		// after a depth of 4.
		// [resource_type].[resource_name].[permissions].[array_index]
		// Example: pipelines.foo.permissions.0
		if len(p) > 4 {
			return v, dyn.ErrSkip
		}

		// We can skip walking at a depth of 3 if the key is not "permissions".
		// Example: pipelines.foo.libraries
		if len(p) == 3 && p[2] != dyn.Key("permissions") {
			return v, dyn.ErrSkip
		}

		// We want to be at the level of an individual permission to check it's
		// user_name and service_principal_name fields.
		if len(p) != 4 || p[2] != dyn.Key("permissions") {
			return v, nil
		}

		// Filter if the user_name matches the current user
		userName, ok := v.Get("user_name").AsString()
		if ok && userName == currentUser {
			return v, dyn.ErrDrop
		}

		// Filter if the service_principal_name matches the current user
		servicePrincipalName, ok := v.Get("service_principal_name").AsString()
		if ok && servicePrincipalName == currentUser {
			return v, dyn.ErrDrop
		}

		return v, nil
	}
}

func (m *filterCurrentUser) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	currentUser := b.Config.Workspace.CurrentUser.UserName

	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		rv, err := dyn.Get(v, "resources")
		if err != nil {
			// If the resources key is not found, we can skip this mutator.
			if dyn.IsNoSuchKeyError(err) {
				return v, nil
			}

			return dyn.InvalidValue, err
		}

		// Walk the resources and filter out the current user from the permissions
		nv, err := dyn.Walk(rv, filter(currentUser))
		if err != nil {
			return dyn.InvalidValue, err
		}

		// Set the resources with the filtered permissions back into the bundle
		return dyn.Set(v, "resources", nv)
	})

	return diag.FromErr(err)
}
