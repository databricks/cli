package configsync

import (
	"context"
	"encoding/json"
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
	lookupPolicy := func(id string) (clusterPolicyRules, error) {
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
	structural, removal := false, false
	for _, path := range slices.Sorted(maps.Keys(changes)) {
		change := *changes[path]
		if change.Value != nil {
			fieldPath, err := structpath.ParsePath(path)
			if err != nil {
				return nil, nil, fmt.Errorf("parsing change path %s.%s: %w", resourceKey, path, err)
			}
			value, keep, err := filter.prune(fieldPath, change.Value, true)
			if err != nil {
				lookupErr, ok := errors.AsType[*clusterPolicyLookupError](err)
				if !ok || !errors.Is(err, apierr.ErrNotFound) {
					return nil, nil, fmt.Errorf("filtering policy values for %s.%s (no configuration was saved): %w", resourceKey, path, err)
				}
				structural = structural || !filter.canSkipField(fieldPath)
				skipped = append(skipped, SkippedChange{
					ResourceKey: resourceKey, Path: path,
					Reason: SkipReasonClusterPolicyNotFound, PolicyID: lookupErr.policyID,
				})
				continue
			}
			switch {
			case !keep:
				// Policy values never sync, even when they replaced an authored value.
				continue
			case change.Operation == OperationReplace && !reflect.DeepEqual(value, change.Value):
				// Dropping policy values can leave exactly the authored value.
				unchanged, err := filter.matchesAuthored(fieldPath, value)
				if err != nil {
					return nil, nil, fmt.Errorf("filtering policy values for %s.%s (no configuration was saved): %w", resourceKey, path, err)
				}
				if unchanged {
					continue
				}
			}
			change.Value = value
		}
		filtered[path] = &change
		// A removal may be paired with a skipped addition, e.g. autoscale -> num_workers.
		removal = removal || change.Operation == OperationRemove
	}
	// A structural change is always skipped too, so one length check covers both cases.
	if len(skipped) > 0 && (structural || removal) {
		return nil, []SkippedChange{{
			ResourceKey: resourceKey, Reason: SkipReasonClusterPolicyNotFound, PolicyID: skipped[0].PolicyID,
		}}, nil
	}
	return filtered, skipped, nil
}

type clusterPolicyRule struct {
	Type         string `json:"type"`
	Value        any    `json:"value"`
	DefaultValue any    `json:"defaultValue"`
}

type clusterPolicyRules map[string]clusterPolicyRule

type clusterPolicyDefinition struct {
	rules clusterPolicyRules
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

// get caches definitions and lookup failures per sync. The API returns the
// effective definition, including policy-family overrides.
func (definitions clusterPolicyDefinitions) get(ctx context.Context, b *bundle.Bundle, id string) (clusterPolicyRules, error) {
	if definition, ok := definitions[id]; ok {
		return definition.rules, definition.err
	}
	policy, err := b.WorkspaceClient(ctx).ClusterPolicies.GetByPolicyId(ctx, id)
	if err != nil {
		lookupErr := &clusterPolicyLookupError{policyID: id, err: err}
		definitions[id] = clusterPolicyDefinition{err: lookupErr}
		return nil, lookupErr
	}
	decoder := json.NewDecoder(strings.NewReader(policy.Definition))
	decoder.UseNumber()
	var rules clusterPolicyRules
	if err := decoder.Decode(&rules); err != nil {
		return nil, fmt.Errorf("parsing cluster policy %s: %w", id, err)
	}
	definitions[id] = clusterPolicyDefinition{rules: rules}
	return rules, nil
}

// supplied returns the value a rule sets, or nil if it sets none.
func (r clusterPolicyRule) supplied(applyDefaults bool) any {
	switch {
	case r.Type == "fixed":
		return r.Value
	case applyDefaults && r.Type != "forbidden":
		return r.DefaultValue
	}
	return nil
}

// supplies reports whether the rule could have set value. The backend rejects
// values that differ from a fixed rule, so a fixed field is frozen whatever its
// remote value; other rules only set their opted-in default.
func (r clusterPolicyRule) supplies(value any, applyDefaults bool) bool {
	if r.Type == "fixed" {
		return true
	}
	expected := r.supplied(applyDefaults)
	return expected != nil && policyValueEquals(expected, value)
}

// policyValueEquals compares a decoded policy value with a normalized remote value.
// Numbers compare by the remote type, and string fields such as spark_conf entries
// match the literal of a JSON number or boolean.
func policyValueEquals(policyValue, value any) bool {
	switch policyValue := policyValue.(type) {
	case json.Number:
		switch value := value.(type) {
		case string:
			return value == policyValue.String()
		case int64:
			i, err := policyValue.Int64()
			return err == nil && i == value
		case float64:
			f, err := policyValue.Float64()
			return err == nil && f == value
		}
		return false
	case bool:
		if value, ok := value.(string); ok {
			return value == strconv.FormatBool(policyValue)
		}
	}
	return reflect.DeepEqual(policyValue, value)
}

type clusterPolicyFilter struct {
	lookupPolicy func(string) (clusterPolicyRules, error)
	config       any
	remote       any
	clusterRoots []*structpath.PatternNode
	sliceKeys    map[string]structdiff.KeyFunc
}

type pruneAction int

const (
	pruneDescend pruneAction = iota // filter children individually
	pruneKeep                       // keep the value as is
	pruneDrop                       // the whole value comes from the policy
)

// prune removes omitted policy values from a normalized plan change, including
// whole-object additions. It reports keep=false when the whole value comes from the policy.
func (f *clusterPolicyFilter) prune(path *structpath.PathNode, value any, isChangeRoot bool) (any, bool, error) {
	action, err := f.classify(path, value, isChangeRoot)
	if err != nil {
		return nil, false, err
	}
	switch action {
	case pruneDrop:
		return nil, false, nil
	case pruneKeep:
		return value, true, nil
	default:
		return f.pruneChildren(path, value)
	}
}

func (f *clusterPolicyFilter) pruneChildren(path *structpath.PathNode, value any) (any, bool, error) {
	switch value := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(value))
		for _, key := range slices.Sorted(maps.Keys(value)) {
			filtered, keep, err := f.prune(structpath.NewStringKey(path, key), value[key], false)
			if err != nil {
				return nil, false, err
			}
			if keep {
				result[key] = filtered
			}
		}
		return result, true, nil
	case []any:
		// Arrays inside clusters are never descended into; elements here cannot be dropped.
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

// classify decides how prune handles a value. Values outside cluster specs are
// descended into because they may contain whole clusters.
func (f *clusterPolicyFilter) classify(path *structpath.PathNode, value any, isChangeRoot bool) (pruneAction, error) {
	clusterPath, ok := f.clusterPath(path)
	if !ok {
		return pruneDescend, nil
	}
	remotePolicyID, _ := structaccess.Get(f.remote, structpath.NewStringKey(clusterPath, "policy_id"))
	policyID, _ := remotePolicyID.(string)
	if policyID == "" {
		return pruneKeep, nil
	}
	fieldPath := path
	if f.sliceKeys != nil {
		// Whole-object changes use remote indices; authored tasks may have another order.
		var err error
		fieldPath, err = structdiff.KeyedPath(f.remote, path, f.sliceKeys)
		if err != nil {
			return 0, err
		}
		clusterPath = fieldPath.Prefix(clusterPath.Len())
	}
	// A change may start inside an array, so protecting only sequence
	// nodes would still allow an element to be emptied or removed.
	for _, node := range fieldPath.SkipPrefix(clusterPath.Len()).AsSlice() {
		if _, indexed := node.Index(); indexed {
			return pruneKeep, nil
		}
	}
	// Keep required fields, but still filter their optional children.
	if isChangeRoot || !isRequiredField(f.remote, fieldPath) {
		matches, err := f.isPolicyOnlySubtree(clusterPath, fieldPath, value, policyID)
		if err != nil {
			return 0, err
		}
		if matches {
			return pruneDrop, nil
		}
	}
	// Removing elements would change indexed policy constraints.
	if _, sequence := value.([]any); sequence {
		return pruneKeep, nil
	}
	return pruneDescend, nil
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
	switch derefType(reflect.TypeOf(remote)).Kind() {
	case reflect.Struct, reflect.Slice, reflect.Array:
		return false
	default:
		return true
	}
}

// isPolicyOnlySubtree requires every leaf to be absent from authored config and
// set by the remote cluster's policy: under a fixed rule, or equal to an opted-in default.
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

	rules, err := f.lookupPolicy(policyID)
	if err != nil {
		return false, err
	}
	// Remote cluster specs omit this input-only flag; use the authored setting.
	applyDefaults, err := f.authoredValue(structpath.NewStringKey(clusterPath, "apply_policy_default_values"))
	if err != nil {
		return false, err
	}
	for _, leaf := range leaves {
		if !policySupplies(rules, leaf.path.SkipPrefix(clusterPath.Len()), leaf.value, applyDefaults == true) {
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
	typ := reflect.TypeOf(parent)
	if typ == nil {
		return false
	}
	field, _, ok := structaccess.FindStructFieldByKeyType(derefType(typ), key)
	return ok && !structtag.JSONTag(field.Tag.Get("json")).OmitEmpty()
}

func derefType(typ reflect.Type) reflect.Type {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ
}

// policyAttributes renders a path relative to the cluster in the policy language,
// once with indices and once with "*" in their place. Dotted map keys stay one key,
// e.g. spark_conf["a.b"] becomes spark_conf.a.b.
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
func policySupplies(rules clusterPolicyRules, path *structpath.PathNode, value any, applyDefaults bool) bool {
	indexed, wildcard := policyAttributes(path)
	if indexed == "cluster_log_conf.s3.region" {
		return rules["cluster_log_conf.region"].supplies(value, applyDefaults)
	}
	if storage, ok := strings.CutPrefix(indexed, "cluster_log_conf."); ok && strings.HasSuffix(storage, ".destination") {
		storage = strings.ToUpper(strings.TrimSuffix(storage, ".destination"))
		logType := rules["cluster_log_conf.type"].supplied(applyDefaults)
		return policyValueEquals(logType, storage) && rules["cluster_log_conf.path"].supplies(value, applyDefaults)
	}
	rule, ok := rules[indexed]
	if !ok {
		rule = rules[wildcard]
	}
	return rule.supplies(value, applyDefaults)
}

func policyFilterForResource(b *bundle.Bundle, resourceKey string, remote any, lookupPolicy func(string) (clusterPolicyRules, error)) (*clusterPolicyFilter, error) {
	filter := &clusterPolicyFilter{remote: remote, lookupPolicy: lookupPolicy}
	switch remote.(type) {
	case *dresources.JobRemote:
		filter.clusterRoots = policyClusterRoots("jobs")
		filter.sliceKeys = (&dresources.ResourceJob{}).KeyedSlices()
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
		if rule.WhenSet.String() == "policy_id" {
			roots = append(roots, rule.Field)
		}
	}
	return roots
}
