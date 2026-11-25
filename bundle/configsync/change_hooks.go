package configsync

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
)

// changeHook holds what a resource type decides about plan changes beyond its rules
// file, because it depends on more than a field's path and value. A hook is consulted
// only for changes of the resource type it is registered for. Every field is optional.
// Paths are full change paths, such as "resources.<type>.<name>.<field>".
type changeHook struct {
	// keepSkipped reports whether a change the plan skipped is still considered for syncing.
	keepSkipped func(path string, cd *deployplan.ChangeDesc) bool

	// syncsBackendDefault reports whether the field at path is synced even where the
	// direct engine's lifecycle rules say its remote value is a backend default.
	syncsBackendDefault func(path string) bool

	// newSkip is called once per ExtractChanges with the plan and the ACLs read by
	// readRemotePermissions, and returns which changes must not be synced.
	newSkip func(ctx context.Context, plan *deployplan.Plan, remote map[string][]dresources.StatePermission) changeSkip
}

// changeSkip reports whether a change at path of the resource under resourceKey must
// not be synced.
type changeSkip func(resourceKey, path string, cd *deployplan.ChangeDesc) bool

// changeHookRegistry holds the change hooks by resource type.
type changeHookRegistry map[string]changeHook

var changeHooks = changeHookRegistry{}

// registerChangeHook registers the change hook of the resource type with the given key.
func registerChangeHook(resourceType string, hook changeHook) {
	changeHooks.register(resourceType, hook)
}

func (r changeHookRegistry) register(resourceType string, hook changeHook) {
	if _, ok := r[resourceType]; ok {
		panic(fmt.Sprintf("change hook for %q registered twice", resourceType))
	}
	r[resourceType] = hook
}

// keepSkipped reports whether the hook of the resource under resourceKey keeps a
// change the plan skipped.
func (r changeHookRegistry) keepSkipped(resourceKey, path string, cd *deployplan.ChangeDesc) bool {
	hook := r[config.GetResourceTypeFromKey(resourceKey)]
	return hook.keepSkipped != nil && hook.keepSkipped(path, cd)
}

// syncsBackendDefault reports whether the hook of the resource under resourceKey
// syncs the field at path as a backend default.
func (r changeHookRegistry) syncsBackendDefault(resourceKey, path string) bool {
	hook := r[config.GetResourceTypeFromKey(resourceKey)]
	return hook.syncsBackendDefault != nil && hook.syncsBackendDefault(path)
}

// newSkip returns the changes that the hook of their resource type says must not be
// synced. It calls the hooks' newSkip in order of resource key.
func (r changeHookRegistry) newSkip(ctx context.Context, plan *deployplan.Plan, remote map[string][]dresources.StatePermission) changeSkip {
	skips := make(map[string]changeSkip)
	for _, resourceType := range slices.Sorted(maps.Keys(r)) {
		if newSkip := r[resourceType].newSkip; newSkip != nil {
			skips[resourceType] = newSkip(ctx, plan, remote)
		}
	}
	return func(resourceKey, path string, cd *deployplan.ChangeDesc) bool {
		skip := skips[config.GetResourceTypeFromKey(resourceKey)]
		return skip != nil && skip(resourceKey, path, cd)
	}
}
