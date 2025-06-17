package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

	"github.com/databricks/cli/cmd"
	"github.com/databricks/cli/cmd/dlt"
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

// If invoked as 'dlt' (or 'dlt.exe' on Windows), returns DLT-specific commands,
// otherwise returns the databricks CLI commands. This is used to allow the same
// binary to be used for both DLT and databricks CLI commands.
func getCommand(ctx context.Context) *cobra.Command {
	invokedAs := filepath.Base(os.Args[0])
	if invokedAs == "dlt" || (runtime.GOOS == "windows" && invokedAs == "dlt.exe") {
		return dlt.New()
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
