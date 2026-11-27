package dresources

import (
	"reflect"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structwalk"
)

// RedactSensitiveConfigValues walks the bundle config and replaces the
// value of every field declared as sensitive_fields for its resource type with
// "[redacted]". It also redacts every sensitive variable, since variables are the
// standard mechanism for supplying secret values and their resolved values must
// not appear in the output of `bundle validate -o json`.
func RedactSensitiveConfigValues(root *config.Root) (*config.Root, error) {
	fields := getSensitiveFields()

	for resourceType, fieldRules := range fields {
		resources, err := structaccess.GetByString(root, "resources."+resourceType)
		if err != nil {
			return nil, err
		}
		err = structwalk.Walk(resources, func(path *structpath.PathNode, val any, _ *reflect.StructField) {
			for _, fieldRule := range fieldRules {
				// The first segment of the path is the resource key, so we need to skip it and check the rest of the path
				rest := path.SkipPrefix(1)
				if rest.HasPatternPrefix(fieldRule.Field) {
					_ = structaccess.SetByString(root, "resources."+resourceType+path.String(), sensitiveRedactedMarker)
				}
			}
		})
		if err != nil {
			return nil, err
		}
	}

	redactSensitiveVariables(root)

	return root, nil
}

const sensitiveRedactedMarker = "[redacted]"

// redactSensitiveVariables redacts the variables marked sensitive. The default is
// redacted too: it may reference another sensitive variable and has been resolved
// to the secret by the time the config is rendered.
func redactSensitiveVariables(root *config.Root) {
	for _, variable := range root.Variables {
		if variable == nil || !config.IsExplicitlyEnabled(variable.Sensitive) {
			continue
		}
		if variable.Default != nil {
			variable.Default = sensitiveRedactedMarker
		}
		if variable.Value != nil {
			variable.Value = sensitiveRedactedMarker
		}
	}
}

// getSensitiveFields returns the map of resource type to list of sensitive fields.
func getSensitiveFields() map[string][]FieldRule {
	cfg := MustLoadConfig()
	fields := make(map[string][]FieldRule)
	for resourceType, rc := range cfg.Resources {
		if rc.SensitiveFields == nil {
			continue
		}
		fields[resourceType] = rc.SensitiveFields
	}
	return fields
}

// SensitiveConfigValues returns the non-empty string values of the sensitive fields and
// sensitive variables in root, the values RedactSensitiveConfigValues replaces.
func SensitiveConfigValues(root *config.Root) []string {
	var values []string
	add := func(v any) {
		if s, ok := v.(string); ok && s != "" {
			values = append(values, s)
		}
	}
	for resourceType, fieldRules := range getSensitiveFields() {
		resources, err := structaccess.GetByString(root, "resources."+resourceType)
		if err != nil {
			continue
		}
		_ = structwalk.Walk(resources, func(path *structpath.PathNode, val any, _ *reflect.StructField) {
			rest := path.SkipPrefix(1)
			for _, fieldRule := range fieldRules {
				if rest.HasPatternPrefix(fieldRule.Field) {
					add(val)
				}
			}
		})
	}
	for _, variable := range root.Variables {
		if variable != nil && config.IsExplicitlyEnabled(variable.Sensitive) {
			add(variable.Default)
			add(variable.Value)
		}
	}
	return values
}

// IsSensitiveField reports whether field, a path inside a resource of resourceType, is a
// sensitive field, lies inside one, or contains one.
func IsSensitiveField(resourceType string, field *structpath.PathNode) bool {
	fieldStr := field.String()
	for _, rule := range MustLoadConfig().Resources[resourceType].SensitiveFields {
		if field.HasPatternPrefix(rule.Field) {
			return true
		}
		ruleStr := rule.Field.String()
		if strings.HasPrefix(ruleStr, fieldStr+".") || strings.HasPrefix(ruleStr, fieldStr+"[") {
			return true
		}
	}
	return false
}
