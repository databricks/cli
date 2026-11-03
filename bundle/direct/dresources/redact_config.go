package dresources

import (
	"reflect"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structwalk"
)

// RedactSensitiveConfigValues walks the bundle config and replaces the
// value of every field declared as sensitive_fields for its resource type with
// "[redacted]". It also redacts the entire variables section when any sensitive
// resource fields are present, since variables are the standard mechanism for
// supplying secret values and their resolved values must not appear in the
// output of `bundle validate -o json`.
func RedactSensitiveConfigValues(root *config.Root) (*config.Root, error) {
	fields := getSensitiveFields()

	var hasSensitiveFields bool
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
					hasSensitiveFields = true
					_ = structaccess.SetByString(root, "resources."+resourceType+path.String(), sensitiveRedactedMarker)
				}
			}
		})
		if err != nil {
			return nil, err
		}
	}

	// Redact all variable values when any sensitive resource fields are present.
	// Variables are the standard mechanism for supplying secret values and their
	// resolved values must not appear in plaintext in the validate output.
	if hasSensitiveFields {
		for _, variable := range root.Variables {
			if variable == nil {
				continue
			}
			if _, ok := variable.Value.(string); ok {
				variable.Value = sensitiveRedactedMarker
			}
		}
	}

	return root, nil
}

const sensitiveRedactedMarker = "[redacted]"

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
