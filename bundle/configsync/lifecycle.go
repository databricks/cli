package configsync

import (
	"reflect"
	"slices"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/structs/structpath"
)

// applyLifecycle applies the direct engine's lifecycle metadata to a change, whatever
// action the plan gave it, and returns nil when nothing is left to sync. The plan action
// is not enough on its own: it is still an update when the config has an undeployed edit
// to such a field, or when a resource's OverrideChangeDesc promotes it (a genie space etag).
//
// A remote value under ignore_remote_changes (output-only, input-only or backend-managed)
// is never written back, and neither is a backend default the config does not set. When
// the plan reports a whole object (one side is nil), these fields are left out of it, and
// the change is skipped if nothing else differs. Inside a remote-only object, only
// defaults pinned to allowed values are left out: a field like node_type_id may be what
// the user picked in the UI.
func applyLifecycle(resourceKey, fieldPath string, cd *deployplan.ChangeDesc) (*deployplan.ChangeDesc, error) {
	path, err := structpath.ParsePath(fieldPath)
	if err != nil {
		return nil, err
	}
	resourceType := config.GetResourceTypeFromKey(resourceKey)
	backendDefault := cd.New == nil && cd.Remote != nil && !isJobRunAsPath(resourceKey+"."+fieldPath)

	var f nestedFieldFilter
	for _, rc := range []*dresources.ResourceLifecycleConfig{
		dresources.GetResourceConfig(resourceType),
		dresources.GetGeneratedResourceConfig(resourceType),
	} {
		for _, rule := range rc.IgnoreRemoteChanges {
			if path.HasPatternPrefix(rule.Field) {
				return nil, nil
			}
			if isNestedPattern(path, rule.Field) {
				f.ignored = append(f.ignored, rule.Field)
			}
		}
		if backendDefault && rc.IsBackendDefault(path, cd.Remote) {
			return nil, nil
		}
		if cd.New == nil {
			for _, rule := range rc.BackendDefaults {
				if len(rule.Values) > 0 && isNestedPattern(path, rule.Field) {
					f.defaults.BackendDefaults = append(f.defaults.BackendDefaults, rule)
				}
			}
		}
	}
	if len(f.ignored) == 0 && len(f.defaults.BackendDefaults) == 0 {
		return cd, nil
	}

	newValue := normalizeValue(cd.New)
	remoteValue := normalizeValue(cd.Remote)
	newValue = stripFields(path, newValue, f.dropFromConfig)
	remoteValue = stripFields(path, remoteValue, f.dropFromRemote)
	if reflect.DeepEqual(newValue, remoteValue) {
		return nil, nil
	}

	stripped := *cd
	stripped.New = newValue
	stripped.Remote = remoteValue
	return &stripped, nil
}

// isNestedPattern reports whether pattern names a field inside the value at path.
func isNestedPattern(path *structpath.PathNode, pattern *structpath.PatternNode) bool {
	return pattern.Len() > path.Len() && path.HasPatternPrefix(pattern.Prefix(path.Len()))
}

// nestedFieldFilter holds the lifecycle rules that name fields inside a whole-object change.
type nestedFieldFilter struct {
	ignored  []*structpath.PatternNode
	defaults dresources.ResourceLifecycleConfig
}

func (f *nestedFieldFilter) dropFromConfig(path *structpath.PathNode, _ any) bool {
	return slices.ContainsFunc(f.ignored, path.HasPatternPrefix)
}

func (f *nestedFieldFilter) dropFromRemote(path *structpath.PathNode, value any) bool {
	return f.dropFromConfig(path, value) || f.defaults.IsBackendDefault(path, value)
}

// stripFields drops the fields of value, the normalized value at path, for which drop
// returns true. A map left empty only by dropping fields becomes nil.
func stripFields(path *structpath.PathNode, value any, drop func(*structpath.PathNode, any) bool) any {
	switch v := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(v))
		for key, val := range v {
			child := structpath.NewStringKey(path, key)
			if drop(child, val) {
				continue
			}
			if s := stripFields(child, val, drop); s != nil || val == nil {
				result[key] = s
			}
		}
		if len(result) == 0 && len(v) > 0 {
			return nil
		}
		return result
	case []any:
		result := make([]any, len(v))
		for i, elem := range v {
			result[i] = stripFields(structpath.NewIndex(path, i), elem, drop)
		}
		return result
	}
	return value
}
