package dsc

import (
	"reflect"

	"github.com/databricks/cli/libs/jsonschema"
)

type PropertyDescriptions map[string]string

// SchemaOptions contains options for schema generation.
type SchemaOptions struct {
	Descriptions PropertyDescriptions
	SchemaDescription string
	ResourceName string
}

func GenerateSchema(t reflect.Type) (any, error) {
	return GenerateSchemaWithOptions(t, SchemaOptions{})
}

func GenerateSchemaWithDescriptions(t reflect.Type, descriptions PropertyDescriptions) (any, error) {
	return GenerateSchemaWithOptions(t, SchemaOptions{Descriptions: descriptions})
}

func GenerateSchemaWithOptions(t reflect.Type, opts SchemaOptions) (any, error) {
	schema, err := jsonschema.FromType(t, nil)
	if err != nil {
		return nil, err
	}
	result := schemaToMapInternal(schema, schema.Definitions, opts.Descriptions)
	result["$schema"] = "https://json-schema.org/draft/2020-12/schema"

	// Add top-level description if provided
	if opts.SchemaDescription != "" {
		result["description"] = opts.SchemaDescription
	}

	// Add _exist property to all schemas
	existDesc := "Indicates whether the resource should exist."
	if opts.ResourceName != "" {
		existDesc = "Indicates whether the " + opts.ResourceName + " should exist."
	}
	props, ok := result["properties"].(map[string]any)
	if !ok {
		props = make(map[string]any)
		result["properties"] = props
	}
	props["_exist"] = map[string]any{
		"type":        "boolean",
		"description": existDesc,
		"default":     true,
	}

	return result, nil
}

func schemaToMapInternal(s jsonschema.Schema, defs map[string]any, descriptions PropertyDescriptions) map[string]any {
	m := make(map[string]any)

	// If this is a reference, try to resolve it inline
	if s.Reference != nil {
		// References are in the form "#/$defs/path/to/type"
		// For simple types like string, we can inline them
		refPath := *s.Reference
		if len(refPath) > 8 && refPath[:8] == "#/$defs/" {
			defPath := refPath[8:]
			if def, ok := resolveRef(defs, defPath); ok {
				if defSchema, ok := def.(jsonschema.Schema); ok {
					return schemaToMapInternal(defSchema, defs, descriptions)
				}
			}
		}
		// If we can't resolve, just add the type as string (common case)
		m["type"] = "string"
		return m
	}

	if s.Type != "" {
		m["type"] = s.Type
	}
	if s.Description != "" {
		m["description"] = s.Description
	}
	if len(s.Required) > 0 {
		m["required"] = s.Required
	}
	if s.Properties != nil {
		props := make(map[string]any)
		for k, v := range s.Properties {
			propSchema := schemaToMapInternal(*v, defs, nil) // Don't pass descriptions recursively
			// Apply custom description if provided
			if descriptions != nil {
				if desc, ok := descriptions[k]; ok {
					propSchema["description"] = desc
				}
			}
			props[k] = propSchema
		}
		m["properties"] = props
	}
	if s.Items != nil {
		m["items"] = schemaToMapInternal(*s.Items, defs, nil)
	}
	if s.AdditionalProperties != nil {
		switch v := s.AdditionalProperties.(type) {
		case bool:
			m["additionalProperties"] = v
		case *jsonschema.Schema:
			m["additionalProperties"] = schemaToMapInternal(*v, defs, nil)
		}
	}
	if s.Default != nil {
		m["default"] = s.Default
	}
	if len(s.Enum) > 0 {
		m["enum"] = s.Enum
	}
	if s.Pattern != "" {
		m["pattern"] = s.Pattern
	}

	return m
}

// resolveRef attempts to resolve a reference path in the definitions map.
func resolveRef(defs map[string]any, path string) (any, bool) {
	if defs == nil {
		return nil, false
	}
	if def, ok := defs[path]; ok {
		return def, true
	}
	return nil, false
}

func GenerateSchemaFromType(v any) (any, error) {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return GenerateSchema(t)
}
