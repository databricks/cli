package clusterpolicy

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
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
