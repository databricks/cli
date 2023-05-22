package template

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
)

type Schema map[string]FieldInfo

type FieldType string

const (
	FieldTypeString  = FieldType("string")
	FieldTypeInt     = FieldType("integer")
	FieldTypeFloat   = FieldType("float")
	FieldTypeBoolean = FieldType("boolean")
)

type FieldInfo struct {
	Type        FieldType `json:"type"`
	Description string    `json:"description"`
	Validation  string    `json:"validation"`
}

// function to check whether a float value represents an integer
func isIntegerValue(v float64) bool {
	return v == float64(int(v))
}

// cast value to integer for config values that are floats but are supposed to be
// integeres according to the schema
//
// Needed because the default json unmarshaller for maps converts all numbers to floats
func castFloatToInt(config map[string]any, schema Schema) error {
	for k, v := range config {
		// error because all config keys should be defined in schema too
		if _, ok := schema[k]; !ok {
			return fmt.Errorf("%s is not defined as an input parameter for the template", k)
		}

		// skip non integer fields
		fieldInfo := schema[k]
		if fieldInfo.Type != FieldTypeInt {
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

func validateType(v any, fieldType FieldType) error {
	validateFunc, ok := validators[fieldType]
	if !ok {
		return nil
	}
	return validateFunc(v)
}

func (schema Schema) ValidateConfig(config map[string]any) error {
	// validate types defined in config
	for k, v := range config {
		fieldMetadata, ok := schema[k]
		if !ok {
			return fmt.Errorf("%s is not defined as an input parameter for the template", k)
		}
		err := validateType(v, fieldMetadata.Type)
		if err != nil {
			return fmt.Errorf("incorrect type for %s. %w", k, err)
		}
	}
	// assert all fields are defined in
	for k := range schema {
		if _, ok := config[k]; !ok {
			return fmt.Errorf("input parameter %s is not defined in config", k)
		}
	}
	return nil
}

func ReadSchema(path string) (Schema, error) {
	schemaBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	schema := Schema{}
	err = json.Unmarshal(schemaBytes, &schema)
	if err != nil {
		return nil, err
	}
	return schema, nil
}

func (schema Schema) ReadConfig(path string) (map[string]any, error) {
	var config map[string]any
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &config)
	if err != nil {
		return nil, err
	}

	// cast any fields that are supposed to be integers. The json unmarshalling
	// for a generic map converts all numbers to floating point
	err = castFloatToInt(config, schema)
	if err != nil {
		return nil, err
	}

	// validate config according to schema
	err = schema.ValidateConfig(config)
	if err != nil {
		return nil, err
	}
	return config, nil
}
