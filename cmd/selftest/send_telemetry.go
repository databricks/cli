package selftest

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
)

func newSendTelemetry() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "send-telemetry",
		Short:   "log some test telemetry events",
		PreRunE: root.MustWorkspaceClient,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		for _, v := range []string{"VALUE1", "VALUE2"} {
			telemetry.Log(cmd.Context(), protos.DatabricksCliLog{
				CliTestEvent: &protos.CliTestEvent{
					Name: protos.DummyCliEnum(v),
				},
			})
		}
		return nil
	}

	return cmd
}
