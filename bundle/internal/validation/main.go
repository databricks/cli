package main

import (
	"log"
)

// This package is meant to be run from the root of the CLI repo.
func main() {
	err := generateRequiredFields("bundle/internal/validation/generated")
	if err != nil {
		log.Fatalf("Error generating code: %v", err)
	}
}
