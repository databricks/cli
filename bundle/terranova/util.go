package terranova

import (
	"fmt"

	"github.com/databricks/cli/libs/dyn"
)

func GetRequiredString(config dyn.Value, path string) (string, error) {
	value := dyn.GetByString(config, path)
	if !value.IsValid() {
		return "", fmt.Errorf("Missing required field %#v", path)
	}

	s, ok := value.AsString()
	if !ok {
		return "", fmt.Errorf("Field %#v must be string, got %s", path, value.Kind().String())
	}

	return s, nil
}

func GetRequiredNonemptyString(config dyn.Value, path string) (string, error) {
	result, err := GetRequiredString(config, path)
	if err != nil {
		return "", err
	}

	if result == "" {
		return "", fmt.Errorf("Field %#v must not be empty", path)
	}

	return result, nil
}
