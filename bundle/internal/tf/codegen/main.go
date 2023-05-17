package main

import (
	"context"
	"log"

	"github.com/databricks/cli/bundle/internal/tf/codegen/generator"
	"github.com/databricks/cli/bundle/internal/tf/codegen/schema"
)

func main() {
	ctx := context.Background()

	schema, err := schema.Load(ctx)
	if err != nil {
		log.Fatal(err)
	}

	err = generator.Run(ctx, schema, "../schema")
	if err != nil {
		log.Fatal(err)
	}
}
