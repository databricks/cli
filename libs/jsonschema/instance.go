package jsonschema

import (
	"encoding/json"
	"fmt"
	"os"
)

// Load a JSON document, also validating it against the JSON schema. Instance here
// refers to a JSON document. see: https://json-schema.org/draft/2020-12/json-schema-core.html#name-instance
func (s *Schema) LoadInstance(path string) (map[string]any, error) {
	instance := make(map[string]any, 0)
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &instance)
	if err != nil {
		return nil, err
	}

	// The default JSON unmarshaler parses untyped number values as float64.
	// We convert integer properties from float64 to int64 here.
	for name, v := range instance {
		propertySchema, ok := s.Properties[name]
		if !ok {
			continue
		}
		if propertySchema.Type != IntegerType {
			continue
		}
		integerValue, err := toInteger(v)
		if err != nil {
			return nil, fmt.Errorf("failed to parse property %s: %w", name, err)
		}
		instance[name] = integerValue
	}
	return instance, s.ValidateInstance(instance)
}

func (s *Schema) ValidateInstance(instance map[string]any) error {
	if err := s.validateAdditionalProperties(instance); err != nil {
		return err
	}
	return s.validateTypes(instance)
}

// If additional properties is set to false, this function validates instance only
// contains properties defined in the schema.
func (s *Schema) validateAdditionalProperties(instance map[string]any) error {
	if s.AdditionalProperties != false {
		return nil
	}
	for k := range instance {
		_, ok := s.Properties[k]
		if !ok {
			return fmt.Errorf("property %s is not defined in the schema", k)
		}
	}
	return nil
}

// Validates the types of all input properties values match their types defined in the schema
func (s *Schema) validateTypes(instance map[string]any) error {
	for k, v := range instance {
		fieldInfo, ok := s.Properties[k]
		if !ok {
			continue
		}
		err := validateType(v, fieldInfo.Type)
		if err != nil {
			return fmt.Errorf("incorrect type for property %s: %w", k, err)
		}
	}
	return nil
}
