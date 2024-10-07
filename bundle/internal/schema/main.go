package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"

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
		removeJobsFields,
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
