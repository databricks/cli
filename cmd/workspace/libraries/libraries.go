// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package libraries

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	clusterlibraries "github.com/databricks/cli/libs/sdktypes/clusterlibraries/v2"
	sdkclusterlibraries "github.com/databricks/sdk-go/clusterlibraries/v2"
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
  Java, Scala, and R. You can upload Python, Java, Scala and R libraries and
  point to external packages in PyPI, Maven, and CRAN repositories.

  Cluster libraries can be used by all notebooks running on a cluster. You can
  install a cluster library directly from a public repository such as PyPI or
  Maven, using a previously installed workspace library, or using an init
  script.

  When you uninstall a library from a cluster, the library is removed only when
  you restart the cluster. Until you restart the cluster, the status of the
  uninstalled library appears as Uninstall pending restart.`,
		GroupID: "compute",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newAllClusterStatuses())
	cmd.AddCommand(newClusterStatus())
	cmd.AddCommand(newInstall())
	cmd.AddCommand(newUninstall())

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
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var allClusterStatusesLimit int

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&allClusterStatusesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).

	cmd.Use = "all-cluster-statuses"
	cmd.Short = `Get all statuses.`
	cmd.Long = `Get all statuses.

  Get the status of all libraries on all clusters. A status is returned for all
  libraries installed on this cluster via the API or the libraries UI.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq := &sdkclusterlibraries.ListAllClusterLibraryStatusesRequest{}

		response, err := w.Libraries.AllClusterStatuses(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if allClusterStatusesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", allClusterStatusesLimit)
		}
		if allClusterStatusesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, allClusterStatusesLimit)
		}

		out, err := clusterlibraries.ListAllClusterLibraryStatusesResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out.Statuses)
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

// start cluster-status command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var clusterStatusOverrides []func(
	*cobra.Command,
	*clusterlibraries.ClusterStatusRequest,
)

func newClusterStatus() *cobra.Command {
	cmd := &cobra.Command{}
	var clusterStatusReq clusterlibraries.ClusterStatusRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var clusterStatusLimit int

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&clusterStatusLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).

	cmd.Use = "cluster-status CLUSTER_ID"
	cmd.Short = `Get status.`
	cmd.Long = `Get status.

  Get the status of libraries on a cluster. A status is returned for all
  libraries installed on this cluster via the API or the libraries UI. The order
  of returned libraries is as follows: 1. Libraries set to be installed on this
  cluster, in the order that the libraries were added to the cluster, are
  returned first. 2. Libraries that were previously requested to be installed on
  this cluster or, but are now marked for removal, in no particular order, are
  returned last.

  Arguments:
    CLUSTER_ID: Unique identifier of the cluster whose status should be retrieved.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		clusterStatusReq.ClusterId = new(args[0])

		sdkReq, err := clusterlibraries.ClusterStatusRequestToSDK(&clusterStatusReq)
		if err != nil {
			return err
		}
		response, err := w.Libraries.ClusterStatus(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if clusterStatusLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", clusterStatusLimit)
		}
		if clusterStatusLimit > 0 {
			ctx = cmdio.WithLimit(ctx, clusterStatusLimit)
		}

		out, err := clusterlibraries.ClusterLibraryStatusesFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out.LibraryStatuses)
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

// start install command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var installOverrides []func(
	*cobra.Command,
	*clusterlibraries.InstallLibrariesRequest,
)

func newInstall() *cobra.Command {
	cmd := &cobra.Command{}
	var installReq clusterlibraries.InstallLibrariesRequest

	var installJson flags.JsonFlag

	cmd.Flags().Var(&installJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "install"
	cmd.Short = `Add a library.`
	cmd.Long = `Add a library.

  Add libraries to install on a cluster. The installation is asynchronous; it
  happens in the background after the completion of this request.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := installJson.Unmarshal(&installReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return errors.New("please provide command input in JSON format by specifying the --json flag")
		}

		sdkReq, err := clusterlibraries.InstallLibrariesRequestToSDK(&installReq)
		if err != nil {
			return err
		}
		response, err := w.Libraries.InstallLibraries(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
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

// start uninstall command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var uninstallOverrides []func(
	*cobra.Command,
	*clusterlibraries.UninstallLibrariesRequest,
)

func newUninstall() *cobra.Command {
	cmd := &cobra.Command{}
	var uninstallReq clusterlibraries.UninstallLibrariesRequest

	var uninstallJson flags.JsonFlag

	cmd.Flags().Var(&uninstallJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "uninstall"
	cmd.Short = `Uninstall libraries.`
	cmd.Long = `Uninstall libraries.

  Set libraries to uninstall from a cluster. The libraries won't be uninstalled
  until the cluster is restarted. A request to uninstall a library that is not
  currently installed is ignored.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := uninstallJson.Unmarshal(&uninstallReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return errors.New("please provide command input in JSON format by specifying the --json flag")
		}

		sdkReq, err := clusterlibraries.UninstallLibrariesRequestToSDK(&uninstallReq)
		if err != nil {
			return err
		}
		response, err := w.Libraries.UninstallLibraries(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
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

// end service Libraries
