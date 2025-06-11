package main

import (
	"context"
	"os"
	"path/filepath"

	"github.com/databricks/cli/cmd"
	"github.com/databricks/cli/cmd/dlt"
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func main() {
	ctx := context.Background()

	invokedAs := filepath.Base(os.Args[0])
	var command *cobra.Command
	if invokedAs == "dlt" {
		command = dlt.New()
	} else {
		command = cmd.New(ctx)
	}
	err := root.Execute(ctx, command)
	if err != nil {
		os.Exit(1)
	}
}
