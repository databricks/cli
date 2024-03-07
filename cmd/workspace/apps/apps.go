// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package apps

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apps",
		Short: `Lakehouse Apps run directly on a customer’s Databricks instance, integrate with their data, use and extend Databricks services, and enable users to interact through single sign-on.`,
		Long: `Lakehouse Apps run directly on a customer’s Databricks instance, integrate
  with their data, use and extend Databricks services, and enable users to
  interact through single sign-on.`,
		GroupID: "serving",
		Annotations: map[string]string{
			"package": "serving",
		},

		// This service is being previewed; hide from help output.
		Hidden: true,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDeleteApp())
	cmd.AddCommand(newGetApp())
	cmd.AddCommand(newGetAppDeploymentStatus())
	cmd.AddCommand(newGetApps())
	cmd.AddCommand(newGetEvents())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOverrides []func(
	*cobra.Command,
	*serving.DeployAppRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq serving.DeployAppRequest
	var createJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: any: resources

	cmd.Use = "create"
	cmd.Short = `Create and deploy an application.`
	cmd.Long = `Create and deploy an application.
  
  Creates and deploys an application.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

		response, err := w.Apps.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createOverrides {
		fn(cmd, &createReq)
	}

	return cmd
}

// start delete-app command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteAppOverrides []func(
	*cobra.Command,
	*serving.DeleteAppRequest,
)

func newDeleteApp() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteAppReq serving.DeleteAppRequest

	// TODO: short flags

	cmd.Use = "delete-app NAME"
	cmd.Short = `Delete an application.`
	cmd.Long = `Delete an application.
  
  Delete an application definition

  Arguments:
    NAME: The name of an application. This field is required.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		deleteAppReq.Name = args[0]

		response, err := w.Apps.DeleteApp(ctx, deleteAppReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteAppOverrides {
		fn(cmd, &deleteAppReq)
	}

	return cmd
}

// start get-app command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getAppOverrides []func(
	*cobra.Command,
	*serving.GetAppRequest,
)

func newGetApp() *cobra.Command {
	cmd := &cobra.Command{}

	var getAppReq serving.GetAppRequest

	// TODO: short flags

	cmd.Use = "get-app NAME"
	cmd.Short = `Get definition for an application.`
	cmd.Long = `Get definition for an application.
  
  Get an application definition

  Arguments:
    NAME: The name of an application. This field is required.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		getAppReq.Name = args[0]

		response, err := w.Apps.GetApp(ctx, getAppReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getAppOverrides {
		fn(cmd, &getAppReq)
	}

	return cmd
}

// start get-app-deployment-status command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getAppDeploymentStatusOverrides []func(
	*cobra.Command,
	*serving.GetAppDeploymentStatusRequest,
)

func newGetAppDeploymentStatus() *cobra.Command {
	cmd := &cobra.Command{}

	var getAppDeploymentStatusReq serving.GetAppDeploymentStatusRequest

	// TODO: short flags

	cmd.Flags().StringVar(&getAppDeploymentStatusReq.IncludeAppLog, "include-app-log", getAppDeploymentStatusReq.IncludeAppLog, `Boolean flag to include application logs.`)

	cmd.Use = "get-app-deployment-status DEPLOYMENT_ID"
	cmd.Short = `Get deployment status for an application.`
	cmd.Long = `Get deployment status for an application.
  
  Get deployment status for an application

  Arguments:
    DEPLOYMENT_ID: The deployment id for an application. This field is required.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		getAppDeploymentStatusReq.DeploymentId = args[0]

		response, err := w.Apps.GetAppDeploymentStatus(ctx, getAppDeploymentStatusReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getAppDeploymentStatusOverrides {
		fn(cmd, &getAppDeploymentStatusReq)
	}

	return cmd
}

// start get-apps command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getAppsOverrides []func(
	*cobra.Command,
)

func newGetApps() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "get-apps"
	cmd.Short = `List all applications.`
	cmd.Long = `List all applications.
  
  List all available applications`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		response, err := w.Apps.GetApps(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getAppsOverrides {
		fn(cmd)
	}

	return cmd
}

// start get-events command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getEventsOverrides []func(
	*cobra.Command,
	*serving.GetEventsRequest,
)

func newGetEvents() *cobra.Command {
	cmd := &cobra.Command{}

	var getEventsReq serving.GetEventsRequest

	// TODO: short flags

	cmd.Use = "get-events NAME"
	cmd.Short = `Get deployment events for an application.`
	cmd.Long = `Get deployment events for an application.
  
  Get deployment events for an application

  Arguments:
    NAME: The name of an application. This field is required.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		getEventsReq.Name = args[0]

		response, err := w.Apps.GetEvents(ctx, getEventsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getEventsOverrides {
		fn(cmd, &getEventsReq)
	}

	return cmd
}

// end service Apps
