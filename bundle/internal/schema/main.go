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
	return fmt.Sprintf(`\$\{(%s(\.[a-zA-Z]+([-_]?[a-zA-Z0-9]+)*(\[[0-9]+\])*)+)\}`, s)
}

func addInterpolationPatterns(typ reflect.Type, s jsonschema.Schema) jsonschema.Schema {
	if typ == reflect.TypeOf(config.Root{}) || typ == reflect.TypeOf(variable.Variable{}) {
		return s
	}

	// The variables block in a target override allows for directly specifying
	// the value if it is a primitive type.
	if typ == reflect.TypeOf(variable.TargetVariable{}) {
		return jsonschema.Schema{
			AnyOf: []jsonschema.Schema{s,
				{Type: jsonschema.StringType},
				{Type: jsonschema.BooleanType},
				{Type: jsonschema.IntegerType},
				{Type: jsonschema.NumberType},
			},
		}
	}

	switch s.Type {
	case jsonschema.ArrayType, jsonschema.ObjectType:
		// arrays and objects can have complex variable values specified.
		return jsonschema.Schema{
			AnyOf: []jsonschema.Schema{
				s,
				{
					Type:    jsonschema.StringType,
					Pattern: interpolationPattern("var"),
				}},
		}
	case jsonschema.IntegerType, jsonschema.NumberType, jsonschema.BooleanType:
		// primitives can have variable values, or references like ${bundle.xyz}
		// or ${workspace.xyz}
		return jsonschema.Schema{
			AnyOf: []jsonschema.Schema{
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
