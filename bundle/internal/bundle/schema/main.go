package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/databricks/cli/bundle/schema"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: go run main.go <output-file>")
		os.Exit(1)
	}

	// Output file, to write the generated schema descriptions to.
	outputFile := os.Args[1]

	// Input file, the databricks openapi spec.
	inputFile := os.Getenv("DATABRICKS_OPENAPI_SPEC")
	if inputFile == "" {
		log.Fatal("DATABRICKS_OPENAPI_SPEC environment variable not set")
	}

	// Generate the schema descriptions.
	docs, err := schema.BundleDocs(inputFile)
	if err != nil {
		log.Fatal(err)
	}
	result, err := json.MarshalIndent(docs, "", "  ")
	if err != nil {
		log.Fatal(err)
	}

	// Write the schema descriptions to the output file.
	err = os.WriteFile(outputFile, result, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
