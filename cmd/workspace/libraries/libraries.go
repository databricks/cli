// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package libraries

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
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
		GroupID: "compute",
		Annotations: map[string]string{
			"package": "compute",
		},
	}

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start all-cluster-statuses command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var allClusterStatusesOverrides []func(
	*cobra.Command,
)

func newAllClusterStatuses() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "all-cluster-statuses"
	cmd.Short = `Get all statuses.`
	cmd.Long = `Get all statuses.
  
  Get the status of all libraries on all clusters. A status will be available
  for all libraries installed on this cluster via the API or the libraries UI as
  well as libraries set to be installed on all clusters via the libraries UI.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		response, err := w.Libraries.AllClusterStatuses(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range allClusterStatusesOverrides {
		fn(cmd)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newAllClusterStatuses())
	})
}

// start cluster-status command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var clusterStatusOverrides []func(
	*cobra.Command,
	*compute.ClusterStatusRequest,
)

func newClusterStatus() *cobra.Command {
	cmd := &cobra.Command{}

	var clusterStatusReq compute.ClusterStatusRequest

	// TODO: short flags

	cmd.Use = "cluster-status CLUSTER_ID"
	cmd.Short = `Get status.`
	cmd.Long = `Get status.
  
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
  guarantee.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		clusterStatusReq.ClusterId = args[0]

		response, err := w.Libraries.ClusterStatus(ctx, clusterStatusReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range clusterStatusOverrides {
		fn(cmd, &clusterStatusReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newClusterStatus())
	})
}

// start install command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var installOverrides []func(
	*cobra.Command,
	*compute.InstallLibraries,
)

func newInstall() *cobra.Command {
	cmd := &cobra.Command{}

	var installReq compute.InstallLibraries
	var installJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&installJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "install"
	cmd.Short = `Add a library.`
	cmd.Long = `Add a library.
  
  Add libraries to be installed on a cluster. The installation is asynchronous;
  it happens in the background after the completion of this request.
  
  **Note**: The actual set of libraries to be installed on a cluster is the
  union of the libraries specified via this method and the libraries set to be
  installed on all clusters via the libraries UI.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = installJson.Unmarshal(&installReq)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

		err = w.Libraries.Install(ctx, installReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range installOverrides {
		fn(cmd, &installReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newInstall())
	})
}

// start uninstall command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var uninstallOverrides []func(
	*cobra.Command,
	*compute.UninstallLibraries,
)

func newUninstall() *cobra.Command {
	cmd := &cobra.Command{}

	var uninstallReq compute.UninstallLibraries
	var uninstallJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&uninstallJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "uninstall"
	cmd.Short = `Uninstall libraries.`
	cmd.Long = `Uninstall libraries.
  
  Set libraries to be uninstalled on a cluster. The libraries won't be
  uninstalled until the cluster is restarted. Uninstalling libraries that are
  not installed on the cluster will have no impact but is not an error.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = uninstallJson.Unmarshal(&uninstallReq)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

		err = w.Libraries.Uninstall(ctx, uninstallReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range uninstallOverrides {
		fn(cmd, &uninstallReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newUninstall())
	})
}

// end service Libraries
