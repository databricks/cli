package template

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"text/template"

	"github.com/databricks/cli/libs/jsonschema"
)

type RenderSchemaInput struct {
	// Path to the JSON file containing input values for rendering.
	InputFile string
}

type RenderSchemaResult struct {
	// The rendered schema content.
	Content string
}

// RenderSchema renders the databricks_template_schema.json file from the template
// using the provided input values. Only fields that can contain template syntax
// are rendered (welcome_message, success_message, property descriptions, and defaults).
//
// Rendering is done in two passes:
// 1. First pass: Render all default values (in order) and add them to the values map
// 2. Second pass: Render descriptions and other fields using the populated values
func RenderSchema(ctx context.Context, reader Reader, input RenderSchemaInput) (*RenderSchemaResult, error) {
	schemaFS, err := reader.SchemaFS(ctx)
	if err != nil {
		return nil, err
	}

	// Load typed schema for ordering
	typedSchema, err := jsonschema.LoadFS(schemaFS, schemaFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to load schema: %w", err)
	}

	// Read the raw schema file content (to preserve all fields in output)
	schemaContent, err := fs.ReadFile(schemaFS, schemaFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	// Parse the schema as JSON
	var schema map[string]any
	if err := json.Unmarshal(schemaContent, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema JSON: %w", err)
	}

	// Load input values from file
	values := make(map[string]any)
	if input.InputFile != "" {
		b, err := os.ReadFile(input.InputFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read input file %s: %w", input.InputFile, err)
		}
		if err := json.Unmarshal(b, &values); err != nil {
			return nil, fmt.Errorf("failed to parse input file %s: %w", input.InputFile, err)
		}
	}

	// missingkey=zero to allow missing variables
	helpers := loadHelpers(ctx)
	baseTmpl := template.New("base").Funcs(helpers).Option("missingkey=zero")

	// renderString renders a template string with the current values.
	renderString := func(s string) (string, error) {
		if s == "" || !strings.Contains(s, "{{") {
			return s, nil
		}
		tmpl, err := baseTmpl.Clone()
		if err != nil {
			return "", err
		}
		tmpl, err = tmpl.Parse(s)
		if err != nil {
			return "", fmt.Errorf("failed to parse template %q: %w", truncate(s, 50), err)
		}
		var buf strings.Builder
		if err := tmpl.Execute(&buf, values); err != nil {
			return "", fmt.Errorf("failed to render template %q: %w", truncate(s, 50), err)
		}
		// Replace <no value> with empty string for missing variables
		result := strings.ReplaceAll(buf.String(), "<no value>", "")
		return result, nil
	}

	// renderField renders a string field in a map if it exists.
	renderField := func(m map[string]any, key, errContext string) error {
		val, ok := m[key].(string)
		if !ok {
			return nil
		}
		rendered, err := renderString(val)
		if err != nil {
			if errContext != "" {
				return fmt.Errorf("%s: %w", errContext, err)
			}
			return err
		}
		m[key] = rendered
		return nil
	}

	// Get properties from raw JSON and use typed schema for ordering
	props, hasProps := schema["properties"].(map[string]any)
	if hasProps {
		orderedProps := typedSchema.OrderedProperties()

		// First pass: Render default values in order and add to values map
		for _, p := range orderedProps {
			propName := p.Name
			prop, ok := props[propName].(map[string]any)
			if !ok {
				continue
			}

			// Skip if value already provided in input
			if _, exists := values[propName]; exists {
				continue
			}

			// Render default value (only if it's a string)
			if def, ok := prop["default"].(string); ok {
				rendered, err := renderString(def)
				if err != nil {
					return nil, fmt.Errorf("property %s default: %w", propName, err)
				}
				prop["default"] = rendered
				// Add to values map for use in subsequent renders
				values[propName] = rendered
			} else if def, ok := prop["default"]; ok {
				// Non-string default (bool, int, etc.) - skip render
				values[propName] = def
			}
		}

		// Second pass: Render descriptions and other fields
		for propName, propVal := range props {
			prop, ok := propVal.(map[string]any)
			if !ok {
				continue
			}
			if err := renderField(prop, "description", "property "+propName); err != nil {
				return nil, err
			}
			if err := renderField(prop, "pattern_match_failure_message", "property "+propName+" pattern_match_failure_message"); err != nil {
				return nil, err
			}
		}
	}

	// Render welcome_message and success_message (after properties so we have all values)
	if err := renderField(schema, "welcome_message", ""); err != nil {
		return nil, err
	}
	if err := renderField(schema, "success_message", ""); err != nil {
		return nil, err
	}

	result, err := json.MarshalIndent(schema, "", "    ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize rendered schema: %w", err)
	}

	return &RenderSchemaResult{
		Content: string(result),
	}, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
