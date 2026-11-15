package configsync

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/structs/structpath"
)

// applyLifecycle applies the direct engine's lifecycle metadata to a change, whatever
// action the plan gave it. The plan action is not enough on its own: it is still an
// update when the config has an undeployed edit to such a field, or when a resource's
// OverrideChangeDesc promotes it (a genie space etag).
//
// A remote value under ignore_remote_changes (output-only, input-only or backend-managed)
// is never written back, and neither is a backend default the config does not set. When
// the plan reports a whole object (one side is nil), the ignored fields inside it are
// left out of both sides, and the change is skipped if nothing else differs.
func applyLifecycle(resourceKey, fieldPath string, cd *deployplan.ChangeDesc) (*deployplan.ChangeDesc, bool, error) {
	path, err := structpath.ParsePath(fieldPath)
	if err != nil {
		return nil, false, err
	}
	resourceType := config.GetResourceTypeFromKey(resourceKey)
	backendDefault := cd.New == nil && cd.Remote != nil && !matchPattern(jobRunAsChildPattern, resourceKey+"."+fieldPath)

	var nested []*structpath.PatternNode
	for _, rc := range []*dresources.ResourceLifecycleConfig{
		dresources.GetResourceConfig(resourceType),
		dresources.GetGeneratedResourceConfig(resourceType),
	} {
		for _, rule := range rc.IgnoreRemoteChanges {
			if path.HasPatternPrefix(rule.Field) {
				return nil, true, nil
			}
			if rule.Field.Len() > path.Len() && path.HasPatternPrefix(rule.Field.Prefix(path.Len())) {
				nested = append(nested, rule.Field)
			}
		}
		if backendDefault && rc.IsBackendDefault(path, cd.Remote) {
			return nil, true, nil
		}
	}
	if len(nested) == 0 {
		return cd, false, nil
	}

	newValue, err := normalizeValue(cd.New)
	if err != nil {
		return nil, false, fmt.Errorf("failed to normalize config value: %w", err)
	}
	remoteValue, err := normalizeValue(cd.Remote)
	if err != nil {
		return nil, false, fmt.Errorf("failed to normalize remote value: %w", err)
	}
	newValue = stripIgnoredFields(nested, path, newValue)
	remoteValue = stripIgnoredFields(nested, path, remoteValue)
	if reflect.DeepEqual(newValue, remoteValue) {
		return nil, true, nil
	}

	stripped := *cd
	stripped.New = newValue
	stripped.Remote = remoteValue
	return &stripped, false, nil
}

// stripIgnoredFields drops the fields of value, the normalized value at path, that
// match one of the rules. A map left empty only by dropping fields becomes nil.
func stripIgnoredFields(rules []*structpath.PatternNode, path *structpath.PathNode, value any) any {
	switch v := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(v))
		for key, val := range v {
			child := structpath.NewStringKey(path, key)
			if slices.ContainsFunc(rules, child.HasPatternPrefix) {
				continue
			}
			if s := stripIgnoredFields(rules, child, val); s != nil || val == nil {
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
			result[i] = stripIgnoredFields(rules, structpath.NewIndex(path, i), elem)
		}
		return result
	}
	return value
}
