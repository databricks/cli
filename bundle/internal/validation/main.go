package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/databricks/cli/bundle/internal/validation/generator"
	"github.com/databricks/cli/libs/jsonschema"
)

// This package is meant to be run from the root of the CLI repo.
func main() {
	b, err := os.ReadFile("bundle/schema/jsonschema.json")
	if err != nil {
		log.Fatalf("Error reading schema file: %v", err)
	}

	rootSchema := &jsonschema.Schema{}
	err = json.Unmarshal(b, rootSchema)
	if err != nil {
		log.Fatalf("Error unmarshaling schema: %v", err)
	}

	err = generator.Generate(rootSchema, "bundle/internal/validation/generated")
	if err != nil {
		log.Fatalf("Error generating code: %v", err)
	}
}
