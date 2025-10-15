// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package libraries

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
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
		Annotations: map[string]string{
			"package": "compute",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newAllClusterStatuses())
	cmd.AddCommand(newClusterStatus())
	cmd.AddCommand(newCreateDefaultBaseEnvironment())
	cmd.AddCommand(newDeleteDefaultBaseEnvironment())
	cmd.AddCommand(newGetDefaultBaseEnvironment())
	cmd.AddCommand(newInstall())
	cmd.AddCommand(newListDefaultBaseEnvironments())
	cmd.AddCommand(newRefreshDefaultBaseEnvironments())
	cmd.AddCommand(newUninstall())
	cmd.AddCommand(newUpdateDefaultBaseEnvironment())
	cmd.AddCommand(newUpdateDefaultDefaultBaseEnvironment())

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
  
  Get the status of all libraries on all clusters. A status is returned for all
  libraries installed on this cluster via the API or the libraries UI.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		response := w.Libraries.AllClusterStatuses(ctx)
		return cmdio.RenderIterator(ctx, response)
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
	*compute.ClusterStatus,
)

func newClusterStatus() *cobra.Command {
	cmd := &cobra.Command{}

	var clusterStatusReq compute.ClusterStatus

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

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		clusterStatusReq.ClusterId = args[0]

		response := w.Libraries.ClusterStatus(ctx, clusterStatusReq)
		return cmdio.RenderIterator(ctx, response)
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

// start create-default-base-environment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createDefaultBaseEnvironmentOverrides []func(
	*cobra.Command,
	*compute.CreateDefaultBaseEnvironmentRequest,
)

func newCreateDefaultBaseEnvironment() *cobra.Command {
	cmd := &cobra.Command{}

	var createDefaultBaseEnvironmentReq compute.CreateDefaultBaseEnvironmentRequest
	var createDefaultBaseEnvironmentJson flags.JsonFlag

	cmd.Flags().Var(&createDefaultBaseEnvironmentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createDefaultBaseEnvironmentReq.RequestId, "request-id", createDefaultBaseEnvironmentReq.RequestId, `A unique identifier for this request.`)

	cmd.Use = "create-default-base-environment"
	cmd.Short = `Create a default base environment.`
	cmd.Long = `Create a default base environment.
  
  Create a default base environment within workspaces to define the environment
  version and a list of dependencies to be used in serverless notebooks and
  jobs. This process will asynchronously generate a cache to optimize dependency
  resolution.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createDefaultBaseEnvironmentJson.Unmarshal(&createDefaultBaseEnvironmentReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

		response, err := w.Libraries.CreateDefaultBaseEnvironment(ctx, createDefaultBaseEnvironmentReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createDefaultBaseEnvironmentOverrides {
		fn(cmd, &createDefaultBaseEnvironmentReq)
	}

	return cmd
}

// start delete-default-base-environment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteDefaultBaseEnvironmentOverrides []func(
	*cobra.Command,
	*compute.DeleteDefaultBaseEnvironmentRequest,
)

func newDeleteDefaultBaseEnvironment() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteDefaultBaseEnvironmentReq compute.DeleteDefaultBaseEnvironmentRequest

	cmd.Use = "delete-default-base-environment ID"
	cmd.Short = `Delete a default base environment.`
	cmd.Long = `Delete a default base environment.
  
  Delete the default base environment given an ID. The default base environment
  may be used by downstream workloads. Please ensure that the deletion is
  intentional.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteDefaultBaseEnvironmentReq.Id = args[0]

		err = w.Libraries.DeleteDefaultBaseEnvironment(ctx, deleteDefaultBaseEnvironmentReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteDefaultBaseEnvironmentOverrides {
		fn(cmd, &deleteDefaultBaseEnvironmentReq)
	}

	return cmd
}

// start get-default-base-environment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getDefaultBaseEnvironmentOverrides []func(
	*cobra.Command,
	*compute.GetDefaultBaseEnvironmentRequest,
)

func newGetDefaultBaseEnvironment() *cobra.Command {
	cmd := &cobra.Command{}

	var getDefaultBaseEnvironmentReq compute.GetDefaultBaseEnvironmentRequest

	cmd.Use = "get-default-base-environment ID"
	cmd.Short = `get a default base environment.`
	cmd.Long = `get a default base environment.
  
  Return the default base environment details for a given ID.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getDefaultBaseEnvironmentReq.Id = args[0]

		response, err := w.Libraries.GetDefaultBaseEnvironment(ctx, getDefaultBaseEnvironmentReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getDefaultBaseEnvironmentOverrides {
		fn(cmd, &getDefaultBaseEnvironmentReq)
	}

	return cmd
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

	cmd.Flags().Var(&installJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "install"
	cmd.Short = `Add a library.`
	cmd.Long = `Add a library.
  
  Add libraries to install on a cluster. The installation is asynchronous; it
  happens in the background after the completion of this request.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := installJson.Unmarshal(&installReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
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

// start list-default-base-environments command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listDefaultBaseEnvironmentsOverrides []func(
	*cobra.Command,
	*compute.ListDefaultBaseEnvironmentsRequest,
)

func newListDefaultBaseEnvironments() *cobra.Command {
	cmd := &cobra.Command{}

	var listDefaultBaseEnvironmentsReq compute.ListDefaultBaseEnvironmentsRequest

	cmd.Flags().IntVar(&listDefaultBaseEnvironmentsReq.PageSize, "page-size", listDefaultBaseEnvironmentsReq.PageSize, ``)
	cmd.Flags().StringVar(&listDefaultBaseEnvironmentsReq.PageToken, "page-token", listDefaultBaseEnvironmentsReq.PageToken, ``)

	cmd.Use = "list-default-base-environments"
	cmd.Short = `List default base environments.`
	cmd.Long = `List default base environments.
  
  List default base environments defined in the workspaces for the requested
  user.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.Libraries.ListDefaultBaseEnvironments(ctx, listDefaultBaseEnvironmentsReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listDefaultBaseEnvironmentsOverrides {
		fn(cmd, &listDefaultBaseEnvironmentsReq)
	}

	return cmd
}

// start refresh-default-base-environments command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var refreshDefaultBaseEnvironmentsOverrides []func(
	*cobra.Command,
	*compute.RefreshDefaultBaseEnvironmentsRequest,
)

func newRefreshDefaultBaseEnvironments() *cobra.Command {
	cmd := &cobra.Command{}

	var refreshDefaultBaseEnvironmentsReq compute.RefreshDefaultBaseEnvironmentsRequest
	var refreshDefaultBaseEnvironmentsJson flags.JsonFlag

	cmd.Flags().Var(&refreshDefaultBaseEnvironmentsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "refresh-default-base-environments"
	cmd.Short = `.`
	cmd.Long = `.
  
  Refresh the cached default base environments for the given IDs. This process
  will asynchronously regenerate the caches. The existing caches remains
  available until it expires.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := refreshDefaultBaseEnvironmentsJson.Unmarshal(&refreshDefaultBaseEnvironmentsReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

		err = w.Libraries.RefreshDefaultBaseEnvironments(ctx, refreshDefaultBaseEnvironmentsReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range refreshDefaultBaseEnvironmentsOverrides {
		fn(cmd, &refreshDefaultBaseEnvironmentsReq)
	}

	return cmd
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

	cmd.Flags().Var(&uninstallJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "uninstall"
	cmd.Short = `Uninstall libraries.`
	cmd.Long = `Uninstall libraries.
  
  Set libraries to uninstall from a cluster. The libraries won't be uninstalled
  until the cluster is restarted. A request to uninstall a library that is not
  currently installed is ignored.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := uninstallJson.Unmarshal(&uninstallReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
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

// start update-default-base-environment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateDefaultBaseEnvironmentOverrides []func(
	*cobra.Command,
	*compute.UpdateDefaultBaseEnvironmentRequest,
)

func newUpdateDefaultBaseEnvironment() *cobra.Command {
	cmd := &cobra.Command{}

	var updateDefaultBaseEnvironmentReq compute.UpdateDefaultBaseEnvironmentRequest
	var updateDefaultBaseEnvironmentJson flags.JsonFlag

	cmd.Flags().Var(&updateDefaultBaseEnvironmentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "update-default-base-environment ID"
	cmd.Short = `Update a default base environment.`
	cmd.Long = `Update a default base environment.
  
  Update the default base environment for the given ID. This process will
  asynchronously regenerate the cache. The existing cache remains available
  until it expires.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateDefaultBaseEnvironmentJson.Unmarshal(&updateDefaultBaseEnvironmentReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}
		updateDefaultBaseEnvironmentReq.Id = args[0]

		response, err := w.Libraries.UpdateDefaultBaseEnvironment(ctx, updateDefaultBaseEnvironmentReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateDefaultBaseEnvironmentOverrides {
		fn(cmd, &updateDefaultBaseEnvironmentReq)
	}

	return cmd
}

// start update-default-default-base-environment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateDefaultDefaultBaseEnvironmentOverrides []func(
	*cobra.Command,
	*compute.UpdateDefaultDefaultBaseEnvironmentRequest,
)

func newUpdateDefaultDefaultBaseEnvironment() *cobra.Command {
	cmd := &cobra.Command{}

	var updateDefaultDefaultBaseEnvironmentReq compute.UpdateDefaultDefaultBaseEnvironmentRequest
	var updateDefaultDefaultBaseEnvironmentJson flags.JsonFlag

	cmd.Flags().Var(&updateDefaultDefaultBaseEnvironmentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().Var(&updateDefaultDefaultBaseEnvironmentReq.BaseEnvironmentType, "base-environment-type", `Supported values: [CPU, GPU]`)
	cmd.Flags().StringVar(&updateDefaultDefaultBaseEnvironmentReq.Id, "id", updateDefaultDefaultBaseEnvironmentReq.Id, ``)

	cmd.Use = "update-default-default-base-environment"
	cmd.Short = `Update the default default base environment.`
	cmd.Long = `Update the default default base environment.
  
  Set the default base environment for the workspace. This marks the specified
  DBE as the workspace default.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateDefaultDefaultBaseEnvironmentJson.Unmarshal(&updateDefaultDefaultBaseEnvironmentReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		}

		response, err := w.Libraries.UpdateDefaultDefaultBaseEnvironment(ctx, updateDefaultDefaultBaseEnvironmentReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateDefaultDefaultBaseEnvironmentOverrides {
		fn(cmd, &updateDefaultDefaultBaseEnvironmentReq)
	}

	return cmd
}

// end service Libraries
