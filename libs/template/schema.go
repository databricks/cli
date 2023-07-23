package template

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
)

// This is a JSON Schema compliant struct that we use to do validation checks on
// the provided configuration
type Schema struct {
	// A list of properties that can be used in the config
	Properties map[string]Property `json:"properties"`
}

type PropertyType string

const (
	PropertyTypeString  = PropertyType("string")
	PropertyTypeInt     = PropertyType("integer")
	PropertyTypeNumber  = PropertyType("number")
	PropertyTypeBoolean = PropertyType("boolean")
)

type Property struct {
	Type        PropertyType `json:"type"`
	Description string       `json:"description"`
	Default     any          `json:"default"`
}

// function to check whether a float value represents an integer
func isIntegerValue(v float64) bool {
	return v == float64(int(v))
}

// cast value to integer for config values that are floats but are supposed to be
// integers according to the schema
//
// Needed because the default json unmarshaler for maps converts all numbers to floats
func castFloatConfigValuesToInt(config map[string]any, schema *Schema) error {
	for k, v := range config {
		// error because all config keys should be defined in schema too
		if _, ok := schema.Properties[k]; !ok {
			return fmt.Errorf("%s is not defined as an input parameter for the template", k)
		}

		// skip non integer fields
		fieldInfo := schema.Properties[k]
		if fieldInfo.Type != PropertyTypeInt {
			continue
		}

		// convert floating point type values to integer
		valueType := reflect.TypeOf(v)
		switch valueType.Kind() {
		case reflect.Float32:
			floatVal := v.(float32)
			if !isIntegerValue(float64(floatVal)) {
				return fmt.Errorf("expected %s to have integer value but it is %v", k, v)
			}
			config[k] = int(floatVal)
		case reflect.Float64:
			floatVal := v.(float64)
			if !isIntegerValue(floatVal) {
				return fmt.Errorf("expected %s to have integer value but it is %v", k, v)
			}
			config[k] = int(floatVal)
		}
	}
	return nil
}

func assignDefaultConfigValues(config map[string]any, schema *Schema) error {
	for k, v := range schema.Properties {
		if _, ok := config[k]; !ok {
			if v.Default == nil {
				return fmt.Errorf("input parameter %s is not defined in config", k)
			}
			config[k] = v.Default
		}
	}
	return nil
}

func validateConfigValueTypes(config map[string]any, schema *Schema) error {
	// validate types defined in config
	for k, v := range config {
		fieldMetadata, ok := schema.Properties[k]
		if !ok {
			return fmt.Errorf("%s is not defined as an input parameter for the template", k)
		}
		err := validateType(v, fieldMetadata.Type)
		if err != nil {
			return fmt.Errorf("incorrect type for %s. %w", k, err)
		}
	}
	return nil
}

func ReadSchema(path string) (*Schema, error) {
	schemaBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	schema := &Schema{}
	err = json.Unmarshal(schemaBytes, schema)
	if err != nil {
		return nil, err
	}
	return schema, nil
}

func (schema *Schema) ReadConfig(path string) (map[string]any, error) {
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
	err = assignDefaultConfigValues(config, schema)
	if err != nil {
		return nil, err
	}

	// cast any fields that are supposed to be integers. The json unmarshalling
	// for a generic map converts all numbers to floating point
	err = castFloatConfigValuesToInt(config, schema)
	if err != nil {
		return nil, err
	}

	// validate config according to schema
	err = validateConfigValueTypes(config, schema)
	if err != nil {
		return nil, err
	}
	return config, nil
}
