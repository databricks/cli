package jsonschema

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
)

// Load a JSON document and validate it against the JSON schema. Instance here
// refers to a JSON document. see: https://json-schema.org/draft/2020-12/json-schema-core.html#name-instance
func (s *Schema) LoadInstance(path string) (map[string]any, error) {
	instance := make(map[string]any)
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
		if !ok || propertySchema.Type != IntegerType {
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

// Validate an instance against the schema
func (s *Schema) ValidateInstance(instance map[string]any) error {
	validations := []func(map[string]any) error{
		s.validateAdditionalProperties,
		s.validateEnum,
		s.validateRequired,
		s.validateTypes,
		s.validatePattern,
		s.validateConst,
		s.validateAnyOf,
	}

	for _, fn := range validations {
		err := fn(instance)
		if err != nil {
			return err
		}
	}
	return nil
}

// If additional properties is set to false, this function validates instance only
// contains properties defined in the schema.
func (s *Schema) validateAdditionalProperties(instance map[string]any) error {
	// Note: AdditionalProperties has the type any.
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

// This function validates that all require properties in the schema have values
// in the instance.
func (s *Schema) validateRequired(instance map[string]any) error {
	for _, name := range s.Required {
		if _, ok := instance[name]; !ok {
			return fmt.Errorf("no value provided for required property %s", name)
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

func (s *Schema) validateEnum(instance map[string]any) error {
	for k, v := range instance {
		fieldInfo, ok := s.Properties[k]
		if !ok || fieldInfo.Enum == nil {
			continue
		}
		if !slices.Contains(fieldInfo.Enum, v) {
			return fmt.Errorf("expected value of property %s to be one of %v. Found: %v", k, fieldInfo.Enum, v)
		}
	}
	return nil
}

func (s *Schema) validatePattern(instance map[string]any) error {
	for k, v := range instance {
		fieldInfo, ok := s.Properties[k]
		if !ok {
			continue
		}
		err := validatePatternMatch(k, v, fieldInfo)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Schema) validateConst(instance map[string]any) error {
	for k, v := range instance {
		fieldInfo, ok := s.Properties[k]
		if !ok || fieldInfo.Const == nil {
			continue
		}
		if v != fieldInfo.Const {
			return fmt.Errorf("expected value of property %s to be %v. Found: %v", k, fieldInfo.Const, v)
		}
	}
	return nil
}

// Validates that the instance matches at least one of the schemas in anyOf
// For more information, see https://json-schema.org/understanding-json-schema/reference/combining#anyof.
func (s *Schema) validateAnyOf(instance map[string]any) error {
	if s.AnyOf == nil {
		return nil
	}

	// According to the JSON schema RFC, anyOf must contain at least one schema.
	// https://json-schema.org/draft/2020-12/json-schema-core
	if len(s.AnyOf) == 0 {
		return errors.New("anyOf must contain at least one schema")
	}

	for _, anyOf := range s.AnyOf {
		err := anyOf.ValidateInstance(instance)
		if err == nil {
			return nil
		}
	}
	return errors.New("instance does not match any of the schemas in anyOf")
}
