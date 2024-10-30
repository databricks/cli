package permissions

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/iamutil"
	"github.com/databricks/cli/libs/log"
)

// ReportPossiblePermissionDenied generates a diagnostic message when a permission denied error is encountered.
//
// Note that since the workspace API doesn't always distinguish between permission denied and path errors,
// we must treat this as a "possible permission error". See acquire.go for more about this.
func ReportPossiblePermissionDenied(ctx context.Context, b *bundle.Bundle, path string) diag.Diagnostics {
	log.Errorf(ctx, "Failed to update, encountered possible permission error: %v", path)

	me := b.Config.Workspace.CurrentUser.User
	userName := me.UserName
	if iamutil.IsServicePrincipal(me) {
		userName = me.DisplayName
	}
	canManageBundle, assistance := analyzeBundlePermissions(b)

	if !canManageBundle {
		return diag.Diagnostics{{
			Summary: fmt.Sprintf("unable to deploy to %s as %s.\n"+
				"Please make sure the current user or one of their groups is listed under the permissions of this bundle.\n"+
				"%s\n"+
				"They may need to redeploy the bundle to apply the new permissions.\n"+
				"Please refer to https://docs.databricks.com/dev-tools/bundles/permissions.html for more on managing permissions.",
				path, userName, assistance),
			Severity: diag.Error,
			ID:       diag.PathPermissionDenied,
		}}
	}

	// According databricks.yml, the current user has the right permissions.
	// But we're still seeing permission errors. So someone else will need
	// to redeploy the bundle with the right set of permissions.
	return diag.Diagnostics{{
		Summary: fmt.Sprintf("unable to deploy to %s as %s. Cannot apply local deployment permissions.\n"+
			"%s\n"+
			"They can redeploy the project to apply the latest set of permissions.\n"+
			"Please refer to https://docs.databricks.com/dev-tools/bundles/permissions.html for more on managing permissions.",
			path, userName, assistance),
		Severity: diag.Error,
		ID:       diag.CannotChangePathPermissions,
	}}
}
