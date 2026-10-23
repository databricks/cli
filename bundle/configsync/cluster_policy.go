package configsync

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/jsonloader"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structtag"
)

type clusterPolicyDefinitions map[string]dyn.Value

// get reads the effective definition, including policy-family overrides, once per sync.
func (definitions clusterPolicyDefinitions) get(ctx context.Context, b *bundle.Bundle, id string) (dyn.Value, error) {
	if definition, ok := definitions[id]; ok {
		return definition, nil
	}
	policy, err := b.WorkspaceClient(ctx).ClusterPolicies.GetByPolicyId(ctx, id)
	if err != nil {
		return dyn.InvalidValue, fmt.Errorf("reading cluster policy %s: %w", id, err)
	}
	definition, err := jsonloader.LoadJSON([]byte(policy.Definition), "cluster policy "+id)
	if err != nil {
		return dyn.InvalidValue, err
	}
	definitions[id] = definition
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
// whole-object additions. A nil result means no value needs to be synced.
func (f *clusterPolicyFilter) filter(ctx context.Context, b *bundle.Bundle, path *structpath.PathNode, changeValue any) (any, error) {
	normalized, err := convert.FromTyped(changeValue, dyn.NilValue)
	if err != nil {
		return nil, err
	}
	filtered, err := dyn.Walk(normalized, func(relative dyn.Path, value dyn.Value) (dyn.Value, error) {
		fieldPath, err := structdiff.KeyedPath(f.remote, appendDynPath(path, relative), f.sliceKeys)
		if err != nil {
			return dyn.InvalidValue, err
		}
		for _, root := range f.clusterRoots {
			if !fieldPath.HasPatternPrefix(root) || fieldPath.Len() == root.Len() {
				continue
			}
			clusterPath := fieldPath.Prefix(root.Len())
			// A change may start inside an array, so protecting only sequence
			// nodes would still allow an element to be emptied or removed.
			for _, node := range fieldPath.SkipPrefix(root.Len()).AsSlice() {
				if _, indexed := node.Index(); indexed {
					return value, dyn.ErrSkip
				}
			}
			// Keep required fields, but still filter their optional children.
			if len(relative) > 0 && isRequiredField(f.remote, fieldPath) {
				return value, nil
			}
			matches, err := f.matchesOmittedPolicySubtree(ctx, b, clusterPath, fieldPath, value)
			if err != nil {
				return dyn.InvalidValue, err
			}
			if matches {
				return dyn.InvalidValue, dyn.ErrDrop
			}
			// Removing elements would change indexed policy constraints.
			if value.Kind() == dyn.KindSequence {
				return value, dyn.ErrSkip
			}
			break
		}
		return value, nil
	})
	if errors.Is(err, dyn.ErrDrop) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return filtered.AsAny(), nil
}

// matchesOmittedPolicySubtree requires every leaf to be absent from authored config
// and equal to a fixed value or an opted-in default in the remote cluster's policy.
func (f *clusterPolicyFilter) matchesOmittedPolicySubtree(ctx context.Context, b *bundle.Bundle, clusterPath, path *structpath.PathNode, value dyn.Value) (bool, error) {
	matches := true
	leaves := 0
	_, err := dyn.Walk(value, func(relative dyn.Path, leaf dyn.Value) (dyn.Value, error) {
		if !matches {
			return leaf, dyn.ErrSkip
		}
		if leaf.Kind() == dyn.KindMap || leaf.Kind() == dyn.KindSequence {
			return leaf, nil
		}
		leaves++
		fieldPath := appendDynPath(path, relative)
		if authored, _ := structaccess.Get(f.config, fieldPath); authored != nil {
			matches = false
			return leaf, nil
		}
		remotePolicyID, _ := structaccess.Get(f.remote, structpath.NewStringKey(clusterPath, "policy_id"))
		policyID, _ := remotePolicyID.(string)
		if policyID == "" {
			matches = false
			return leaf, nil
		}
		definition, err := f.definitions.get(ctx, b, policyID)
		if err != nil {
			return dyn.InvalidValue, err
		}
		// Remote cluster specs omit this input-only flag; use the authored setting.
		applyDefaults, _ := structaccess.Get(f.config, structpath.NewStringKey(clusterPath, "apply_policy_default_values"))
		expected := policyValueForAPIAttribute(definition, policyAttribute(fieldPath.SkipPrefix(clusterPath.Len())), applyDefaults == true)
		matches = expected.IsValid() && expected.AsAny() != nil && reflect.DeepEqual(expected.AsAny(), leaf.AsAny())
		return leaf, nil
	})
	return matches && leaves > 0, err
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

func appendDynPath(base *structpath.PathNode, suffix dyn.Path) *structpath.PathNode {
	for _, component := range suffix {
		if key := component.Key(); key != "" {
			base = structpath.NewStringKey(base, key)
		} else {
			base = structpath.NewIndex(base, component.Index())
		}
	}
	return base
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
