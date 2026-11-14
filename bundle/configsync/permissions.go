package configsync

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"golang.org/x/sync/errgroup"
)

const permissionsReadLimit = 10

// addPermissionsChanges adds remote permission changes to the permissions field of their
// resource: per principal where the config declares permissions, and as a whole for deployed
// resources that declare none.
func addPermissionsChanges(ctx context.Context, changes Changes, b *bundle.Bundle, plan *deployplan.Plan) error {
	for resourceKey, entry := range plan.Plan {
		if err := addDeclaredPermissionsChanges(changes, b, resourceKey, entry); err != nil {
			return err
		}
	}

	for resourceKey, permissions := range readUnmanagedPermissions(ctx, b, plan) {
		value, err := normalizeValue(permissions)
		if err != nil {
			return fmt.Errorf("failed to compute config change for %s: %w", resourceKey, err)
		}
		changes.set(resourceKey, "permissions", &ConfigChangeDesc{
			Operation: OperationAdd,
			Value:     value,
		})
	}
	return nil
}

// addDeclaredPermissionsChanges turns the plan's changes to a resource's permissions into
// changes of its permissions field, e.g. "[user_name='a'].level" becomes
// "permissions[user_name='a'].level". Only entries written in the resource's config are changed:
// the ones FixPermissions or bundle-level permissions add have no YAML node to write to.
func addDeclaredPermissionsChanges(changes Changes, b *bundle.Bundle, resourceKey string, entry *deployplan.PlanEntry) error {
	if _, ok := entry.RemoteState.(*dresources.PermissionsState); !ok {
		return nil
	}
	declared := declaredPrincipals(b, resourceKey)
	if len(declared) == 0 {
		return nil
	}
	parentKey := strings.TrimSuffix(resourceKey, ".permissions")

	for path, changeDesc := range entry.Changes {
		node, err := structpath.ParsePath(path)
		if err != nil {
			return fmt.Errorf("failed to parse path %s: %w", path, err)
		}
		nodes := node.AsSlice()
		if len(nodes) == 0 {
			// The remote ACL has no direct entries left, so the change covers the whole list.
			if changeDesc.Action != deployplan.Skip {
				for selector := range declared {
					changes.set(parentKey, "permissions"+selector, &ConfigChangeDesc{Operation: OperationRemove})
				}
			}
			continue
		}
		if _, _, ok := nodes[0].KeyValue(); !ok || (changeDesc.New != nil && !declared[nodes[0].String()]) {
			continue
		}

		fieldPath := "permissions" + path
		change, err := convertPlanChange(parentKey+"."+fieldPath, changeDesc, b.Config.Presets.NamePrefix)
		if err != nil {
			return fmt.Errorf("failed to compute config change for path %s: %w", resourceKey+"."+path, err)
		}
		if change != nil {
			changes.set(parentKey, fieldPath, change)
		}
	}
	return nil
}

// declaredPrincipals returns the permissions written in the resource's config as the plan's
// selectors, e.g. "[user_name='a']". Entries added by mutators have no location.
func declaredPrincipals(b *bundle.Bundle, resourceKey string) map[string]bool {
	path, err := dyn.NewPathFromString(resourceKey)
	if err != nil {
		return nil
	}
	permissions, _ := dyn.GetByPath(b.Config.Value(), path)
	sequence, _ := permissions.AsSequence()
	result := make(map[string]bool)
	for _, permission := range sequence {
		if permission.Location().File == "" {
			continue
		}
		for _, key := range []string{"user_name", "service_principal_name", "group_name"} {
			if value, ok := permission.Get(key).AsString(); ok && value != "" {
				result[structpath.NewKeyValue(nil, key, value).String()] = true
			}
		}
	}
	return result
}

func (c Changes) set(resourceKey, fieldPath string, change *ConfigChangeDesc) {
	if c[resourceKey] == nil {
		c[resourceKey] = make(ResourceChanges)
	}
	c[resourceKey][fieldPath] = change
}

// readUnmanagedPermissions reads the remote permissions of deployed resources whose config
// declares none, keyed by resource. They are imported as a whole: the current ACL cannot tell
// grants made before the deployment from the ones made since.
func readUnmanagedPermissions(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan) map[string][]dresources.StatePermission {
	var mu sync.Mutex
	var g errgroup.Group
	g.SetLimit(permissionsReadLimit)
	result := make(map[string][]dresources.StatePermission)
	currentUser := currentUserName(b)

	for resourceKey, adapter := range unmanagedPermissionsAdapters(b, plan) {
		g.Go(func() error {
			permissions, err := readPermissions(ctx, b, adapter, resourceKey)
			if err != nil {
				// A user who can deploy a resource is not necessarily allowed to read its ACL.
				log.Warnf(ctx, "reading permissions of %s: %s", resourceKey, err)
				return nil
			}
			permissions = withoutCurrentUserManage(permissions, currentUser)
			if len(permissions) > 0 {
				mu.Lock()
				result[resourceKey] = permissions
				mu.Unlock()
			}
			return nil
		})
	}

	_ = g.Wait()
	return result
}

// unmanagedPermissionsAdapters returns the permissions adapter of every deployed resource whose
// config declares no permissions. A state-only permissions entry is deleted on the next deploy
// without changing the ACL, so it does not count.
func unmanagedPermissionsAdapters(b *bundle.Bundle, plan *deployplan.Plan) map[string]*dresources.Adapter {
	result := make(map[string]*dresources.Adapter)
	for resourceKey, entry := range plan.Plan {
		if isPermissionsOrGrantsSubResource(resourceKey) || !entry.Action.KeepsID() {
			continue
		}
		permissionsKey := resourceKey + ".permissions"
		if permissions := plan.Plan[permissionsKey]; permissions != nil && permissions.Action != deployplan.Delete {
			continue
		}
		adapter := b.DeploymentBundle.Adapters[config.GetResourceTypeFromKey(permissionsKey)]
		if adapter != nil && adapter.RemoteType() == reflect.TypeFor[*dresources.PermissionsState]() {
			result[resourceKey] = adapter
		}
	}
	return result
}

func readPermissions(ctx context.Context, b *bundle.Bundle, adapter *dresources.Adapter, resourceKey string) ([]dresources.StatePermission, error) {
	sv, err := adapter.PrepareInputConfig([]dresources.StatePermission{}, resourceKey+".permissions")
	if err != nil {
		return nil, err
	}
	// The object ID references a field of the resource, e.g. "/jobs/${resources.jobs.foo.id}". It is
	// resolved here by key, since a key with e.g. spaces does not parse as a reference.
	objectID := sv.Refs["object_id"]
	objectType, field, ok := strings.Cut(objectID, "${"+resourceKey+".")
	field, hasSuffix := strings.CutSuffix(field, "}")
	if !ok || !hasSuffix {
		return nil, fmt.Errorf("unexpected object_id %q", objectID)
	}
	value, err := lookupDeployedField(b, resourceKey, field)
	if err != nil {
		return nil, fmt.Errorf("resolving %s.%s: %w", resourceKey, field, err)
	}

	id := objectType + fmt.Sprint(value)
	remote, err := retry.With(ctx, retry.IsTransientRead, func() (any, error) {
		return adapter.DoRead(ctx, id)
	})
	if err != nil {
		return nil, err
	}
	return remote.(*dresources.PermissionsState).EmbeddedSlice, nil
}

// lookupDeployedField reads a field of a deployed resource, such as the ID its permissions are
// keyed by. The plan drops the new state of unchanged resources, so other fields come from the
// remote state it read.
func lookupDeployedField(b *bundle.Bundle, resourceKey, field string) (any, error) {
	if field == "id" {
		return b.DeploymentBundle.StateDB.GetResourceID(resourceKey), nil
	}
	path, err := structpath.ParsePath(field)
	if err != nil {
		return nil, err
	}
	remote, ok := b.DeploymentBundle.RemoteStateCache.Load(resourceKey)
	if !ok {
		return nil, fmt.Errorf("no remote state for %s", resourceKey)
	}
	return structaccess.Get(remote, path)
}

// withoutCurrentUserManage drops the current user's IS_OWNER and CAN_MANAGE, one of which
// FixPermissions adds on deploy. Another principal's IS_OWNER is kept, so a transferred owner
// stays the owner after the next deploy.
func withoutCurrentUserManage(permissions []dresources.StatePermission, currentUser string) []dresources.StatePermission {
	return slices.DeleteFunc(permissions, func(p dresources.StatePermission) bool {
		isCurrentUser := currentUser != "" && (p.UserName == currentUser || p.ServicePrincipalName == currentUser)
		return isCurrentUser && (p.Level == iam.PermissionLevelIsOwner || p.Level == iam.PermissionLevelCanManage)
	})
}

func currentUserName(b *bundle.Bundle) string {
	if b.Config.Workspace.CurrentUser == nil {
		return ""
	}
	return b.Config.Workspace.CurrentUser.UserName
}
