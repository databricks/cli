package permissions

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/dyn"
	dync "github.com/databricks/cli/libs/dyn/convert"
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

func filter(currentUser string) func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
	return func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		// Permissions are defined at top level of a resource. We can skip walking
		// after a depth of 4.
		// [resource_type].[resource_name].[permissions].[array_index]
		// Example: pipelines.foo.permissions.0
		if len(p) > 4 {
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

func (m *filterCurrentUser) Apply(ctx context.Context, b *bundle.Bundle) error {
	rv, err := dync.FromTyped(b.Config.Resources, dyn.NilValue)
	if err != nil {
		return err
	}

	currentUser := b.Config.Workspace.CurrentUser.UserName
	nv, err := dyn.Walk(rv, filter(currentUser))
	if err != nil {
		return err
	}

	err = dync.ToTyped(&b.Config.Resources, nv)
	if err != nil {
		return err
	}

	return nil
}
