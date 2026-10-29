package configsync

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/clusterpolicy"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structtag"
	"github.com/databricks/databricks-sdk-go/apierr"
)

// clusterPolicyGate is the cluster spec field that attaches a policy. The direct engine's
// remote addition rules gated on it mark where the backend applies policy values.
const clusterPolicyGate = "policy_id"

// FilterClusterPolicyValues removes policy-applied values from already extracted
// and selected changes. The plan and input changes are read-only. A resource whose
// policy cannot be read is left out entirely and reported as skipped: its changes may
// pair with each other (e.g. autoscale -> num_workers), so a partial result is unsafe.
func FilterClusterPolicyValues(ctx context.Context, b *bundle.Bundle, plan *deployplan.Plan, changes Changes) (Changes, []SkippedChange, error) {
	definitions := make(clusterPolicyDefinitions)
	lookupPolicy := func(id string) (clusterpolicy.Definition, error) {
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
		resourceChanges, err := filter.filterChanges(changes[resourceKey])
		if lookupErr, ok := errors.AsType[*clusterPolicyLookupError](err); ok {
			skipped = append(skipped, lookupErr.skippedChange(resourceKey))
			continue
		}
		if err != nil {
			return nil, nil, fmt.Errorf("filtering policy values for %s: %w", resourceKey, err)
		}
		if len(resourceChanges) > 0 {
			filtered[resourceKey] = resourceChanges
		}
	}
	return filtered, skipped, nil
}

type clusterPolicyDefinition struct {
	rules clusterpolicy.Definition
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

func (e *clusterPolicyLookupError) skippedChange(resourceKey string) SkippedChange {
	skip := SkippedChange{ResourceKey: resourceKey, Reason: SkipReasonClusterPolicyReadFailed, PolicyID: e.policyID, Error: e.err.Error()}
	if errors.Is(e.err, apierr.ErrNotFound) {
		skip.Reason, skip.Error = SkipReasonClusterPolicyNotFound, ""
	}
	return skip
}

// get caches definitions and failures per sync. The API returns the effective
// definition, including policy-family overrides.
func (definitions clusterPolicyDefinitions) get(ctx context.Context, b *bundle.Bundle, id string) (clusterpolicy.Definition, error) {
	if definition, ok := definitions[id]; ok {
		return definition.rules, definition.err
	}
	var rules clusterpolicy.Definition
	policy, err := b.WorkspaceClient(ctx).ClusterPolicies.GetByPolicyId(ctx, id)
	if err == nil {
		rules, err = clusterpolicy.ParseDefinition(policy.Definition)
	}
	if err != nil {
		err = &clusterPolicyLookupError{policyID: id, err: err}
	}
	definitions[id] = clusterPolicyDefinition{rules: rules, err: err}
	return rules, err
}

// elementSupplies reports whether the policy element could have set value. The backend
// rejects values that differ from a fixed element, so a fixed field is frozen whatever its
// remote value; other elements only set their opted-in default.
func elementSupplies(element clusterpolicy.Element, value any, applyDefaults bool) bool {
	if element.Type == "fixed" {
		return true
	}
	expected := element.EffectiveValue(applyDefaults)
	return expected != nil && clusterpolicy.ValueEquals(expected, value)
}

type clusterPolicyFilter struct {
	lookupPolicy func(string) (clusterpolicy.Definition, error)
	config       any
	remote       any
	clusterRoots []*structpath.PatternNode
}

func (f *clusterPolicyFilter) filterChanges(changes ResourceChanges) (ResourceChanges, error) {
	filtered := make(ResourceChanges, len(changes))
	for _, path := range slices.Sorted(maps.Keys(changes)) {
		change := *changes[path]
		if change.Value != nil {
			fieldPath, err := structpath.ParsePath(path)
			if err != nil {
				return nil, fmt.Errorf("parsing change path %s: %w", path, err)
			}
			value, keep, err := f.prune(fieldPath, change.Value, true)
			if err != nil {
				return nil, err
			}
			if !keep {
				// Policy values never sync, even when they replaced an authored value.
				continue
			}
			if change.Operation == OperationReplace && !reflect.DeepEqual(value, change.Value) {
				// Dropping policy values can leave exactly the authored value.
				unchanged, err := f.matchesAuthored(fieldPath, value)
				if err != nil {
					return nil, err
				}
				if unchanged {
					continue
				}
			}
			change.Value = value
		}
		filtered[path] = &change
	}
	return filtered, nil
}

// prune removes policy values from a normalized change value, including whole-object
// additions. It reports keep=false when the whole value comes from the policy. Values
// outside cluster specs are descended into because they may contain whole clusters.
func (f *clusterPolicyFilter) prune(path *structpath.PathNode, value any, isChangeRoot bool) (any, bool, error) {
	clusterPath, inCluster := f.clusterPath(path)
	if !inCluster {
		return f.pruneChildren(path, value)
	}
	remotePolicyID, _ := structaccess.Get(f.remote, structpath.NewStringKey(clusterPath, clusterPolicyGate))
	policyID, _ := remotePolicyID.(string)
	if policyID == "" {
		return value, true, nil
	}
	// A change may start inside an array, so protecting only sequence
	// nodes would still allow an element to be emptied or removed.
	for _, node := range path.SkipPrefix(clusterPath.Len()).AsSlice() {
		if _, indexed := node.Index(); indexed {
			return value, true, nil
		}
	}
	// Keep required fields, but still filter their optional children.
	if isChangeRoot || !isRequiredField(f.remote, path) {
		policyOnly, err := f.isPolicyOnlySubtree(clusterPath, path, value, policyID)
		if err != nil {
			return nil, false, err
		}
		if policyOnly {
			return nil, false, nil
		}
	}
	// Removing elements would change indexed policy constraints.
	if _, sequence := value.([]any); sequence {
		return value, true, nil
	}
	return f.pruneChildren(path, value)
}

func (f *clusterPolicyFilter) pruneChildren(path *structpath.PathNode, value any) (any, bool, error) {
	switch value := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(value))
		for _, key := range slices.Sorted(maps.Keys(value)) {
			child, keep, err := f.prune(structpath.NewStringKey(path, key), value[key], false)
			if err != nil {
				return nil, false, err
			}
			if keep {
				result[key] = child
			}
		}
		return result, true, nil
	case []any:
		// Only reached outside cluster specs, e.g. a whole tasks array added remotely;
		// arrays inside a cluster are kept as is above. Elements are never dropped.
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

// clusterPath finds the cluster spec containing a field; the spec itself is excluded.
func (f *clusterPolicyFilter) clusterPath(path *structpath.PathNode) (*structpath.PathNode, bool) {
	for _, root := range f.clusterRoots {
		if path.HasPatternPrefix(root) && path.Len() > root.Len() {
			return path.Prefix(root.Len()), true
		}
	}
	return nil, false
}

// isPolicyOnlySubtree requires every leaf to be absent from authored config and
// set by the remote cluster's policy: under a fixed element, or equal to an opted-in default.
func (f *clusterPolicyFilter) isPolicyOnlySubtree(clusterPath, path *structpath.PathNode, value any, policyID string) (bool, error) {
	leaves := appendLeaves(nil, path, value)
	if len(leaves) == 0 {
		return false, nil
	}
	for _, leaf := range leaves {
		authored, err := f.authoredValue(leaf.path)
		if err != nil {
			return false, err
		}
		if authored != nil {
			return false, nil
		}
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
	for _, leaf := range leaves {
		if !policySupplies(definition, leaf.path.SkipPrefix(clusterPath.Len()), leaf.value, applyDefaults == true) {
			return false, nil
		}
	}
	return true, nil
}

// authoredValue returns the config value at path, or nil when the config does not declare
// it. Change paths select keyed slice elements by key, so they address the same element in
// config and remote regardless of order.
func (f *clusterPolicyFilter) authoredValue(path *structpath.PathNode) (any, error) {
	value, err := structaccess.Get(f.config, path)
	if _, missing := errors.AsType[*structaccess.NotFoundError](err); missing {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading authored field %s: %w", path, err)
	}
	return value, nil
}

func (f *clusterPolicyFilter) matchesAuthored(path *structpath.PathNode, value any) (bool, error) {
	authored, err := f.authoredValue(path)
	if err != nil {
		return false, err
	}
	normalized, err := normalizeValue(authored)
	if err != nil {
		return false, err
	}
	return reflect.DeepEqual(normalized, value), nil
}

type policyLeaf struct {
	path  *structpath.PathNode
	value any
}

// appendLeaves collects the normalized change's scalars, including nulls.
// Empty containers have no leaves and cannot alone establish a policy match.
func appendLeaves(leaves []policyLeaf, path *structpath.PathNode, value any) []policyLeaf {
	switch value := value.(type) {
	case map[string]any:
		for _, key := range slices.Sorted(maps.Keys(value)) {
			leaves = appendLeaves(leaves, structpath.NewStringKey(path, key), value[key])
		}
	case []any:
		for i, child := range value {
			leaves = appendLeaves(leaves, structpath.NewIndex(path, i), child)
		}
	default:
		leaves = append(leaves, policyLeaf{path: path, value: value})
	}
	return leaves
}

func isRequiredField(remote any, path *structpath.PathNode) bool {
	key, ok := path.StringKey()
	if !ok {
		return false
	}
	parent, _ := structaccess.Get(remote, path.Parent())
	parentValue := reflect.Indirect(reflect.ValueOf(parent))
	if !parentValue.IsValid() || parentValue.Kind() != reflect.Struct {
		return false
	}
	field, _, ok := structaccess.FindStructFieldByKeyType(parentValue.Type(), key)
	return ok && !structtag.JSONTag(field.Tag.Get("json")).OmitEmpty()
}

// policyAttributes renders a path relative to the cluster in the policy language,
// once with indices and once with "*" in their place. Dotted map keys stay one key,
// e.g. spark_conf["a.b"] becomes spark_conf.a.b. Map keys are not wildcarded: a rule
// such as spark_conf.* constrains values but is not treated as supplying them.
func policyAttributes(path *structpath.PathNode) (indexed, wildcard string) {
	var indexedParts, wildcardParts []string
	for _, node := range path.AsSlice() {
		if key, ok := node.StringKey(); ok {
			indexedParts = append(indexedParts, key)
			wildcardParts = append(wildcardParts, key)
		} else if index, ok := node.Index(); ok {
			indexedParts = append(indexedParts, strconv.Itoa(index))
			wildcardParts = append(wildcardParts, "*")
		}
	}
	return strings.Join(indexedParts, "."), strings.Join(wildcardParts, ".")
}

// policySupplies reports whether the policy could have set a leaf, translating the
// policy language's log-delivery aliases. Indexed rules take precedence over wildcards.
// https://docs.databricks.com/aws/en/admin/clusters/policy-definition
func policySupplies(definition clusterpolicy.Definition, path *structpath.PathNode, value any, applyDefaults bool) bool {
	indexed, wildcard := policyAttributes(path)
	if indexed == "cluster_log_conf.s3.region" {
		return elementSupplies(definition["cluster_log_conf.region"], value, applyDefaults)
	}
	if storage, ok := strings.CutPrefix(indexed, "cluster_log_conf."); ok && strings.HasSuffix(storage, ".destination") {
		storage = strings.ToUpper(strings.TrimSuffix(storage, ".destination"))
		logType := definition["cluster_log_conf.type"].EffectiveValue(applyDefaults)
		return clusterpolicy.ValueEquals(logType, storage) && elementSupplies(definition["cluster_log_conf.path"], value, applyDefaults)
	}
	element, ok := definition[indexed]
	if !ok {
		element = definition[wildcard]
	}
	return elementSupplies(element, value, applyDefaults)
}

func policyFilterForResource(b *bundle.Bundle, resourceKey string, remote any, lookupPolicy func(string) (clusterpolicy.Definition, error)) (*clusterPolicyFilter, error) {
	filter := &clusterPolicyFilter{remote: remote, lookupPolicy: lookupPolicy}
	switch remote.(type) {
	case *dresources.JobRemote:
		filter.clusterRoots = policyClusterRoots("jobs")
	case *dresources.ClusterRemote:
		filter.clusterRoots = policyClusterRoots("clusters")
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

// policyClusterRoots reuses the direct engine's policy-gated remote addition rules,
// which mark where the backend applies policy values. A nil root is the resource itself.
func policyClusterRoots(resourceType string) []*structpath.PatternNode {
	var roots []*structpath.PatternNode
	for _, rule := range dresources.GetResourceConfig(resourceType).IgnoreRemoteAdditions {
		if rule.WhenSet.String() == clusterPolicyGate {
			roots = append(roots, rule.Field)
		}
	}
	return roots
}
