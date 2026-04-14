package main

import (
	"context"
	"log"

	"github.com/databricks/cli/bundle/internal/tf/codegen/generator"
	"github.com/databricks/cli/bundle/internal/tf/codegen/schema"
)

func main() {
	ctx := context.Background()

	s, err := schema.Load(ctx)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("fetching provider checksums for v%s", schema.ProviderVersion)
	checksums, err := schema.FetchProviderChecksums(schema.ProviderVersion)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("  linux_amd64: %s", checksums.LinuxAmd64)
	log.Printf("  linux_arm64: %s", checksums.LinuxArm64)

	err = generator.Run(ctx, s, checksums, "../schema")
	if err != nil {
		log.Fatal(err)
	}
}
