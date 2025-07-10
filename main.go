package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/cmd"
	"github.com/databricks/cli/cmd/pipelines"
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

// If invoked as 'pipelines' (or 'pipelines.exe' on Windows), returns pipelines-specific commands,
// otherwise returns the databricks CLI commands. This is used to allow the same
// binary to be used for both pipelines and databricks CLI commands.
func getCommand(ctx context.Context) *cobra.Command {
	invokedAs := filepath.Base(os.Args[0])
	if strings.HasPrefix(invokedAs, "pipelines") {
		return pipelines.New(ctx)
	}
	return cmd.New(ctx)
}

func main() {
	ctx := context.Background()
	command := getCommand(ctx)
	err := root.Execute(ctx, command)
	if err != nil {
		os.Exit(1)
	}
}
