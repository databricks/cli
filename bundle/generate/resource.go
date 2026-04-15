package generate

import (
	"encoding/json"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlsaver"
)

var generatedResourceSkipFields = []string{
	"id",
	"modified_status",
	"url",
}

// ConvertResourceToValue converts a resource config struct into generated bundle YAML.
func ConvertResourceToValue(resource any, skipFields []string, dst map[string]dyn.Value) (dyn.Value, error) {
	if dst == nil {
		dst = make(map[string]dyn.Value)
	}

	skipFields = append(append([]string{}, generatedResourceSkipFields...), skipFields...)
	return yamlsaver.ConvertToMapValue(resource, nil, skipFields, dst)
}

// ConvertResponseToValue maps an SDK response into a bundle resource config and serializes it.
func ConvertResponseToValue[Config any](response any, skipFields []string, dst map[string]dyn.Value) (dyn.Value, error) {
	data, err := json.Marshal(response)
	if err != nil {
		return dyn.InvalidValue, err
	}

	var resource Config
	err = json.Unmarshal(data, &resource)
	if err != nil {
		return dyn.InvalidValue, err
	}

	return ConvertResourceToValue(resource, skipFields, dst)
}
