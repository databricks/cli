package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/libs/jsonschema"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

func interpolationPattern(s string) string {
	return fmt.Sprintf(`\$\{(%s(\.[a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\[[0-9]+\])*)+)\}`, s)
}

func addInterpolationPatterns(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
	if typ == reflect.TypeOf(config.Root{}) || typ == reflect.TypeOf(variable.Variable{}) {
		return s
	}

	// The variables block in a target override allows for directly specifying
	// the value of the variable.
	if typ == reflect.TypeOf(variable.TargetVariable{}) {
		return jsonschema.Schema{
			AnyOf: []jsonschema.Schema{
				// We keep the original schema so that autocomplete suggestions
				// continue to work.
				s,
				// All values are valid for a variable value, be it primitive types
				// like string/bool or complex ones like objects/arrays. Thus we override
				// the schema to allow all valid JSON values.
				{},
			},
		}
	}

	// Allows using variables in enum fields
	if s.Type == jsonschema.StringType && s.Enum != nil {
		return jsonschema.Schema{
			OneOf: []jsonschema.Schema{
				s,
				{
					Type:    jsonschema.StringType,
					Pattern: interpolationPattern("var"),
				},
			},
		}
	}

	switch s.Type {
	case jsonschema.ArrayType, jsonschema.ObjectType:
		// arrays and objects can have complex variable values specified.
		return jsonschema.Schema{
			// OneOf is used because we don't expect more than 1 match and schema-based auto-complete works better with OneOf
			OneOf: []jsonschema.Schema{
				s,
				{
					Type:    jsonschema.StringType,
					Pattern: interpolationPattern("var"),
				},
			},
		}
	case jsonschema.IntegerType, jsonschema.NumberType, jsonschema.BooleanType:
		// primitives can have variable values, or references like ${bundle.xyz}
		// or ${workspace.xyz}
		return jsonschema.Schema{
			OneOf: []jsonschema.Schema{
				s,
				{Type: jsonschema.StringType, Pattern: interpolationPattern("resources")},
				{Type: jsonschema.StringType, Pattern: interpolationPattern("bundle")},
				{Type: jsonschema.StringType, Pattern: interpolationPattern("workspace")},
				{Type: jsonschema.StringType, Pattern: interpolationPattern("artifacts")},
				{Type: jsonschema.StringType, Pattern: interpolationPattern("var")},
			},
		}
	default:
		return s
	}
}

func removeJobsFields(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
	switch typ {
	case reflect.TypeOf(resources.Job{}):
		// This field has been deprecated in jobs API v2.1 and is always set to
		// "MULTI_TASK" in the backend. We should not expose it to the user.
		delete(s.Properties, "format")

		// These fields are only meant to be set by the DABs client (ie the CLI)
		// and thus should not be exposed to the user. These are used to annotate
		// jobs that were created by DABs.
		delete(s.Properties, "deployment")
		delete(s.Properties, "edit_mode")

	case reflect.TypeOf(jobs.GitSource{}):
		// These fields are readonly and are not meant to be set by the user.
		delete(s.Properties, "job_source")
		delete(s.Properties, "git_snapshot")

	default:
		// Do nothing
	}

	return s
}

func removePipelineFields(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
	switch typ {
	case reflect.TypeOf(resources.Pipeline{}):
		// Even though DABs supports this field, TF provider does not. Thus, we
		// should not expose it to the user.
		delete(s.Properties, "dry_run")

		// These fields are only meant to be set by the DABs client (ie the CLI)
		// and thus should not be exposed to the user. These are used to annotate
		// pipelines that were created by DABs.
		delete(s.Properties, "deployment")
	default:
		// Do nothing
	}

	return s
}

// While volume_type is required in the volume create API, DABs automatically sets
// it's value to "MANAGED" if it's not provided. Thus, we make it optional
// in the bundle schema.
func makeVolumeTypeOptional(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
	if typ != reflect.TypeOf(resources.Volume{}) {
		return s
	}

	var res []string
	for _, r := range s.Required {
		if r != "volume_type" {
			res = append(res, r)
		}
	}
	s.Required = res
	return s
}

func removeOutputOnlyFields(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
	// Only process object types with properties
	if s.Type != jsonschema.ObjectType || s.Properties == nil {
		return s
	}

	// Collect property names to remove
	var toRemove []string
	for name, prop := range s.Properties {
		// Check if this property is marked as output-only via FieldBehaviors
		if prop.FieldBehaviors != nil {
			for _, behavior := range prop.FieldBehaviors {
				if behavior == "OUTPUT_ONLY" {
					toRemove = append(toRemove, name)
					break
				}
			}
		}
	}

	// Remove output-only properties
	for _, name := range toRemove {
		delete(s.Properties, name)

		// Also remove from required list if present
		var newRequired []string
		for _, r := range s.Required {
			if r != name {
				newRequired = append(newRequired, r)
			}
		}
		s.Required = newRequired
	}

	return s
}

func enhanceDescriptions(_ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
	// Add enum values suffix for string types with enums
	if s.Type == jsonschema.StringType && len(s.Enum) > 0 {
		// Skip if description already contains 2+ enum values. This is a rubric / best effort to avoid
		// duplicate information if the description already contains the enum values.
		enumCount := 0
		descLower := strings.ToLower(s.Description)
		for _, v := range s.Enum {
			if str, ok := v.(string); ok && strings.Contains(descLower, strings.ToLower(str)) {
				enumCount++
			}
		}

		if enumCount < 2 {
			suffix := formatEnumSuffix(s.Enum)
			if s.Description == "" {
				s.Description = suffix
			} else {
				s.Description = strings.TrimSuffix(s.Description, ".") + ". " + suffix
			}
		}
	}

	// Add "Required." prefix for object properties that are required
	if s.Type == jsonschema.ObjectType && s.Properties != nil {
		for name, prop := range s.Properties {
			// If the description contains "required" (case insensitive) we skip the prefix.
			// This is a rubric to prevent malformed descriptions like "Required. Required, my field description".
			// Adding the prefix is best effort.
			if slices.Contains(s.Required, name) && !strings.Contains(strings.ToLower(prop.Description), "required") {
				if prop.Description == "" {
					prop.Description = "Required."
				} else {
					prop.Description = "Required. " + prop.Description
				}
			}
		}
	}

	return s
}

func formatEnumSuffix(enum []any) string {
	strs := make([]string, 0, len(enum))
	for _, v := range enum {
		if str, ok := v.(string); ok {
			strs = append(strs, "`"+str+"`")
		}
	}

	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return "Valid values are: " + strs[0] + "."
	}
	if len(strs) == 2 {
		return "Valid values are: " + strs[0] + " and " + strs[1] + "."
	}
	return "Valid values are: " + strings.Join(strs[:len(strs)-1], ", ") + ", and " + strs[len(strs)-1] + "."
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run main.go <work-dir> <output-file>")
		os.Exit(1)
	}

	// Directory with annotation files
	workdir := os.Args[1]
	// Output file, where the generated JSON schema will be written to.
	outputFile := os.Args[2]

	generateSchema(workdir, outputFile)
}

func generateSchema(workdir, outputFile string) {
	annotationsPath := filepath.Join(workdir, "annotations.yml")
	annotationsOpenApiPath := filepath.Join(workdir, "annotations_openapi.yml")
	annotationsOpenApiOverridesPath := filepath.Join(workdir, "annotations_openapi_overrides.yml")

	// Input file, the databricks openapi spec.
	inputFile := os.Getenv("DATABRICKS_OPENAPI_SPEC")
	if inputFile != "" {
		p, err := newParser(inputFile)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Writing OpenAPI annotations to %s\n", annotationsOpenApiPath)
		err = p.extractAnnotations(reflect.TypeOf(config.Root{}), annotationsOpenApiPath, annotationsOpenApiOverridesPath)
		if err != nil {
			log.Fatal(err)
		}
	}

	a, err := newAnnotationHandler([]string{annotationsOpenApiPath, annotationsOpenApiOverridesPath, annotationsPath})
	if err != nil {
		log.Fatal(err)
	}

	// Generate the JSON schema from the bundle Go struct.
	s, err := jsonschema.FromType(reflect.TypeOf(config.Root{}), []func(reflect.Type, jsonschema.Schema) jsonschema.Schema{
		removeJobsFields,
		removePipelineFields,
		makeVolumeTypeOptional,
		a.addAnnotations,
		removeOutputOnlyFields,
		enhanceDescriptions,
		addInterpolationPatterns,
	})

	// AdditionalProperties is set to an empty schema to allow non-typed keys used as yaml-anchors
	// Example:
	// some_anchor: &some_anchor
	//   file_path: /some/path/
	// workspace:
	//   <<: *some_anchor
	s.AdditionalProperties = jsonschema.Schema{}

	if err != nil {
		log.Fatal(err)
	}

	// Overwrite the input annotation file, adding missing annotations
	err = a.syncWithMissingAnnotations(annotationsPath)
	if err != nil {
		log.Fatal(err)
	}

	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	// Write the schema descriptions to the output file.
	err = os.WriteFile(outputFile, b, 0o644)
	if err != nil {
		log.Fatal(err)
	}
}
