package libraries

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/libraries"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "libraries",
	Short: `The Libraries API allows you to install and uninstall libraries and get the status of libraries on a cluster.`,
}

func init() {
	Cmd.AddCommand(allClusterStatusesCmd)

}

var allClusterStatusesCmd = &cobra.Command{
	Use:   "all-cluster-statuses",
	Short: `Get all statuses.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Libraries.AllClusterStatuses(ctx)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var clusterStatusReq libraries.ClusterStatus

func init() {
	Cmd.AddCommand(clusterStatusCmd)
	// TODO: short flags

	clusterStatusCmd.Flags().StringVar(&clusterStatusReq.ClusterId, "cluster-id", "", `Unique identifier of the cluster whose status should be retrieved.`)

}

var clusterStatusCmd = &cobra.Command{
	Use:   "cluster-status",
	Short: `Get status.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.Libraries.ClusterStatus(ctx, clusterStatusReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var installReq libraries.InstallLibraries

func init() {
	Cmd.AddCommand(installCmd)
	// TODO: short flags

	installCmd.Flags().StringVar(&installReq.ClusterId, "cluster-id", "", `Unique identifier for the cluster on which to install these libraries.`)
	// TODO: complex arg: libraries

}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: `Add a library.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Libraries.Install(ctx, installReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var uninstallReq libraries.UninstallLibraries

func init() {
	Cmd.AddCommand(uninstallCmd)
	// TODO: short flags

	uninstallCmd.Flags().StringVar(&uninstallReq.ClusterId, "cluster-id", "", `Unique identifier for the cluster on which to uninstall these libraries.`)
	// TODO: complex arg: libraries

}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: `Uninstall libraries.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Libraries.Uninstall(ctx, uninstallReq)
		if err != nil {
			return err
		}

		return nil
	},
}
