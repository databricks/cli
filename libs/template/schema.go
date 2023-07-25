package template

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/databricks/cli/libs/jsonschema"
)

// function to check whether a float value represents an integer
func isIntegerValue(v float64) bool {
	return v == float64(int(v))
}

// cast value to integer for config values that are floats but are supposed to be
// integers according to the schema
//
// Needed because the default json unmarshaler for maps converts all numbers to floats
func castFloatConfigValuesToInt(config map[string]any, jsonSchema *jsonschema.Schema) error {
	for k, v := range config {
		// error because all config keys should be defined in schema too
		fieldInfo, ok := jsonSchema.Properties[k]
		if !ok {
			return fmt.Errorf("%s is not defined as an input parameter for the template", k)
		}
		// skip non integer fields
		if fieldInfo.Type != jsonschema.IntegerType {
			continue
		}

		// convert floating point type values to integer
		switch floatVal := v.(type) {
		case float32:
			if !isIntegerValue(float64(floatVal)) {
				return fmt.Errorf("expected %s to have integer value but it is %v", k, v)
			}
			config[k] = int(floatVal)
		case float64:
			if !isIntegerValue(floatVal) {
				return fmt.Errorf("expected %s to have integer value but it is %v", k, v)
			}
			config[k] = int(floatVal)
		}
	}
	return nil
}

func assignDefaultConfigValues(config map[string]any, schema *jsonschema.Schema) error {
	for k, v := range schema.Properties {
		if _, ok := config[k]; ok {
			continue
		}
		if v.Default == nil {
			return fmt.Errorf("input parameter %s is not defined in config", k)
		}
		config[k] = v.Default
	}
	return nil
}

func validateConfigValueTypes(config map[string]any, schema *jsonschema.Schema) error {
	// validate types defined in config
	for k, v := range config {
		fieldInfo, ok := schema.Properties[k]
		if !ok {
			return fmt.Errorf("%s is not defined as an input parameter for the template", k)
		}
		err := validateType(v, fieldInfo.Type)
		if err != nil {
			return fmt.Errorf("incorrect type for %s. %w", k, err)
		}
	}
	return nil
}

func ReadSchema(path string) (*jsonschema.Schema, error) {
	schemaBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	schema := &jsonschema.Schema{}
	err = json.Unmarshal(schemaBytes, schema)
	if err != nil {
		return nil, err
	}
	return schema, nil
}

func ReadConfig(path string, jsonSchema *jsonschema.Schema) (map[string]any, error) {
	// Read config file
	var config map[string]any
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &config)
	if err != nil {
		return nil, err
	}

	// Assign default value to any fields that do not have a value yet
	err = assignDefaultConfigValues(config, jsonSchema)
	if err != nil {
		return nil, err
	}

	// cast any fields that are supposed to be integers. The json unmarshalling
	// for a generic map converts all numbers to floating point
	err = castFloatConfigValuesToInt(config, jsonSchema)
	if err != nil {
		return nil, err
	}

	// validate config according to schema
	err = validateConfigValueTypes(config, jsonSchema)
	if err != nil {
		return nil, err
	}
	return config, nil
}
