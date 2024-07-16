package labs

import (
	"github.com/databricks/cli/cmd/labs/project"
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func newUpgradeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade NAME",
		Args:  root.ExactArgs(1),
		Short: "Upgrades project",
		RunE: func(cmd *cobra.Command, args []string) error {
			inst, err := project.NewUpgrader(cmd, args[0])
			if err != nil {
				return err
			}
			return inst.Upgrade(cmd.Context())
		},
	}
}
