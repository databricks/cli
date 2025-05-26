package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/databricks/cli/bundle/internal/validation/generator"
	"github.com/databricks/cli/libs/jsonschema"
)

func main() {
	// The context is not used by jsonschema.Load or the current generator.Generate,
	// but it's good practice to have it if future versions or dependencies need it.
	ctx := context.Background()

	// Define the input schema path and output directory
	// Input is relative to the workspace root (e.g. /Users/shreyas.goenka/repos/cli)
	schemaFilePath := "bundle/schema/jsonschema.json"
	outputDir := "bundle/internal/validation/generated"

	log.Printf("Loading schema from: %s", schemaFilePath)
	b, err := os.ReadFile(schemaFilePath)
	if err != nil {
		log.Fatalf("Error reading schema file: %v", err)
	}

	rootSchema := &jsonschema.Schema{}
	err = json.Unmarshal(b, rootSchema)
	if err != nil {
		log.Fatalf("Error unmarshalling schema: %v", err)
	}

	if rootSchema == nil {
		log.Fatalf("Loaded root schema from %s is nil", schemaFilePath)
	}

	log.Printf("Generating required fields map...")
	// Pass the root schema to the generator.
	// The generator will handle resolving internal $defs.
	err = generator.Generate(ctx, rootSchema, outputDir)
	if err != nil {
		log.Fatalf("Error generating code: %v", err)
	}

	log.Printf("Code generation complete. Output in %s/required_fields.go", outputDir)
}
