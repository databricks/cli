package labs

import (
	"fmt"

	"github.com/databricks/cli/cmd/labs/project"
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func newUninstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall NAME",
		Args:  root.ExactArgs(1),
		Short: "Uninstalls project",
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			var names []string
			installed, _ := project.Installed(cmd.Context())
			for _, v := range installed {
				names = append(names, v.Name)
			}
			return names, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			installed, err := project.Installed(ctx)
			if err != nil {
				return err
			}
			name := args[0]
			for _, prj := range installed {
				if prj.Name != name {
					continue
				}
				return prj.Uninstall(cmd)
			}
			return fmt.Errorf("not found: %s", name)
		},
	}
}
