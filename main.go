package main

import (
	"context"
	"os"

	"github.com/databricks/cli/cmd"
	"github.com/databricks/cli/cmd/root"

	// Registers a disk-cached HostMetadataResolver factory on the SDK so every
	// *config.Config the CLI constructs reuses the cached /.well-known lookup.
	_ "github.com/databricks/cli/libs/hostmetadata"
)

func main() {
	ctx := context.Background()
	err := root.Execute(ctx, cmd.New(ctx))
	if err != nil {
		os.Exit(1)
	}
}
