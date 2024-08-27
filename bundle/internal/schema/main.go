package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/libs/jsonschema"
)

func interpolationPattern(s string) string {
	return fmt.Sprintf(`\$\{(%s(\.[a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\[[0-9]+\])*)*(\[[0-9]+\])*)\}`, s)
}

func addInterpolationPatterns(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
	if typ == reflect.TypeOf(config.Root{}) || typ == reflect.TypeOf(variable.Variable{}) {
		return s
	}

	switch s.Type {
	case jsonschema.ArrayType, jsonschema.ObjectType:
		// arrays and objects can have complex variable values specified.
		return jsonschema.Schema{
			AnyOf: []jsonschema.Schema{s, {
				Type: jsonschema.StringType,
				// TODO: Are multi-level complex variable references supported?
				Pattern: interpolationPattern("var"),
			}},
		}
	case jsonschema.IntegerType, jsonschema.NumberType, jsonschema.BooleanType:
		// primitives can have variable values, or references like ${bundle.xyz}
		// or ${workspace.xyz}
		// TODO: Followup, do not allow references like ${} in the schema unless
		// they are of the permitted patterns?
		return jsonschema.Schema{
			AnyOf: []jsonschema.Schema{s,
				// TODO: Is it only resource IDs or is it resources in general?
				{Type: jsonschema.StringType, Pattern: interpolationPattern("resources")},
				{Type: jsonschema.StringType, Pattern: interpolationPattern("bundle")},
				{Type: jsonschema.StringType, Pattern: interpolationPattern("workspace")},
				{Type: jsonschema.StringType, Pattern: interpolationPattern("var")},
			},
		}
	default:
		return s
	}
}

// TODO: Add a couple of end to end tests that the bundle schema generated is
// correct.
// TODO: Call out in the PR description that recursive types like "for_each_task"
// are now supported. Manually test for_each_task.
// TODO: The bundle_descriptions.json file contains a bunch of custom descriptions
// as well. Make sure to pull those in.
// TODO: Add unit tests for all permutations of structs, maps and slices for the FromType
// method.
// TODO: Note the minor regression of losing the bundle descriptions.
// TODO: Ensure descriptions work for target resources section.

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <output-file>")
		os.Exit(1)
	}

	// Output file, where the generated JSON schema will be written to.
	outputFile := os.Args[1]

	// Input file, the databricks openapi spec.
	inputFile := os.Getenv("DATABRICKS_OPENAPI_SPEC")
	if inputFile == "" {
		log.Fatal("DATABRICKS_OPENAPI_SPEC environment variable not set")
	}

	p, err := newParser(inputFile)
	if err != nil {
		log.Fatal(err)
	}

	// Generate the JSON schema from the bundle Go struct.
	s, err := jsonschema.FromType(reflect.TypeOf(config.Root{}), []func(reflect.Type, jsonschema.Schema) jsonschema.Schema{
		p.addDescriptions,
		p.addEnums,
		addInterpolationPatterns,
	})
	if err != nil {
		log.Fatal(err)
	}

	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	// Write the schema descriptions to the output file.
	err = os.WriteFile(outputFile, b, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
