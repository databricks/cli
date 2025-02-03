package main

import (
	"context"
	"fmt"
	"os"

	"github.com/databricks/cli/cmd"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/telemetry"
)

func main() {
	ctx := context.Background()

	// Uploading telemetry data spawns a new process. We handle this separately
	// from the rest of the CLI commands.
	// This is done because [root.Execute] spawns a new process to run the
	// "telemetry upload" command if there are logs to be uploaded. Having this outside
	// of [root.Execute] ensures that the telemetry upload process is not spawned
	// infinitely in a recursive manner.
	if len(os.Args) == 3 && os.Args[1] == "telemetry" && os.Args[2] == "upload" {
		var err error

		// If the environment variable is set, redirect stdout to the file.
		// This is useful for testing.
		if v := os.Getenv(telemetry.UploadLogsFileEnvVar); v != "" {
			os.Stdout, _ = os.OpenFile(v, os.O_CREATE|os.O_WRONLY, 0o644)
			os.Stderr = os.Stdout
		}

		err = telemetry.Upload()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	err := root.Execute(ctx, cmd.New(ctx))
	if err != nil {
		os.Exit(1)
	}
}
