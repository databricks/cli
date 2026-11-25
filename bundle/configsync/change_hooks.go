package configsync

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
)

// changeHook holds what a resource type decides about plan changes beyond its rules
// file, because it depends on more than a field's path and value. Every hook is asked
// about every change, so it must only match the paths of its own resource type. Paths
// are full change paths, such as "resources.<type>.<name>.<field>".
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

var changeHooks = map[string]changeHook{}

// registerChangeHook registers the change hook of the resource type with the given key.
func registerChangeHook(resourceType string, hook changeHook) {
	if _, ok := changeHooks[resourceType]; ok {
		panic(fmt.Sprintf("change hook for %q registered twice", resourceType))
	}
	changeHooks[resourceType] = hook
}

// sortedChangeHooks returns the registered change hooks in order of resource key.
var sortedChangeHooks = sync.OnceValue(func() []changeHook {
	var hooks []changeHook
	for _, resourceType := range slices.Sorted(maps.Keys(changeHooks)) {
		hooks = append(hooks, changeHooks[resourceType])
	}
	return hooks
})

func keepSkippedChange(path string, cd *deployplan.ChangeDesc) bool {
	return slices.ContainsFunc(sortedChangeHooks(), func(hook changeHook) bool { return hook.keepSkipped(path, cd) })
}

func syncsBackendDefault(path string) bool {
	return slices.ContainsFunc(sortedChangeHooks(), func(hook changeHook) bool { return hook.syncsBackendDefault(path) })
}

// newChangeSkip returns the changes that any change hook says must not be synced.
func newChangeSkip(ctx context.Context, plan *deployplan.Plan, remote map[string][]dresources.StatePermission) changeSkip {
	var skips []changeSkip
	for _, hook := range sortedChangeHooks() {
		skips = append(skips, hook.newSkip(ctx, plan, remote))
	}
	return func(resourceKey, path string, cd *deployplan.ChangeDesc) bool {
		return slices.ContainsFunc(skips, func(skip changeSkip) bool { return skip(resourceKey, path, cd) })
	}
}
