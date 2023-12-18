package labs

import (
	"context"

	"github.com/databricks/cli/cmd/labs/project"
	"github.com/databricks/cli/libs/log"
	"github.com/spf13/cobra"
)

func New(ctx context.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "labs",
		Short: "Manage Databricks Labs installations",
		Long:  `Manage experimental Databricks Labs apps`,
	}

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
		return cmd
	}
	for _, v := range all {
		v.Register(cmd)
	}
	return cmd
}
