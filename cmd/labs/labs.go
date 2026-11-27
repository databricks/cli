package labs

import (
	"context"

	"github.com/databricks/cli/cmd/labs/project"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/log"
	"github.com/spf13/cobra"
)

// Spec declares the labs command.
var Spec = root.CommandSpec{
	Use:   "labs",
	Short: "Manage Databricks Labs installations",
	Build: buildCommand,
}

func buildCommand(ctx context.Context, cmd *cobra.Command) {
	cmd.Long = `Manage experimental Databricks Labs apps`

	cmd.AddGroup(&cobra.Group{
		ID:    "labs",
		Title: "Installed Databricks Labs",
	})

	cmd.AddCommand(
		newListCommand(),
		newInstallCommand(),
		newUpgradeCommand(),
		newInstalledCommand(),
		newClearCacheCommand(),
		newUninstallCommand(),
		newShowCommand(),
	)
	all, err := project.Installed(ctx)
	if err != nil {
		log.Errorf(ctx, "Cannot retrieve installed labs: %s", err)
		return
	}
	for _, v := range all {
		v.Register(cmd)
	}
}
