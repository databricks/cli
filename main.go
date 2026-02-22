package main

import (
	"context"
	"os"

	"github.com/databricks/cli/cmd"
	"github.com/databricks/cli/cmd/root"
)

func main() {
	ctx := context.Background()
	err := root.Execute(ctx, cmd.New(ctx))
	if err != nil {
		os.Exit(1)
	}
}
