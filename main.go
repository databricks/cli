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

func main() {
	ctx := context.Background()

	// Branch command based on program name: 'dlt' runs DLT-specific commands,
	// while 'databricks' runs the main CLI commands
	invokedAs := filepath.Base(os.Args[0])
	var command *cobra.Command
	if invokedAs == "dlt" || (runtime.GOOS == "windows" && invokedAs == "dlt.exe") {
		command = dlt.New()
	} else {
		command = cmd.New(ctx)
	}
	err := root.Execute(ctx, command)
	if err != nil {
		os.Exit(1)
	}
}
