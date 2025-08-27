// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package apps

import (
	"fmt"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apps",
		Short: `Apps run directly on a customer’s Databricks instance, integrate with their data, use and extend Databricks services, and enable users to interact through single sign-on.`,
		Long: `Apps run directly on a customer’s Databricks instance, integrate with their
  data, use and extend Databricks services, and enable users to interact through
  single sign-on.`,
		GroupID: "apps",
		Annotations: map[string]string{
			"package": "apps",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newDeploy())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newGetDeployment())
	cmd.AddCommand(newGetPermissionLevels())
	cmd.AddCommand(newGetPermissions())
	cmd.AddCommand(newList())
	cmd.AddCommand(newListDeployments())
	cmd.AddCommand(newSetPermissions())
	cmd.AddCommand(newStart())
	cmd.AddCommand(newStop())
	cmd.AddCommand(newUpdate())
	cmd.AddCommand(newUpdatePermissions())

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
	*apps.CreateAppRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq apps.CreateAppRequest
	createReq.App = apps.App{}
	var createJson flags.JsonFlag

	var createSkipWait bool
	var createTimeout time.Duration

	cmd.Flags().BoolVar(&createSkipWait, "no-wait", createSkipWait, `do not wait to reach ACTIVE state`)
	cmd.Flags().DurationVar(&createTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach ACTIVE state`)

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&createReq.NoCompute, "no-compute", createReq.NoCompute, `If true, the app will not be started after creation.`)
	// TODO: complex arg: active_deployment
	// TODO: complex arg: app_status
	cmd.Flags().StringVar(&createReq.App.BudgetPolicyId, "budget-policy-id", createReq.App.BudgetPolicyId, ``)
	// TODO: complex arg: compute_status
	cmd.Flags().StringVar(&createReq.App.Description, "description", createReq.App.Description, `The description of the app.`)
	// TODO: array: effective_user_api_scopes
	// TODO: complex arg: pending_deployment
	// TODO: array: resources
	// TODO: array: user_api_scopes

	cmd.Use = "create NAME"
	cmd.Short = `Create an app.`
	cmd.Long = `Create an app.
  
  Creates a new app.

  Arguments:
    NAME: The name of the app. The name must contain only lowercase alphanumeric
      characters and hyphens. It must be unique within the workspace.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq.App)
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
		if !cmd.Flags().Changed("json") {
			createReq.App.Name = args[0]
		}

		wait, err := w.Apps.Create(ctx, createReq)
		if err != nil {
			return err
		}
		if createSkipWait {
			return cmdio.Render(ctx, wait.Response)
		}
		spinner := cmdio.Spinner(ctx)
		info, err := wait.OnProgress(func(i *apps.App) {
			if i.ComputeStatus == nil {
				return
			}
			status := i.ComputeStatus.State
			statusMessage := fmt.Sprintf("current status: %s", status)
			if i.ComputeStatus != nil {
				statusMessage = i.ComputeStatus.Message
			}
			spinner <- statusMessage
		}).GetWithTimeout(createTimeout)
		close(spinner)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, info)
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

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*apps.DeleteAppRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq apps.DeleteAppRequest

	cmd.Use = "delete NAME"
	cmd.Short = `Delete an app.`
	cmd.Long = `Delete an app.
  
  Deletes an app.

  Arguments:
    NAME: The name of the app.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteReq.Name = args[0]

		response, err := w.Apps.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteOverrides {
		fn(cmd, &deleteReq)
	}

	return cmd
}

// start deploy command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deployOverrides []func(
	*cobra.Command,
	*apps.CreateAppDeploymentRequest,
)

func newDeploy() *cobra.Command {
	cmd := &cobra.Command{}

	var deployReq apps.CreateAppDeploymentRequest
	deployReq.AppDeployment = apps.AppDeployment{}
	var deployJson flags.JsonFlag

	var deploySkipWait bool
	var deployTimeout time.Duration

	cmd.Flags().BoolVar(&deploySkipWait, "no-wait", deploySkipWait, `do not wait to reach SUCCEEDED state`)
	cmd.Flags().DurationVar(&deployTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach SUCCEEDED state`)

	cmd.Flags().Var(&deployJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: deployment_artifacts
	cmd.Flags().StringVar(&deployReq.AppDeployment.DeploymentId, "deployment-id", deployReq.AppDeployment.DeploymentId, `The unique id of the deployment.`)
	cmd.Flags().Var(&deployReq.AppDeployment.Mode, "mode", `The mode of which the deployment will manage the source code. Supported values: [AUTO_SYNC, SNAPSHOT]`)
	cmd.Flags().StringVar(&deployReq.AppDeployment.SourceCodePath, "source-code-path", deployReq.AppDeployment.SourceCodePath, `The workspace file system path of the source code used to create the app deployment.`)
	// TODO: complex arg: status

	cmd.Use = "deploy APP_NAME"
	cmd.Short = `Create an app deployment.`
	cmd.Long = `Create an app deployment.
  
  Creates an app deployment for the app with the supplied name.

  Arguments:
    APP_NAME: The name of the app.`

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
			diags := deployJson.Unmarshal(&deployReq.AppDeployment)
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
		deployReq.AppName = args[0]

		wait, err := w.Apps.Deploy(ctx, deployReq)
		if err != nil {
			return err
		}
		if deploySkipWait {
			return cmdio.Render(ctx, wait.Response)
		}
		spinner := cmdio.Spinner(ctx)
		info, err := wait.OnProgress(func(i *apps.AppDeployment) {
			if i.Status == nil {
				return
			}
			status := i.Status.State
			statusMessage := fmt.Sprintf("current status: %s", status)
			if i.Status != nil {
				statusMessage = i.Status.Message
			}
			spinner <- statusMessage
		}).GetWithTimeout(deployTimeout)
		close(spinner)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, info)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deployOverrides {
		fn(cmd, &deployReq)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*apps.GetAppRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq apps.GetAppRequest

	cmd.Use = "get NAME"
	cmd.Short = `Get an app.`
	cmd.Long = `Get an app.
  
  Retrieves information for the app with the supplied name.

  Arguments:
    NAME: The name of the app.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getReq.Name = args[0]

		response, err := w.Apps.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getOverrides {
		fn(cmd, &getReq)
	}

	return cmd
}

// start get-deployment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getDeploymentOverrides []func(
	*cobra.Command,
	*apps.GetAppDeploymentRequest,
)

func newGetDeployment() *cobra.Command {
	cmd := &cobra.Command{}

	var getDeploymentReq apps.GetAppDeploymentRequest

	cmd.Use = "get-deployment APP_NAME DEPLOYMENT_ID"
	cmd.Short = `Get an app deployment.`
	cmd.Long = `Get an app deployment.
  
  Retrieves information for the app deployment with the supplied name and
  deployment id.

  Arguments:
    APP_NAME: The name of the app.
    DEPLOYMENT_ID: The unique id of the deployment.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getDeploymentReq.AppName = args[0]
		getDeploymentReq.DeploymentId = args[1]

		response, err := w.Apps.GetDeployment(ctx, getDeploymentReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getDeploymentOverrides {
		fn(cmd, &getDeploymentReq)
	}

	return cmd
}

// start get-permission-levels command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPermissionLevelsOverrides []func(
	*cobra.Command,
	*apps.GetAppPermissionLevelsRequest,
)

func newGetPermissionLevels() *cobra.Command {
	cmd := &cobra.Command{}

	var getPermissionLevelsReq apps.GetAppPermissionLevelsRequest

	cmd.Use = "get-permission-levels APP_NAME"
	cmd.Short = `Get app permission levels.`
	cmd.Long = `Get app permission levels.
  
  Gets the permission levels that a user can have on an object.

  Arguments:
    APP_NAME: The app for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getPermissionLevelsReq.AppName = args[0]

		response, err := w.Apps.GetPermissionLevels(ctx, getPermissionLevelsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getPermissionLevelsOverrides {
		fn(cmd, &getPermissionLevelsReq)
	}

	return cmd
}

// start get-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPermissionsOverrides []func(
	*cobra.Command,
	*apps.GetAppPermissionsRequest,
)

func newGetPermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var getPermissionsReq apps.GetAppPermissionsRequest

	cmd.Use = "get-permissions APP_NAME"
	cmd.Short = `Get app permissions.`
	cmd.Long = `Get app permissions.
  
  Gets the permissions of an app. Apps can inherit permissions from their root
  object.

  Arguments:
    APP_NAME: The app for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getPermissionsReq.AppName = args[0]

		response, err := w.Apps.GetPermissions(ctx, getPermissionsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getPermissionsOverrides {
		fn(cmd, &getPermissionsReq)
	}

	return cmd
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*apps.ListAppsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq apps.ListAppsRequest

	cmd.Flags().IntVar(&listReq.PageSize, "page-size", listReq.PageSize, `Upper bound for items returned.`)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Pagination token to go to the next page of apps.`)

	cmd.Use = "list"
	cmd.Short = `List apps.`
	cmd.Long = `List apps.
  
  Lists all apps in the workspace.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.Apps.List(ctx, listReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOverrides {
		fn(cmd, &listReq)
	}

	return cmd
}

// start list-deployments command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listDeploymentsOverrides []func(
	*cobra.Command,
	*apps.ListAppDeploymentsRequest,
)

func newListDeployments() *cobra.Command {
	cmd := &cobra.Command{}

	var listDeploymentsReq apps.ListAppDeploymentsRequest

	cmd.Flags().IntVar(&listDeploymentsReq.PageSize, "page-size", listDeploymentsReq.PageSize, `Upper bound for items returned.`)
	cmd.Flags().StringVar(&listDeploymentsReq.PageToken, "page-token", listDeploymentsReq.PageToken, `Pagination token to go to the next page of apps.`)

	cmd.Use = "list-deployments APP_NAME"
	cmd.Short = `List app deployments.`
	cmd.Long = `List app deployments.
  
  Lists all app deployments for the app with the supplied name.

  Arguments:
    APP_NAME: The name of the app.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listDeploymentsReq.AppName = args[0]

		response := w.Apps.ListDeployments(ctx, listDeploymentsReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listDeploymentsOverrides {
		fn(cmd, &listDeploymentsReq)
	}

	return cmd
}

// start set-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var setPermissionsOverrides []func(
	*cobra.Command,
	*apps.AppPermissionsRequest,
)

func newSetPermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var setPermissionsReq apps.AppPermissionsRequest
	var setPermissionsJson flags.JsonFlag

	cmd.Flags().Var(&setPermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "set-permissions APP_NAME"
	cmd.Short = `Set app permissions.`
	cmd.Long = `Set app permissions.
  
  Sets permissions on an object, replacing existing permissions if they exist.
  Deletes all direct permissions if none are specified. Objects can inherit
  permissions from their root object.

  Arguments:
    APP_NAME: The app for which to get or manage permissions.`

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
			diags := setPermissionsJson.Unmarshal(&setPermissionsReq)
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
		setPermissionsReq.AppName = args[0]

		response, err := w.Apps.SetPermissions(ctx, setPermissionsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range setPermissionsOverrides {
		fn(cmd, &setPermissionsReq)
	}

	return cmd
}

// start start command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var startOverrides []func(
	*cobra.Command,
	*apps.StartAppRequest,
)

func newStart() *cobra.Command {
	cmd := &cobra.Command{}

	var startReq apps.StartAppRequest

	var startSkipWait bool
	var startTimeout time.Duration

	cmd.Flags().BoolVar(&startSkipWait, "no-wait", startSkipWait, `do not wait to reach ACTIVE state`)
	cmd.Flags().DurationVar(&startTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach ACTIVE state`)

	cmd.Use = "start NAME"
	cmd.Short = `Start an app.`
	cmd.Long = `Start an app.
  
  Start the last active deployment of the app in the workspace.

  Arguments:
    NAME: The name of the app.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		startReq.Name = args[0]

		wait, err := w.Apps.Start(ctx, startReq)
		if err != nil {
			return err
		}
		if startSkipWait {
			return cmdio.Render(ctx, wait.Response)
		}
		spinner := cmdio.Spinner(ctx)
		info, err := wait.OnProgress(func(i *apps.App) {
			if i.ComputeStatus == nil {
				return
			}
			status := i.ComputeStatus.State
			statusMessage := fmt.Sprintf("current status: %s", status)
			if i.ComputeStatus != nil {
				statusMessage = i.ComputeStatus.Message
			}
			spinner <- statusMessage
		}).GetWithTimeout(startTimeout)
		close(spinner)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, info)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range startOverrides {
		fn(cmd, &startReq)
	}

	return cmd
}

// start stop command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var stopOverrides []func(
	*cobra.Command,
	*apps.StopAppRequest,
)

func newStop() *cobra.Command {
	cmd := &cobra.Command{}

	var stopReq apps.StopAppRequest

	var stopSkipWait bool
	var stopTimeout time.Duration

	cmd.Flags().BoolVar(&stopSkipWait, "no-wait", stopSkipWait, `do not wait to reach STOPPED state`)
	cmd.Flags().DurationVar(&stopTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach STOPPED state`)

	cmd.Use = "stop NAME"
	cmd.Short = `Stop an app.`
	cmd.Long = `Stop an app.
  
  Stops the active deployment of the app in the workspace.

  Arguments:
    NAME: The name of the app.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		stopReq.Name = args[0]

		wait, err := w.Apps.Stop(ctx, stopReq)
		if err != nil {
			return err
		}
		if stopSkipWait {
			return cmdio.Render(ctx, wait.Response)
		}
		spinner := cmdio.Spinner(ctx)
		info, err := wait.OnProgress(func(i *apps.App) {
			if i.ComputeStatus == nil {
				return
			}
			status := i.ComputeStatus.State
			statusMessage := fmt.Sprintf("current status: %s", status)
			if i.ComputeStatus != nil {
				statusMessage = i.ComputeStatus.Message
			}
			spinner <- statusMessage
		}).GetWithTimeout(stopTimeout)
		close(spinner)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, info)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range stopOverrides {
		fn(cmd, &stopReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*apps.UpdateAppRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq apps.UpdateAppRequest
	updateReq.App = apps.App{}
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: active_deployment
	// TODO: complex arg: app_status
	cmd.Flags().StringVar(&updateReq.App.BudgetPolicyId, "budget-policy-id", updateReq.App.BudgetPolicyId, ``)
	// TODO: complex arg: compute_status
	cmd.Flags().StringVar(&updateReq.App.Description, "description", updateReq.App.Description, `The description of the app.`)
	// TODO: array: effective_user_api_scopes
	// TODO: complex arg: pending_deployment
	// TODO: array: resources
	// TODO: array: user_api_scopes

	cmd.Use = "update NAME"
	cmd.Short = `Update an app.`
	cmd.Long = `Update an app.
  
  Updates the app with the supplied name.

  Arguments:
    NAME: The name of the app. The name must contain only lowercase alphanumeric
      characters and hyphens. It must be unique within the workspace.`

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
			diags := updateJson.Unmarshal(&updateReq.App)
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
		updateReq.Name = args[0]

		response, err := w.Apps.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateOverrides {
		fn(cmd, &updateReq)
	}

	return cmd
}

// start update-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updatePermissionsOverrides []func(
	*cobra.Command,
	*apps.AppPermissionsRequest,
)

func newUpdatePermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var updatePermissionsReq apps.AppPermissionsRequest
	var updatePermissionsJson flags.JsonFlag

	cmd.Flags().Var(&updatePermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "update-permissions APP_NAME"
	cmd.Short = `Update app permissions.`
	cmd.Long = `Update app permissions.
  
  Updates the permissions on an app. Apps can inherit permissions from their
  root object.

  Arguments:
    APP_NAME: The app for which to get or manage permissions.`

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
			diags := updatePermissionsJson.Unmarshal(&updatePermissionsReq)
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
		updatePermissionsReq.AppName = args[0]

		response, err := w.Apps.UpdatePermissions(ctx, updatePermissionsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updatePermissionsOverrides {
		fn(cmd, &updatePermissionsReq)
	}

	return cmd
}

// end service Apps
