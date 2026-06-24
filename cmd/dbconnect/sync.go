package dbconnect

import (
	"github.com/databricks/cli/cmd/root"
	libsdbconnect "github.com/databricks/cli/libs/dbconnect"
	"github.com/spf13/cobra"
)

func newSyncCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Merge managed dependencies into an existing pyproject.toml and re-provision",
	}
	cmd.PreRunE = root.MustWorkspaceClient
	addTargetFlags(cmd)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runPipeline(cmd, libsdbconnect.ModeSync)
	}
	return cmd
}
