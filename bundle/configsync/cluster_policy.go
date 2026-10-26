package configsync

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/jsonloader"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structtag"
	"github.com/databricks/databricks-sdk-go/apierr"
)

// FilterClusterPolicyValues removes policy-applied values from already extracted
// and selected changes. The plan and input changes are read-only.
func FilterClusterPolicyValues(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan, changes Changes) (Changes, []SkippedChange, error) {
	definitions := make(clusterPolicyDefinitions)
	lookupPolicy := func(id string) (dyn.Value, error) {
		return definitions.get(ctx, b, id)
	}
	filtered := make(Changes, len(changes))
	var skipped []SkippedChange
	for _, resourceKey := range slices.Sorted(maps.Keys(changes)) {
		filter, err := policyFilterForResource(b, resourceKey, plan.Plan[resourceKey].RemoteState, lookupPolicy)
		if err != nil {
			return nil, nil, err
		}
		if filter == nil {
			filtered[resourceKey] = changes[resourceKey]
			continue
		}
		resourceChanges, resourceSkipped, err := filterClusterPolicyChanges(resourceKey, changes[resourceKey], filter)
		if err != nil {
			return nil, nil, err
		}
		if len(resourceChanges) > 0 {
			filtered[resourceKey] = resourceChanges
		}
		skipped = append(skipped, resourceSkipped...)
	}
	return filtered, skipped, nil
}

// filterClusterPolicyChanges preserves atomic replacements and paired removals
// when a missing policy prevents classifying part of a resource's changes.
func filterClusterPolicyChanges(resourceKey string, changes ResourceChanges, filter *clusterPolicyFilter) (ResourceChanges, []SkippedChange, error) {
	filtered := make(ResourceChanges, len(changes))
	var skipped []SkippedChange
	hasStructuralSkip, hasRemoval := false, false
	for _, path := range slices.Sorted(maps.Keys(changes)) {
		change := *changes[path]
		if change.Value != nil {
			fieldPath, err := structpath.ParsePath(path)
			if err != nil {
				return nil, nil, fmt.Errorf("parsing change path %s.%s: %w", resourceKey, path, err)
			}
			change.Value, err = filter.filter(fieldPath, change.Value)
			if err != nil {
				lookupErr, ok := errors.AsType[*clusterPolicyLookupError](err)
				if !ok || !errors.Is(err, apierr.ErrNotFound) {
					return nil, nil, fmt.Errorf("filtering policy values for %s.%s (no configuration was saved): %w", resourceKey, path, err)
				}
				hasStructuralSkip = hasStructuralSkip || !filter.canSkipField(fieldPath)
				skipped = append(skipped, SkippedChange{
					ResourceKey: resourceKey, Path: path,
					Reason: SkipReasonClusterPolicyNotFound, PolicyID: lookupErr.policyID,
				})
				continue
			}
			if change.Value == nil {
				if change.Operation != OperationReplace {
					continue
				}
				// A policy-only replacement still removes the previous authored value.
				change.Operation = OperationRemove
			}
		}
		filtered[path] = &change
		// A removal may be paired with a skipped addition, e.g. autoscale -> num_workers.
		hasRemoval = hasRemoval || change.Operation == OperationRemove
	}
	if hasStructuralSkip || (len(skipped) > 0 && hasRemoval) {
		return nil, []SkippedChange{{
			ResourceKey: resourceKey, Reason: SkipReasonClusterPolicyNotFound, PolicyID: skipped[0].PolicyID,
		}}, nil
	}
	return filtered, skipped, nil
}

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
	lookupPolicy func(string) (dyn.Value, error)
	config       any
	remote       any
	clusterRoots []*structpath.PatternNode
	sliceKeys    map[string]structdiff.KeyFunc
}

// filter removes omitted policy values from a normalized plan change, including
// whole-object additions. A nil result means the value comes entirely from the policy.
func (f *clusterPolicyFilter) filter(path *structpath.PathNode, value any) (any, error) {
	filtered, keep, err := f.prune(path, value, true)
	if err != nil || !keep {
		return nil, err
	}
	return filtered, nil
}

// prune reports keep=false when the whole value comes from the cluster policy.
func (f *clusterPolicyFilter) prune(path *structpath.PathNode, value any, isChangeRoot bool) (any, bool, error) {
	if clusterPath, ok := f.clusterPath(path); ok {
		remotePolicyID, _ := structaccess.Get(f.remote, structpath.NewStringKey(clusterPath, "policy_id"))
		policyID, _ := remotePolicyID.(string)
		if policyID == "" {
			return value, true, nil
		}
		fieldPath := path
		if f.sliceKeys != nil {
			// Whole-object changes use remote indices; authored tasks may have another order.
			var err error
			fieldPath, err = structdiff.KeyedPath(f.remote, path, f.sliceKeys)
			if err != nil {
				return nil, false, err
			}
			clusterPath = fieldPath.Prefix(clusterPath.Len())
		}
		// A change may start inside an array, so protecting only sequence
		// nodes would still allow an element to be emptied or removed.
		for _, node := range fieldPath.SkipPrefix(clusterPath.Len()).AsSlice() {
			if _, indexed := node.Index(); indexed {
				return value, true, nil
			}
		}
		// Keep required fields, but still filter their optional children.
		if isChangeRoot || !isRequiredField(f.remote, fieldPath) {
			matches, err := f.isPolicyOnlySubtree(clusterPath, fieldPath, value, policyID)
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
			filtered, keep, err := f.prune(structpath.NewStringKey(path, key), child, false)
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
			filtered, _, err := f.prune(structpath.NewIndex(path, i), child, false)
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
func (f *clusterPolicyFilter) isPolicyOnlySubtree(clusterPath, path *structpath.PathNode, value any, policyID string) (bool, error) {
	hasLeaves := false
	for leafPath := range normalizedLeaves(path, value) {
		hasLeaves = true
		authored, err := f.authoredValue(leafPath)
		if err != nil {
			return false, err
		}
		if authored != nil {
			return false, nil
		}
	}
	if !hasLeaves {
		return false, nil
	}

	definition, err := f.lookupPolicy(policyID)
	if err != nil {
		return false, err
	}
	// Remote cluster specs omit this input-only flag; use the authored setting.
	applyDefaults, err := f.authoredValue(structpath.NewStringKey(clusterPath, "apply_policy_default_values"))
	if err != nil {
		return false, err
	}
	for leafPath, leaf := range normalizedLeaves(path, value) {
		expected := policyValueForAPIAttribute(definition, policyAttribute(leafPath.SkipPrefix(clusterPath.Len())), applyDefaults == true)
		if !expected.IsValid() || expected.AsAny() == nil || !reflect.DeepEqual(expected.AsAny(), leaf) {
			return false, nil
		}
	}
	return true, nil
}

// authoredValue validates selectors before looking up fields so duplicate task
// identities cannot make an authored value appear absent.
func (f *clusterPolicyFilter) authoredValue(path *structpath.PathNode) (any, error) {
	var value any
	keyed, err := structdiff.KeyedPath(f.config, path, f.sliceKeys)
	if err == nil {
		value, err = structaccess.Get(f.config, keyed)
	}
	if _, missing := errors.AsType[*structaccess.NotFoundError](err); missing {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading authored field %s: %w", path, err)
	}
	return value, nil
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

func policyFilterForResource(b *bundle.Bundle, resourceKey string, remote any, lookupPolicy func(string) (dyn.Value, error)) (*clusterPolicyFilter, error) {
	filter := &clusterPolicyFilter{remote: remote, lookupPolicy: lookupPolicy}
	switch remote.(type) {
	case *dresources.JobRemote:
		var err error
		filter.clusterRoots, err = jobClusterRoots()
		if err != nil {
			return nil, err
		}
		filter.sliceKeys = (&dresources.ResourceJob{}).KeyedSlices()
	case *dresources.ClusterRemote:
		filter.clusterRoots = []*structpath.PatternNode{nil} // nil matches the resource root
	default:
		// Only jobs and clusters expand policy values into their remote state.
		return nil, nil
	}
	// Skip entries have no NewState; authored values must come from bundle config.
	config, err := b.Config.GetResourceConfig(resourceKey)
	if err != nil {
		return nil, fmt.Errorf("reading config for %s: %w", resourceKey, err)
	}
	filter.config = config
	return filter, nil
}

var jobClusterRoots = sync.OnceValues(func() ([]*structpath.PatternNode, error) {
	var roots []*structpath.PatternNode
	for _, pattern := range []string{"tasks[*].new_cluster", "tasks[*].for_each_task.task.new_cluster", "job_clusters[*].new_cluster"} {
		root, err := structpath.ParsePattern(pattern)
		if err != nil {
			return nil, err
		}
		roots = append(roots, root)
	}
	return roots, nil
})
