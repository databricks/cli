package direct

import (
	"reflect"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/dyn/jsonloader"
	"github.com/databricks/cli/libs/structs/structpath"
)

var resourcesPrefix = dyn.MustPathFromString("resources")

// ResolveConfigAgainstState resolves ${resources.*} references within the resource at
// target so its runner ("bundle run") sees concrete values rather than references. For a
// reference resources.<group>.<name>.<field>, the value is taken from that resource's
// persisted state (the source of truth after deploy) when available, falling back to
// the config for anything not in state. Non-resource references are left untouched, and
// references that resolve to neither are left as-is rather than failing the command.
//
// Only the target resource is resolved: its runner is the sole consumer of resolved
// config, so resolving other resources is unnecessary work. It is also unsafe — an
// unrelated resource may reference a resource that was never deployed, whose ${...}.id
// falls back to an empty config string and then fails int type-checking.
//
// The state DB holds fields like the immutable snapshot's full_path that never reach the
// config, so it must be open.
func (b *DeploymentBundle) ResolveConfigAgainstState(cfg *config.Root, target dyn.Path) error {
	return cfg.Mutate(func(root dyn.Value) (dyn.Value, error) {
		// Fall back to the fully-normalized config so references to fields that are
		// implied (not explicitly set) still resolve.
		normalized, _ := convert.Normalize(cfg, root, convert.IncludeMissingFields)

		return dyn.MapByPath(root, target, func(_ dyn.Path, resource dyn.Value) (dyn.Value, error) {
			return dynvar.Resolve(resource, func(path dyn.Path) (dyn.Value, error) {
				if !path.HasPrefix(resourcesPrefix) {
					return dyn.InvalidValue, dynvar.ErrSkipResolution
				}
				if v, ok := b.lookupStateField(path); ok {
					return v, nil
				}
				v, err := dyn.GetByPath(normalized, path)
				if err != nil {
					return dyn.InvalidValue, dynvar.ErrSkipResolution
				}
				return v, nil
			})
		})
	})
}

// lookupStateField returns the value at resources.<group>.<name>.<field...> from the
// resource's persisted state, if that resource is in state and holds the field.
func (b *DeploymentBundle) lookupStateField(path dyn.Path) (dyn.Value, bool) {
	if len(path) < 4 || path[0].Key() != "resources" {
		return dyn.InvalidValue, false
	}
	resourceKey := "resources." + path[1].Key() + "." + path[2].Key()
	entry, ok := b.StateDB.GetResourceEntry(resourceKey)
	if !ok || len(entry.State) == 0 {
		return dyn.InvalidValue, false
	}
	stateVal, err := jsonloader.LoadJSON(entry.State, resourceKey)
	if err != nil {
		return dyn.InvalidValue, false
	}
	fieldVal, err := dyn.GetByPath(stateVal, path[3:])
	if err != nil || !fieldVal.IsValid() {
		return dyn.InvalidValue, false
	}
	return fieldVal, true
}

// ResolveAllResourcesAgainstState resolves ${resources.*} references in every resource
// for read-only commands that display resources (summary, open), so names and URLs show
// the deployed values instead of the references. Values come from the referenced
// resource's state, falling back to explicitly set config fields (including the ids that
// statemgmt.Load copies from state).
//
// Display is best effort, so a reference stays as it is when its value is not known or
// not safe to show:
//   - a resource that was never deployed: unlike ResolveConfigAgainstState there is no
//     fallback to implied empty fields, which would turn an id into "" (see #6690);
//   - a sensitive field (state stores it zeroed, config holds the secret), or a field
//     whose config references one, directly or through other references (its deployed
//     value in state holds the secret);
//   - a state field stored as a content hash, or a state string that holds a literal
//     ${...} (it would be interpolated again).
//
// A resource whose resolved values don't fit its Go type keeps its references.
func (b *DeploymentBundle) ResolveAllResourcesAgainstState(cfg *config.Root) {
	stateCache := map[string]dyn.Value{}
	_ = cfg.Mutate(func(root dyn.Value) (dyn.Value, error) {
		lookup := func(path dyn.Path) (dyn.Value, error) {
			if len(path) < 4 || !path.HasPrefix(resourcesPrefix) {
				return dyn.InvalidValue, dynvar.ErrSkipResolution
			}
			if dependsOnSensitiveField(root, path, map[string]bool{}) {
				return dyn.InvalidValue, dynvar.ErrSkipResolution
			}
			if v, ok := b.cachedStateField(stateCache, path); ok {
				if str, isStr := v.AsString(); isStr && (dresources.IsStateHashPlaceholder(str) || dynvar.ContainsVariableReference(str)) {
					return dyn.InvalidValue, dynvar.ErrSkipResolution
				}
				return v, nil
			}
			v, err := dyn.GetByPath(root, path)
			if err != nil || !v.IsValid() {
				return dyn.InvalidValue, dynvar.ErrSkipResolution
			}
			return v, nil
		}

		pattern := dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey())
		resolved, err := dyn.MapByPattern(root, pattern, func(p dyn.Path, resource dyn.Value) (dyn.Value, error) {
			// A resolved id is a string; normalize so it gets the field's type (e.g. int
			// job_id). A resource that fails to resolve or doesn't fit its type keeps its
			// references.
			if v, err := dynvar.Resolve(resource, lookup); err == nil {
				if normalized, ok := normalizeResource(p[1].Key(), v); ok {
					return normalized, nil
				}
			}
			return resource, nil
		})
		if err != nil {
			return dyn.InvalidValue, err
		}

		// Mutate leaves the typed config half-written if conversion fails, so check first.
		var check config.Root
		if err := convert.ToTyped(&check, resolved); err != nil {
			return dyn.InvalidValue, err
		}
		return resolved, nil
	})
}

// dependsOnSensitiveField reports whether path, resources.<group>.<name>.<field...>, is a
// sensitive field or its config value references one through ${resources.*} references.
func dependsOnSensitiveField(root dyn.Value, path dyn.Path, visited map[string]bool) bool {
	key := path.String()
	if visited[key] {
		return false
	}
	visited[key] = true

	field, err := structpath.ParsePath(path[3:].String())
	if err != nil || dresources.IsSensitiveField(path[1].Key(), field) {
		return true
	}

	value, err := dyn.GetByPath(root, path)
	if err != nil || !value.IsValid() {
		return false
	}
	found := false
	_ = dyn.WalkReadOnly(value, func(_ dyn.Path, leaf dyn.Value) error {
		ref, ok := dynvar.NewRef(leaf)
		if found || !ok {
			return nil
		}
		for _, s := range ref.References() {
			refPath, err := dyn.NewPathFromString(s)
			if err == nil && len(refPath) >= 4 && refPath.HasPrefix(resourcesPrefix) && dependsOnSensitiveField(root, refPath, visited) {
				found = true
				return nil
			}
		}
		return nil
	})
	return found
}

// normalizeResource normalizes v, a resource of the given group, to the resource's Go type
// and reports whether it converts to that type without errors.
func normalizeResource(group string, v dyn.Value) (dyn.Value, bool) {
	resourcesType := reflect.TypeFor[config.Resources]()
	for field := range resourcesType.Fields() {
		if strings.Split(field.Tag.Get("json"), ",")[0] != group || field.Type.Kind() != reflect.Map {
			continue
		}
		elemType := field.Type.Elem()
		if elemType.Kind() == reflect.Pointer {
			elemType = elemType.Elem()
		}
		typed := reflect.New(elemType).Interface()
		normalized, diags := convert.Normalize(typed, v)
		if len(diags) > 0 {
			return dyn.InvalidValue, false
		}
		if err := convert.ToTyped(typed, normalized); err != nil {
			return dyn.InvalidValue, false
		}
		return normalized, true
	}
	return dyn.InvalidValue, false
}

// cachedStateField is lookupStateField with each resource's parsed state cached in cache.
func (b *DeploymentBundle) cachedStateField(cache map[string]dyn.Value, path dyn.Path) (dyn.Value, bool) {
	resourceKey := "resources." + path[1].Key() + "." + path[2].Key()
	stateVal, ok := cache[resourceKey]
	if !ok {
		entry, found := b.StateDB.GetResourceEntry(resourceKey)
		if found && len(entry.State) > 0 {
			if v, err := jsonloader.LoadJSON(entry.State, resourceKey); err == nil {
				stateVal = v
			}
		}
		cache[resourceKey] = stateVal
	}
	if !stateVal.IsValid() {
		return dyn.InvalidValue, false
	}
	fieldVal, err := dyn.GetByPath(stateVal, path[3:])
	if err != nil || !fieldVal.IsValid() {
		return dyn.InvalidValue, false
	}
	return fieldVal, true
}
