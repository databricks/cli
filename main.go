package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/databricks/cli/cmd"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/databricks-sdk-go/logger"
)

func main() {
	ctx := context.Background()

	// A new process is spawned for uploading telemetry data. We handle this separately
	// from the rest of the CLI commands.
	// This is done because [root.Execute] spawns a new process to run the
	// "telemetry upload" command if there are logs to be uploaded. Having this outside
	// of [root.Execute] ensures that the telemetry upload process is not spawned
	// infinitely in a recursive manner.
	if len(os.Args) == 3 && os.Args[1] == "telemetry" && os.Args[2] == "upload" {
		// Suppress non-error logs from the SDK.
		logger.DefaultLogger = &logger.SimpleLogger{
			Level: logger.LevelError,
		}

		resp, err := telemetry.Upload(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			os.Exit(1)
		}
		fmt.Printf("Telemetry logs uploaded successfully\n")
		fmt.Println("Response:")
		b, err := json.Marshal(resp)
		if err == nil {
			fmt.Println(string(b))
		}
		os.Exit(0)
	}

	err := root.Execute(ctx, cmd.New(ctx))
	if err != nil {
		os.Exit(1)
	}
}
