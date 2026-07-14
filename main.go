package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/databricks/cli/cmd"
	"github.com/databricks/cli/cmd/root"

	// Registers a disk-cached HostMetadataResolver factory on the SDK so every
	// *config.Config the CLI constructs reuses the cached /.well-known lookup.
	_ "github.com/databricks/cli/libs/hostmetadata"
)

func main() {
	// Configure DATABRICKS_CLI_PATH only if our caller intends to use this specific version of this binary.
	// Otherwise, if it is equal to its basename, processes can find it in $PATH.
	// This runs in main rather than in a package init so that importing CLI
	// packages (e.g. from test binaries or generators) does not mutate the
	// process environment.
	arg0 := os.Args[0]
	if arg0 != filepath.Base(arg0) {
		os.Setenv("DATABRICKS_CLI_PATH", arg0)
	}

	ctx := context.Background()
	err := root.Execute(ctx, cmd.New(ctx))
	if err != nil {
		os.Exit(1)
	}
}
