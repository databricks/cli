package main

import (
	"context"
	"log"

	"github.com/databricks/bricks/bundle/internal/tf-codegen/codegen"
	"github.com/databricks/bricks/bundle/internal/tf-codegen/schema"
)

func main() {
	ctx := context.Background()

	schema, err := schema.Load(ctx)
	if err != nil {
		log.Fatal(err)
	}

	err = codegen.Run(ctx, schema, "../tf")
	if err != nil {
		log.Fatal(err)
	}
}
