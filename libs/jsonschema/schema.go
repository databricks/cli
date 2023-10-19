package jsonschema

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"slices"

	"github.com/databricks/cli/internal/build"
	"golang.org/x/mod/semver"
)

// defines schema for a json object
type Schema struct {
	// Type of the object
	Type Type `json:"type,omitempty"`

	// Description of the object. This is rendered as inline documentation in the
	// IDE. This is manually injected here using schema.Docs
	Description string `json:"description,omitempty"`

	// Schemas for the fields of an struct. The keys are the first json tag.
	// The values are the schema for the type of the field
	Properties map[string]*Schema `json:"properties,omitempty"`

	// The schema for all values of an array
	Items *Schema `json:"items,omitempty"`

	// The schema for any properties not mentioned in the Schema.Properties field.
	// this validates maps[string]any in bundle configuration
	// OR
	// A boolean type with value false. Setting false here validates that all
	// properties in the config have been defined in the json schema as properties
	//
	// Its type during runtime will either be *Schema or bool
	AdditionalProperties any `json:"additionalProperties,omitempty"`

	// Required properties for the object. Any fields missing the "omitempty"
	// json tag will be included
	Required []string `json:"required,omitempty"`

	// URI to a json schema
	Reference *string `json:"$ref,omitempty"`

	// Default value for the property / object
	Default any `json:"default,omitempty"`

	// List of valid values for a JSON instance for this schema.
	Enum []any `json:"enum,omitempty"`

	// A pattern is a regular expression the object will be validated against.
	// Can only be used with type "string". The regex syntax supported is available
	// here: https://github.com/google/re2/wiki/Syntax
	Pattern string `json:"pattern,omitempty"`

	// Extension embeds our custom JSON schema extensions.
	Extension
}

type Type string

const (
	InvalidType Type = "invalid"
	BooleanType Type = "boolean"
	StringType  Type = "string"
	NumberType  Type = "number"
	ObjectType  Type = "object"
	ArrayType   Type = "array"
	IntegerType Type = "integer"
)

// Validate property types are all valid JSON schema types.
func (schema *Schema) validateSchemaPropertyTypes() error {
	for _, v := range schema.Properties {
		switch v.Type {
		case NumberType, BooleanType, StringType, IntegerType:
			continue
		case "int", "int32", "int64":
			return fmt.Errorf("type %s is not a recognized json schema type. Please use \"integer\" instead", v.Type)
		case "float", "float32", "float64":
			return fmt.Errorf("type %s is not a recognized json schema type. Please use \"number\" instead", v.Type)
		case "bool":
			return fmt.Errorf("type %s is not a recognized json schema type. Please use \"boolean\" instead", v.Type)
		default:
			return fmt.Errorf("type %s is not a recognized json schema type", v.Type)
		}
	}
	return nil
}

// Validate default property values are consistent with types.
func (schema *Schema) validateSchemaDefaultValueTypes() error {
	for name, property := range schema.Properties {
		if property.Default == nil {
			continue
		}
		if err := validateType(property.Default, property.Type); err != nil {
			return fmt.Errorf("type validation for default value of property %s failed: %w", name, err)
		}
	}
	return nil
}

// Validate enum field values for properties are consistent with types.
func (schema *Schema) validateSchemaEnumValueTypes() error {
	for name, property := range schema.Properties {
		if property.Enum == nil {
			continue
		}
		for i, enum := range property.Enum {
			err := validateType(enum, property.Type)
			if err != nil {
				return fmt.Errorf("type validation for enum at index %v failed for property %s: %w", i, name, err)
			}
		}
	}
	return nil
}

// Validate default value is contained in the list of enums if both are defined.
func (schema *Schema) validateSchemaDefaultValueIsInEnums() error {
	for name, property := range schema.Properties {
		if property.Default == nil || property.Enum == nil {
			continue
		}
		// We expect the default value to be consistent with the list of enum
		// values.
		if !slices.Contains(property.Enum, property.Default) {
			return fmt.Errorf("list of enum values for property %s does not contain default value %v: %v", name, property.Default, property.Enum)
		}
	}
	return nil
}

// Validate usage of "pattern" is consistent.
func (schema *Schema) validateSchemaPattern() error {
	for name, property := range schema.Properties {
		pattern := property.Pattern
		if pattern == "" {
			continue
		}

		// validate property type is string
		if property.Type != StringType {
			return fmt.Errorf("property %q has a non-empty regex pattern %q specified. Patterns are only supported for string properties", name, pattern)
		}

		// validate regex pattern syntax
		r, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid regex pattern %q provided for property %q: %w", pattern, name, err)
		}

		// validate default value against the pattern
		if property.Default != nil && !r.MatchString(property.Default.(string)) {
			return fmt.Errorf("default value %q for property %q does not match specified regex pattern: %q", property.Default, name, pattern)
		}

		// validate enum values against the pattern
		for i, enum := range property.Enum {
			if !r.MatchString(enum.(string)) {
				return fmt.Errorf("enum value %q at index %v for property %q does not match specified regex pattern: %q", enum, i, name, pattern)
			}
		}
	}

	return nil
}

func (schema *Schema) validateSchemaMinimumCliVersion(currentVersion string) func() error {
	return func() error {
		if schema.MinDatabricksCliVersion == "" {
			return nil
		}

		// Ignore this validation rule for local builds.
		if semver.Compare("v"+build.DefaultSemver, currentVersion) == 0 {
			return nil
		}

		// Confirm that MinDatabricksCliVersion is a valid semver.
		if !semver.IsValid(schema.MinDatabricksCliVersion) {
			return fmt.Errorf("invalid minimum CLI version %q specified. Please specify the version in the format v0.0.0", schema.MinDatabricksCliVersion)
		}

		// Confirm that MinDatabricksCliVersion is less than or equal to the current version.
		if semver.Compare(schema.MinDatabricksCliVersion, currentVersion) > 0 {
			return fmt.Errorf("minimum CLI version %q is greater than current CLI version %q. Please upgrade your current Databricks CLI", schema.MinDatabricksCliVersion, currentVersion)
		}
		return nil
	}
}

func (schema *Schema) validate() error {
	for _, fn := range []func() error{
		schema.validateSchemaPropertyTypes,
		schema.validateSchemaDefaultValueTypes,
		schema.validateSchemaEnumValueTypes,
		schema.validateSchemaDefaultValueIsInEnums,
		schema.validateSchemaPattern,
		schema.validateSchemaMinimumCliVersion("v" + build.GetInfo().Version),
	} {
		err := fn()
		if err != nil {
			return err
		}
	}
	return nil
}

func Load(path string) (*Schema, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	schema := &Schema{}
	err = json.Unmarshal(b, schema)
	if err != nil {
		return nil, err
	}

	// Convert the default values of top-level properties to integers.
	// This is required because the default JSON unmarshaler parses numbers
	// as floats when the Golang field it's being loaded to is untyped.
	//
	// NOTE: properties can be recursively defined in a schema, but the current
	// use-cases only uses the first layer of properties so we skip converting
	// any recursive properties.
	for name, property := range schema.Properties {
		if property.Type != IntegerType {
			continue
		}
		if property.Default != nil {
			property.Default, err = toInteger(property.Default)
			if err != nil {
				return nil, fmt.Errorf("failed to parse default value for property %s: %w", name, err)
			}
		}
		for i, enum := range property.Enum {
			property.Enum[i], err = toInteger(enum)
			if err != nil {
				return nil, fmt.Errorf("failed to parse enum value %v at index %v for property %s: %w", enum, i, name, err)
			}
		}
	}

	return schema, schema.validate()
}
