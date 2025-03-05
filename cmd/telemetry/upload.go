package telemetry

import (
	"encoding/json"
	"os"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/spf13/cobra"
)

func newTelemetryUpload() *cobra.Command {
	return &cobra.Command{
		Use: "upload",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := cmd.Context()

			// We exit the process explicitly at the end of Run to avoid the possibility
			// of the worker being spawned recursively. This could otherwise happen if
			// say telemetry was logged as part of this command.
			defer os.Exit(0)

			resp, err := telemetry.Upload(ctx)
			if err != nil {
				log.Error(ctx, err.Error())
				os.Exit(1)
			}

			cmdio.LogString(ctx, "Telemetry logs uploaded successfully")
			cmdio.LogString(ctx, "Response:")
			b, err := json.Marshal(resp)
			if err == nil {
				cmdio.LogString(ctx, string(b))
			}
		},
	}
}
