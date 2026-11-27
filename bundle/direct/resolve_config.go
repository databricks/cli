package direct

import (
	"errors"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/dyn/jsonloader"
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
// Unlike ResolveConfigAgainstState, it does not fall back to implied empty fields: a
// reference to a resource that was never deployed stays a reference rather than turning
// into "" (which an int field such as job_id would reject, see #6690). If the resolved
// config does not type-check anyway, the config is left unchanged: display is best effort.
func (b *DeploymentBundle) ResolveAllResourcesAgainstState(cfg *config.Root) {
	if !b.StateDB.IsOpen() {
		return
	}
	_ = cfg.Mutate(func(root dyn.Value) (dyn.Value, error) {
		pattern := dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey())
		resolved, err := dyn.MapByPattern(root, pattern, func(_ dyn.Path, resource dyn.Value) (dyn.Value, error) {
			return dynvar.Resolve(resource, func(path dyn.Path) (dyn.Value, error) {
				if !path.HasPrefix(resourcesPrefix) {
					return dyn.InvalidValue, dynvar.ErrSkipResolution
				}
				if v, ok := b.lookupStateField(path); ok {
					return v, nil
				}
				v, err := dyn.GetByPath(root, path)
				if err != nil || !v.IsValid() {
					return dyn.InvalidValue, dynvar.ErrSkipResolution
				}
				return v, nil
			})
		})
		if err != nil {
			return dyn.InvalidValue, err
		}
		// A resolved id is a string; normalize so it gets the field's type (e.g. int job_id).
		normalized, diags := convert.Normalize(cfg, resolved)
		if len(diags) > 0 {
			return dyn.InvalidValue, errors.New("resolved resources do not match their types")
		}
		return normalized, nil
	})
}
