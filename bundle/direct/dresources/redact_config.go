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
// "[redacted]". This is used before printing the full config to stdout (e.g.
// `bundle validate -o json`) so plaintext secrets are never shown to the user.
//
// The function only handles resource types that have at least one sensitive_field
// declared in resources.yml
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
	return root, nil
}

const sensitiveRedactedMarker = "[redacted]"

// getSensiti
// veFields returns one dyn.Pattern per sensitive field rule across
// all resource types. Each pattern covers: resources.<type>.*.<field_path>.
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
