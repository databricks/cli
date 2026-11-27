package configsync

import (
	"context"
	"errors"
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
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"golang.org/x/sync/errgroup"
)

const permissionsReadLimit = 10

// addPermissionsChanges adds remote permission changes to the permissions field of their
// resource, per principal. The ACL of a resource that declares no permissions is left alone: it
// is not managed by the bundle, and a block written for it would replace the ACL on every later
// deploy, by any identity and to any target. remote holds the ACLs read by readRemotePermissions.
func addPermissionsChanges(changes Changes, b *bundle.Bundle, plan *deployplan.Plan, remote map[string][]dresources.StatePermission) error {
	for resourceKey, entry := range plan.Plan {
		if err := addDeclaredPermissionsChanges(changes, b, resourceKey, entry); err != nil {
			return err
		}
	}

	for resourceKey, permissions := range remote {
		// Otherwise read for the job's owner only, see unreadPermissionsAdapters.
		if entry := plan.Plan[resourceKey+".permissions"]; entry != nil && entry.Action == deployplan.Create {
			if err := addUndeployedPermissionsChanges(changes, b, resourceKey, permissions); err != nil {
				return err
			}
		}
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
		change, err := convertPlanChange(parentKey, fieldPath, changeDesc, "", b.Config.Presets.NamePrefix)
		if err != nil {
			return fmt.Errorf("failed to compute config change for path %s: %w", resourceKey+"."+path, err)
		}
		if change != nil {
			changes.set(parentKey, fieldPath, change)
		}
	}
	return nil
}

// addUndeployedPermissionsChanges compares the remote ACL of a deployed resource with the
// permissions its config declares but that were never deployed.
// The plan creates those without reading the ACL, and creating them replaces it, so a grant made
// since would be dropped by the next deploy.
func addUndeployedPermissionsChanges(changes Changes, b *bundle.Bundle, resourceKey string, remote []dresources.StatePermission) error {
	permissionsKey := resourceKey + ".permissions"
	sv, ok := b.DeploymentBundle.StateCache.Load(permissionsKey)
	if !ok {
		return fmt.Errorf("no config state for %s", permissionsKey)
	}
	local := sv.Value.(*dresources.PermissionsState)
	// Without an entry for the current user, their default grants would be added as new principals.
	currentUser := currentUserName(b)
	if !slices.ContainsFunc(local.EmbeddedSlice, func(p dresources.StatePermission) bool { return isCurrentUserPermission(p, currentUser) }) {
		remote = withoutCurrentUserDefaults(slices.Clone(remote), currentUser, config.GetResourceTypeFromKey(resourceKey))
	}
	remoteState := &dresources.PermissionsState{ObjectID: local.ObjectID, EmbeddedSlice: remote}
	adapter := b.DeploymentBundle.Adapters[config.GetResourceTypeFromKey(permissionsKey)]
	diff, err := structdiff.GetStructDiff(remoteState, local, adapter.KeyedSlices())
	if err != nil {
		return fmt.Errorf("failed to compare permissions of %s: %w", resourceKey, err)
	}

	entry := &deployplan.PlanEntry{RemoteState: remoteState, Changes: make(deployplan.Changes)}
	for _, ch := range diff {
		entry.Changes[ch.Path.String()] = &deployplan.ChangeDesc{Action: deployplan.Update, New: ch.New, Remote: ch.Old}
	}
	return addDeclaredPermissionsChanges(changes, b, permissionsKey, entry)
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

// readRemotePermissions reads the remote permissions the sync needs and the plan does not read,
// keyed by resource. A resource whose ACL cannot be read is returned as skipped, so the rest of
// the sync still goes through.
func readRemotePermissions(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan) (map[string][]dresources.StatePermission, []SkippedChange) {
	var mu sync.Mutex
	var g errgroup.Group
	g.SetLimit(permissionsReadLimit)
	result := make(map[string][]dresources.StatePermission)
	var skipped []SkippedChange

	for resourceKey, adapter := range unreadPermissionsAdapters(b, plan) {
		g.Go(func() error {
			permissions, err := readPermissions(ctx, b, adapter, resourceKey)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				skipped = append(skipped, permissionsReadSkip(resourceKey, err))
				return nil
			}
			result[resourceKey] = permissions
			return nil
		})
	}

	_ = g.Wait()
	slices.SortFunc(skipped, func(a, b SkippedChange) int { return strings.Compare(a.ResourceKey, b.ResourceKey) })
	return result, skipped
}

// permissionsReadSkip reports a resource whose ACL could not be read. A user who can deploy a
// resource is not necessarily allowed to read its ACL.
func permissionsReadSkip(resourceKey string, err error) SkippedChange {
	reason := SkipReasonPermissionsReadFailed
	if errors.Is(err, apierr.ErrPermissionDenied) {
		reason = SkipReasonPermissionsPermissionDenied
	}
	return SkippedChange{ResourceKey: resourceKey, Reason: reason, Error: err.Error()}
}

// unreadPermissionsAdapters returns the permissions adapter of every deployed resource whose
// config declares permissions that were never deployed, and of every job without deployed
// permissions whose owner the run_as sync needs.
func unreadPermissionsAdapters(b *bundle.Bundle, plan *deployplan.Plan) map[string]*dresources.Adapter {
	result := make(map[string]*dresources.Adapter)
	for resourceKey, entry := range plan.Plan {
		if isPermissionsOrGrantsSubResource(resourceKey) || !entry.Action.KeepsID() {
			continue
		}
		permissionsKey := resourceKey + ".permissions"
		permissionsEntry := plan.Plan[permissionsKey]
		if permissionsEntry != nil && permissionsEntry.Action != deployplan.Delete && permissionsEntry.Action != deployplan.Create {
			continue
		}
		undeployed := permissionsEntry != nil && permissionsEntry.Action == deployplan.Create
		if !undeployed && !needsJobOwner(entry) {
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

// withoutCurrentUserDefaults drops the current user's grants that exist without being declared.
// For most resources these are IS_OWNER and CAN_MANAGE, one of which FixPermissions adds on
// deploy; another principal's IS_OWNER is kept, so a transferred owner stays the owner after the
// next deploy. For a cluster policy it is the CAN_USE the backend grants its creator: only admins
// create policies and admins can use every policy, so the grant is never needed in the config.
func withoutCurrentUserDefaults(permissions []dresources.StatePermission, currentUser, resourceType string) []dresources.StatePermission {
	return slices.DeleteFunc(permissions, func(p dresources.StatePermission) bool {
		if !isCurrentUserPermission(p, currentUser) {
			return false
		}
		if resourceType == "cluster_policies" {
			return p.Level == iam.PermissionLevelCanUse
		}
		return p.Level == iam.PermissionLevelIsOwner || p.Level == iam.PermissionLevelCanManage
	})
}

func isCurrentUserPermission(p dresources.StatePermission, currentUser string) bool {
	return currentUser != "" && (p.UserName == currentUser || p.ServicePrincipalName == currentUser)
}

func currentUserName(b *bundle.Bundle) string {
	if b.Config.Workspace.CurrentUser == nil {
		return ""
	}
	return b.Config.Workspace.CurrentUser.UserName
}
