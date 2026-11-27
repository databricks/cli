package main

import (
	"log"
)

// This package is meant to be run from the root of the CLI repo. It generates typed
// walkers over the bundle configuration: the defaults walker and the string walker of
// bundle/config/walk. The required and enum validators walk by the cached struct
// information of libs/structs/structinfo and need no generated code.
func main() {
	err := generateWalkers()
	if err != nil {
		log.Fatalf("Error generating walkers: %v", err)
	}
}
