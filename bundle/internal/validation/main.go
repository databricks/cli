package main

import (
	"log"

	"github.com/databricks/cli/bundle/internal/validation/required"
)

// This package is meant to be run from the root of the CLI repo.
func main() {
	err := required.Generate("bundle/internal/validation/generated")
	if err != nil {
		log.Fatalf("Error generating code: %v", err)
	}
}
