package generate

import (
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
)

// ConvertResourceToValue serializes a bundle resource for a generated YAML file.
// Nil fields are dropped and the remaining keys are sorted alphabetically.
func ConvertResourceToValue(resource any) (dyn.Value, error) {
	return yamlsaver.ConvertToMapValue(resource, nil, nil, make(map[string]dyn.Value))
}
