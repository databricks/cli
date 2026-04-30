package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type validateGenieSpacePermissions struct{}

// ValidateGenieSpacePermissions errors if any genie_space resource has
// permissions configured. The Databricks workspace API does not expose
// PUT /permissions/genie/spaces/<id>, so the deploy would create the
// space and then fail when applying permissions, leaving partial state.
// Bundle-level permissions are propagated to genie_spaces by
// ApplyBundlePermissions and are caught here as well.
func ValidateGenieSpacePermissions() bundle.Mutator {
	return &validateGenieSpacePermissions{}
}

func (m *validateGenieSpacePermissions) Name() string {
	return "ValidateGenieSpacePermissions"
}

func (m *validateGenieSpacePermissions) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	for key, space := range b.Config.Resources.GenieSpaces {
		if space == nil || len(space.Permissions) == 0 {
			continue
		}

		diags = diags.Append(diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   "Genie Space permissions are not supported",
			Detail:    "Databricks workspaces do not expose a permissions endpoint for Genie Spaces, so a deploy with permissions configured would create the space and then fail. Remove the permissions block, or remove the Genie Space from any bundle-level permissions, until the API adds support.",
			Locations: b.Config.GetLocations(fmt.Sprintf("resources.genie_spaces.%s.permissions", key)),
		})
	}

	return diags
}
