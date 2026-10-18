package main

import (
	"log"
)

// This package is meant to be run from the root of the CLI repo. It generates typed
// walkers over the bundle configuration: those of the required and enum field
// validators, and the string walker of bundle/config/walk.
func main() {
	err := generateWalkers("bundle/config/validate")
	if err != nil {
		log.Fatalf("Error generating walkers: %v", err)
	}
}
