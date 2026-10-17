package main

import (
	"log"
)

// This package is meant to be run from the root of the CLI repo. It generates the
// walkers of the required and enum field validators from the bundle configuration types.
func main() {
	err := generateWalkers("bundle/config/validate")
	if err != nil {
		log.Fatalf("Error generating walkers: %v", err)
	}
}
