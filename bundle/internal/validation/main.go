package main

import (
	"log"
	"reflect"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/internal/sdkmeta"
	"github.com/databricks/cli/internal/clijson"
)

// This package is meant to be run from the root of the CLI repo.
func main() {
	doc, err := clijson.Parse(".codegen/cli.json")
	if err != nil {
		log.Fatalf("Error reading cli.json: %v", err)
	}
	meta, err := sdkmeta.Build(reflect.TypeFor[config.Root](), doc.Schemas)
	if err != nil {
		log.Fatalf("Error mapping API types: %v", err)
	}

	err = generateRequiredFields("bundle/internal/validation/generated", meta)
	if err != nil {
		log.Fatalf("Error generating required fields: %v", err)
	}

	err = generateEnumFields("bundle/internal/validation/generated", meta)
	if err != nil {
		log.Fatalf("Error generating enum fields: %v", err)
	}
}
