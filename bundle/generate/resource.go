package generate

import (
	"encoding/json"
	"slices"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
)

// cliManagedFields are populated by the CLI on every bundle resource and never
// belong in configuration a user maintains.
var cliManagedFields = []string{"id", "modified_status", "url"}

// FromResponse narrows an API response to the bundle resource type C by
// round-tripping it through JSON. Only fields present in C's schema survive.
// The response must share C's field layout: wrapper responses (for example a
// struct with a single `experiment` field) have to be unwrapped by the caller.
func FromResponse[C any](response any) (C, error) {
	var resource C
	data, err := json.Marshal(response)
	if err != nil {
		return resource, err
	}
	err = json.Unmarshal(data, &resource)
	return resource, err
}

// ConvertResourceToValue serializes a bundle resource for a generated YAML file,
// dropping nil fields, CLI-managed fields, and skipFields.
func ConvertResourceToValue(resource any, skipFields ...string) (dyn.Value, error) {
	return yamlsaver.ConvertToMapValue(resource, nil, slices.Concat(cliManagedFields, skipFields), make(map[string]dyn.Value))
}
