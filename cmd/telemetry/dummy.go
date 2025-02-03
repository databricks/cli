package telemetry

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
)

func newDummyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dummy",
		Short:   "log dummy telemetry events",
		Long:    "Fire a test telemetry event against the configured Databricks workspace.",
		Hidden:  true,
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
