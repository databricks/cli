// Package permissions emits diagnostics when the current deployment identity
// lacks the UC privileges required to manage a ucm deployment's resources.
//
// Parallels bundle/permissions: bundle checks the workspace-folder permission
// set; UCM checks the UC securable permission set. The CAN_MANAGE equivalent
// in Unity Catalog is the MANAGE (or ALL_PRIVILEGES) privilege, queried via
// the Grants.GetEffective API so inherited grants are honored.
package permissions

import (
	"context"
	"errors"
	"net/http"
	"sort"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

// PermissionDiagnostics walks the declared catalogs and schemas and emits an
// error diagnostic for each one the current user cannot manage. Returns nil
// when every resource grants MANAGE (directly or inherited), when no resources
// are declared, or when CurrentUser / workspace client are unavailable — the
// check is best-effort and should never block deploy on its own bookkeeping.
//
// Resources whose securable doesn't yet exist (404) are skipped: the deploy
// will create them, and the parent-scope permission is a metastore-level
// concern UCM does not check here.
func PermissionDiagnostics(ctx context.Context, u *ucm.Ucm) diag.Diagnostics {
	if u == nil || u.CurrentUser == nil || u.CurrentUser.User == nil || u.CurrentUser.UserName == "" {
		return nil
	}

	w, err := u.WorkspaceClientE()
	if err != nil || w == nil || w.Grants == nil {
		log.Debugf(ctx, "permissions: skipping precheck, workspace client unavailable: %v", err)
		return nil
	}

	principal := u.CurrentUser.UserName

	var diags diag.Diagnostics
	for _, name := range sortedKeys(u.Config.Resources.Catalogs) {
		c := u.Config.Resources.Catalogs[name]
		if c == nil || c.Name == "" {
			continue
		}
		diags = append(diags, checkSecurable(ctx, w.Grants, catalog.SecurableTypeCatalog, c.Name, principal)...)
	}
	for _, name := range sortedKeys(u.Config.Resources.Schemas) {
		s := u.Config.Resources.Schemas[name]
		if s == nil || s.Name == "" || s.Catalog == "" {
			continue
		}
		fullName := s.Catalog + "." + s.Name
		diags = append(diags, checkSecurable(ctx, w.Grants, catalog.SecurableTypeSchema, fullName, principal)...)
	}
	return diags
}

func checkSecurable(ctx context.Context, grants catalog.GrantsInterface, securableType catalog.SecurableType, fullName, principal string) diag.Diagnostics {
	resp, err := grants.GetEffective(ctx, catalog.GetEffectiveRequest{
		SecurableType: string(securableType),
		FullName:      fullName,
		Principal:     principal,
	})
	if err != nil {
		var aerr *apierr.APIError
		if errors.As(err, &aerr) && aerr.StatusCode == http.StatusNotFound {
			log.Debugf(ctx, "permissions: %s %q not found, skipping", securableType, fullName)
			return nil
		}
		return diag.Errorf("permission precheck failed for %s %q: %v", securableType, fullName, err)
	}

	if hasManage(resp, principal) {
		return nil
	}
	return diag.Errorf("user %q does not have MANAGE on %s %q; deployment will fail. "+
		"Grant MANAGE (or ALL_PRIVILEGES) to this principal, or run the deploy as a user with sufficient privileges.",
		principal, securableType, fullName)
}

func hasManage(resp *catalog.EffectivePermissionsList, principal string) bool {
	if resp == nil {
		return false
	}
	for _, a := range resp.PrivilegeAssignments {
		if a.Principal != principal {
			continue
		}
		for _, p := range a.Privileges {
			if p.Privilege == catalog.PrivilegeManage || p.Privilege == catalog.PrivilegeAllPrivileges {
				return true
			}
		}
	}
	return false
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
