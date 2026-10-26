package configsync

import (
	"context"
	"fmt"
	"iter"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/jsonloader"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structtag"
)

type clusterPolicyDefinition struct {
	value dyn.Value
	err   error
}

type clusterPolicyDefinitions map[string]clusterPolicyDefinition

type clusterPolicyLookupError struct {
	policyID string
	err      error
}

func (e *clusterPolicyLookupError) Error() string {
	return fmt.Sprintf("reading cluster policy %s: %v", e.policyID, e.err)
}

func (e *clusterPolicyLookupError) Unwrap() error { return e.err }

// get caches effective definitions, including policy-family overrides, and lookup failures per sync.
func (definitions clusterPolicyDefinitions) get(ctx context.Context, b *bundle.Bundle, id string) (dyn.Value, error) {
	if definition, ok := definitions[id]; ok {
		return definition.value, definition.err
	}
	policy, err := b.WorkspaceClient(ctx).ClusterPolicies.GetByPolicyId(ctx, id)
	if err != nil {
		lookupErr := &clusterPolicyLookupError{policyID: id, err: err}
		definitions[id] = clusterPolicyDefinition{err: lookupErr}
		return dyn.InvalidValue, lookupErr
	}
	definition, err := jsonloader.LoadJSON([]byte(policy.Definition), "cluster policy "+id)
	if err != nil {
		return dyn.InvalidValue, err
	}
	definitions[id] = clusterPolicyDefinition{value: definition}
	return definition, nil
}

type clusterPolicyFilter struct {
	definitions  clusterPolicyDefinitions
	config       any
	remote       any
	clusterRoots []*structpath.PatternNode
	sliceKeys    map[string]structdiff.KeyFunc
}

// filter removes omitted policy values from a normalized plan change, including
// whole-object additions. A nil result means the value comes entirely from the policy.
func (f *clusterPolicyFilter) filter(ctx context.Context, b *bundle.Bundle, path *structpath.PathNode, value any) (any, error) {
	filtered, keep, err := f.prune(ctx, b, path, value, true)
	if err != nil || !keep {
		return nil, err
	}
	return filtered, nil
}

// prune reports keep=false when the whole value comes from the cluster policy.
func (f *clusterPolicyFilter) prune(ctx context.Context, b *bundle.Bundle, path *structpath.PathNode, value any, isChangeRoot bool) (any, bool, error) {
	fieldPath, err := structdiff.KeyedPath(f.remote, path, f.sliceKeys)
	if err != nil {
		return nil, false, err
	}
	if clusterPath, ok := f.clusterPath(fieldPath); ok {
		// A change may start inside an array, so protecting only sequence
		// nodes would still allow an element to be emptied or removed.
		for _, node := range fieldPath.SkipPrefix(clusterPath.Len()).AsSlice() {
			if _, indexed := node.Index(); indexed {
				return value, true, nil
			}
		}
		// Keep required fields, but still filter their optional children.
		if isChangeRoot || !isRequiredField(f.remote, fieldPath) {
			matches, err := f.isPolicyOnlySubtree(ctx, b, clusterPath, fieldPath, value)
			if err != nil {
				return nil, false, err
			}
			if matches {
				return nil, false, nil
			}
		}
		// Removing elements would change indexed policy constraints.
		if _, sequence := value.([]any); sequence {
			return value, true, nil
		}
	}

	switch value := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(value))
		for _, key := range slices.Sorted(maps.Keys(value)) {
			child := value[key]
			filtered, keep, err := f.prune(ctx, b, structpath.NewStringKey(path, key), child, false)
			if err != nil {
				return nil, false, err
			}
			if keep {
				result[key] = filtered
			}
		}
		return result, true, nil
	case []any:
		// Arrays inside clusters return above; elements here cannot be dropped.
		result := make([]any, len(value))
		for i, child := range value {
			filtered, _, err := f.prune(ctx, b, structpath.NewIndex(path, i), child, false)
			if err != nil {
				return nil, false, err
			}
			result[i] = filtered
		}
		return result, true, nil
	default:
		return value, true, nil
	}
}

// clusterPath finds the cluster containing a field; the cluster itself is excluded.
func (f *clusterPolicyFilter) clusterPath(path *structpath.PathNode) (*structpath.PathNode, bool) {
	for _, root := range f.clusterRoots {
		if path.HasPatternPrefix(root) && path.Len() > root.Len() {
			return path.Prefix(root.Len()), true
		}
	}
	return nil, false
}

// canSkipField allows scalar and map changes strictly inside a cluster spec.
// The caller must also check for removals paired with a skipped addition.
func (f *clusterPolicyFilter) canSkipField(path *structpath.PathNode) bool {
	if _, ok := f.clusterPath(path); !ok {
		return false
	}
	remote, err := structaccess.Get(f.remote, path)
	if err != nil || remote == nil {
		return false
	}
	typ := reflect.TypeOf(remote)
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	switch typ.Kind() {
	case reflect.Struct, reflect.Slice, reflect.Array:
		return false
	default:
		return true
	}
}

// isPolicyOnlySubtree requires every leaf to be absent from authored config
// and equal to a fixed value or an opted-in default in the remote cluster's policy.
func (f *clusterPolicyFilter) isPolicyOnlySubtree(ctx context.Context, b *bundle.Bundle, clusterPath, path *structpath.PathNode, value any) (bool, error) {
	hasLeaves := false
	for leafPath := range normalizedLeaves(path, value) {
		hasLeaves = true
		if authored, _ := structaccess.Get(f.config, leafPath); authored != nil {
			return false, nil
		}
	}
	if !hasLeaves {
		return false, nil
	}

	remotePolicyID, _ := structaccess.Get(f.remote, structpath.NewStringKey(clusterPath, "policy_id"))
	policyID, _ := remotePolicyID.(string)
	if policyID == "" {
		return false, nil
	}
	definition, err := f.definitions.get(ctx, b, policyID)
	if err != nil {
		return false, err
	}
	// Remote cluster specs omit this input-only flag; use the authored setting.
	applyDefaults, _ := structaccess.Get(f.config, structpath.NewStringKey(clusterPath, "apply_policy_default_values"))
	for leafPath, leaf := range normalizedLeaves(path, value) {
		expected := policyValueForAPIAttribute(definition, policyAttribute(leafPath.SkipPrefix(clusterPath.Len())), applyDefaults == true)
		if !expected.IsValid() || expected.AsAny() == nil || !reflect.DeepEqual(expected.AsAny(), leaf) {
			return false, nil
		}
	}
	return true, nil
}

// normalizedLeaves preserves the normalized change's scalars, including nulls.
// Empty containers have no leaves and cannot alone establish a policy match.
func normalizedLeaves(path *structpath.PathNode, value any) iter.Seq2[*structpath.PathNode, any] {
	return func(yield func(*structpath.PathNode, any) bool) {
		switch value := value.(type) {
		case map[string]any:
			for _, key := range slices.Sorted(maps.Keys(value)) {
				for leafPath, leaf := range normalizedLeaves(structpath.NewStringKey(path, key), value[key]) {
					if !yield(leafPath, leaf) {
						return
					}
				}
			}
		case []any:
			for i, child := range value {
				for leafPath, leaf := range normalizedLeaves(structpath.NewIndex(path, i), child) {
					if !yield(leafPath, leaf) {
						return
					}
				}
			}
		default:
			yield(path, value)
		}
	}
}

func isRequiredField(remote any, path *structpath.PathNode) bool {
	key, ok := path.StringKey()
	if !ok {
		return false
	}
	parent, _ := structaccess.Get(remote, path.Parent())
	typ := reflect.TypeOf(parent)
	if typ == nil {
		return false
	}
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	field, _, ok := structaccess.FindStructFieldByKeyType(typ, key)
	return ok && !structtag.JSONTag(field.Tag.Get("json")).OmitEmpty()
}

// policyAttribute preserves dotted map keys, e.g. spark_conf["a.b"] becomes spark_conf.a.b.
func policyAttribute(path *structpath.PathNode) string {
	var parts []string
	for _, node := range path.AsSlice() {
		if key, ok := node.StringKey(); ok {
			parts = append(parts, key)
		} else if index, ok := node.Index(); ok {
			parts = append(parts, strconv.Itoa(index))
		}
	}
	return strings.Join(parts, ".")
}

// policyValueForAPIAttribute translates the policy language's log-delivery aliases.
// https://docs.databricks.com/aws/en/admin/clusters/policy-definition
func policyValueForAPIAttribute(definition dyn.Value, attribute string, applyDefaults bool) dyn.Value {
	if attribute == "cluster_log_conf.s3.region" {
		attribute = "cluster_log_conf.region"
	}
	if storage, ok := strings.CutPrefix(attribute, "cluster_log_conf."); ok {
		if storage, ok := strings.CutSuffix(storage, ".destination"); ok {
			logType := policyRuleValue(definition, "cluster_log_conf.type", applyDefaults)
			if !logType.IsValid() || logType.AsAny() != strings.ToUpper(storage) {
				return dyn.InvalidValue
			}
			return policyRuleValue(definition, "cluster_log_conf.path", applyDefaults)
		}
	}
	return policyRuleValue(definition, attribute, applyDefaults)
}

// policyRuleValue returns the value supplied by a rule, or InvalidValue if it supplies none.
func policyRuleValue(definition dyn.Value, attribute string, applyDefaults bool) dyn.Value {
	rule, err := dyn.GetByPath(definition, dyn.NewPath(dyn.Key(attribute)))
	if err != nil {
		return dyn.InvalidValue
	}
	ruleType, _ := dyn.Get(rule, "type")
	var value dyn.Value
	switch {
	case ruleType.AsAny() == "fixed":
		value, _ = dyn.Get(rule, "value")
	case applyDefaults && ruleType.AsAny() != "forbidden":
		value, _ = dyn.Get(rule, "defaultValue")
	}
	return value
}
