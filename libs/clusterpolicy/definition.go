package clusterpolicy

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"

	"github.com/databricks/cli/libs/structs/structpath"
)

// Element is the part of a cluster policy element that materializes a value. The backend
// supports more element types, but only these fields supply one: "fixed" elements set Value,
// every limiting type ("allowlist", "blocklist", "regex", "range", "unlimited") may carry
// DefaultValue. "forbidden" only rejects.
// https://docs.databricks.com/aws/en/admin/clusters/policy-definition
type Element struct {
	Type         string `json:"type"`
	Value        any    `json:"value"`
	DefaultValue any    `json:"defaultValue"`
}

// Definition maps policy attribute paths (e.g. "spark_version", "custom_tags.CostCenter")
// to their elements.
type Definition map[string]Element

// ParseDefinition decodes a policy definition. Numbers stay json.Number so integer and
// float literals compare exactly against typed cluster fields.
func ParseDefinition(definition string) (Definition, error) {
	decoder := json.NewDecoder(strings.NewReader(definition))
	decoder.UseNumber()
	var result Definition
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// EffectiveValue returns the value the element materializes into a spec that omits the
// attribute, or nil if it supplies none. "fixed" applies whether or not the request sets
// apply_policy_default_values, "defaultValue" only when it does. Pinned against a real
// workspace by acceptance/bundle/resources/jobs/cluster_policy (fixed_addition, default_flag).
func (e Element) EffectiveValue(applyDefaults bool) any {
	if e.Type == "fixed" {
		return e.Value
	}
	if applyDefaults {
		return e.DefaultValue
	}
	return nil
}

// ValueEquals compares a decoded policy value with a cluster field value. Numbers compare
// by the field's type, and string fields such as spark_conf entries match the literal of a
// JSON number or boolean.
func ValueEquals(policyValue, value any) bool {
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

// IsPolicyValue reports whether the policy, not the user, determines the value of the
// cluster attribute at path, a path relative to the cluster spec. A fixed element
// determines the field whatever its current value, because the backend rejects any other
// value; other elements only when apply_policy_default_values is set and value equals
// their defaultValue. This is a filtering predicate, not validation: true for a fixed
// element does not mean value equals the element's value. The policy language's
// log-delivery aliases are translated, and indexed rules take precedence over wildcards.
// https://docs.databricks.com/aws/en/admin/clusters/policy-definition
func (d Definition) IsPolicyValue(path *structpath.PathNode, value any, applyDefaults bool) bool {
	indexed, wildcard := attributes(path)
	if indexed == "cluster_log_conf.s3.region" {
		return d["cluster_log_conf.region"].isPolicyValue(value, applyDefaults)
	}
	if storage, ok := strings.CutPrefix(indexed, "cluster_log_conf."); ok && strings.HasSuffix(storage, ".destination") {
		storage = strings.ToUpper(strings.TrimSuffix(storage, ".destination"))
		logType := d["cluster_log_conf.type"].EffectiveValue(applyDefaults)
		return ValueEquals(logType, storage) && d["cluster_log_conf.path"].isPolicyValue(value, applyDefaults)
	}
	element, ok := d[indexed]
	if !ok {
		element = d[wildcard]
	}
	return element.isPolicyValue(value, applyDefaults)
}

func (e Element) isPolicyValue(value any, applyDefaults bool) bool {
	if e.Type == "fixed" {
		return true
	}
	expected := e.EffectiveValue(applyDefaults)
	return expected != nil && ValueEquals(expected, value)
}

// attributes renders a path relative to the cluster in the policy language,
// once with indices and once with "*" in their place. Dotted map keys stay one key,
// e.g. spark_conf["a.b"] becomes spark_conf.a.b. Map keys are not wildcarded: a rule
// such as spark_conf.* constrains values but is not treated as supplying them.
func attributes(path *structpath.PathNode) (indexed, wildcard string) {
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
