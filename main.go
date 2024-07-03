package main

import (
	"context"
	"os"

	"github.com/databricks/cli/cmd"
	"github.com/databricks/cli/cmd/root"
)

func main() {
	ctx := context.Background()
	code := root.Execute(ctx, cmd.New(ctx))

	os.Exit(code)
}
