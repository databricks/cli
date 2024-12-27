package labs

import (
	"github.com/databricks/cli/cmd/labs/project"
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func newInstallCommand() *cobra.Command {
	cmd := &cobra.Command{}
	var offlineInstall bool

	cmd.Flags().BoolVar(&offlineInstall, "offline", offlineInstall, `If installing in offline mode, set this flag to true.`)

	cmd.Use = "install NAME"
	cmd.Args = root.ExactArgs(1)
	cmd.Short = "Installs project"
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		inst, err := project.NewInstaller(cmd, args[0], offlineInstall)
		if err != nil {
			return err
		}
		return inst.Install(cmd.Context())
	}
	return cmd
}
