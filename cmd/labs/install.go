package labs

import (
	"github.com/databricks/cli/cmd/labs/project"
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func newInstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "install NAME",
		Args:  root.ExactArgs(1),
		Short: "Installs project",
		RunE: func(cmd *cobra.Command, args []string) error {
			inst, err := project.NewInstaller(cmd, args[0])
			if err != nil {
				return err
			}
			return inst.Install(cmd.Context())
		},
	}
}
