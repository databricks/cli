package libraries

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/libraries"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "libraries",
	Short: `The Libraries API allows you to install and uninstall libraries and get the status of libraries on a cluster.`,
	Long: `The Libraries API allows you to install and uninstall libraries and get the
  status of libraries on a cluster.
  
  To make third-party or custom code available to notebooks and jobs running on
  your clusters, you can install a library. Libraries can be written in Python,
  Java, Scala, and R. You can upload Java, Scala, and Python libraries and point
  to external packages in PyPI, Maven, and CRAN repositories.
  
  Cluster libraries can be used by all notebooks running on a cluster. You can
  install a cluster library directly from a public repository such as PyPI or
  Maven, using a previously installed workspace library, or using an init
  script.
  
  When you install a library on a cluster, a notebook already attached to that
  cluster will not immediately see the new library. You must first detach and
  then reattach the notebook to the cluster.
  
  When you uninstall a library from a cluster, the library is removed only when
  you restart the cluster. Until you restart the cluster, the status of the
  uninstalled library appears as Uninstall pending restart.`,
}

func init() {
	Cmd.AddCommand(allClusterStatusesCmd)

}

var allClusterStatusesCmd = &cobra.Command{
	Use:   "all-cluster-statuses",
	Short: `Get all statuses.`,
	Long: `Get all statuses.
  
  Get the status of all libraries on all clusters. A status will be available
  for all libraries installed on this cluster via the API or the libraries UI as
  well as libraries set to be installed on all clusters via the libraries UI.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
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
	Long: `Get status.
  
  Get the status of libraries on a cluster. A status will be available for all
  libraries installed on this cluster via the API or the libraries UI as well as
  libraries set to be installed on all clusters via the libraries UI. The order
  of returned libraries will be as follows.
  
  1. Libraries set to be installed on this cluster will be returned first.
  Within this group, the final order will be order in which the libraries were
  added to the cluster.
  
  2. Libraries set to be installed on all clusters are returned next. Within
  this group there is no order guarantee.
  
  3. Libraries that were previously requested on this cluster or on all
  clusters, but now marked for removal. Within this group there is no order
  guarantee.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
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
	// TODO: array: libraries

}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: `Add a library.`,
	Long: `Add a library.
  
  Add libraries to be installed on a cluster. The installation is asynchronous;
  it happens in the background after the completion of this request.
  
  **Note**: The actual set of libraries to be installed on a cluster is the
  union of the libraries specified via this method and the libraries set to be
  installed on all clusters via the libraries UI.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
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
	// TODO: array: libraries

}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: `Uninstall libraries.`,
	Long: `Uninstall libraries.
  
  Set libraries to be uninstalled on a cluster. The libraries won't be
  uninstalled until the cluster is restarted. Uninstalling libraries that are
  not installed on the cluster will have no impact but is not an error.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Libraries.Uninstall(ctx, uninstallReq)
		if err != nil {
			return err
		}

		return nil
	},
}

// end service Libraries

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

}
