package jsonschema

import (
	"encoding/json"
	"fmt"
	"os"
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

func (schema *Schema) validate() error {
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
	return schema, schema.validate()
}
