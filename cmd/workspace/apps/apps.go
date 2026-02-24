// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package apps

import (
	"fmt"
	"strings"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/experimental/api"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apps",
		Short: `Apps run directly on a customer's Databricks instance, integrate with their data, use and extend Databricks services, and enable users to interact through single sign-on.`,
		Long: `Apps run directly on a customer's Databricks instance, integrate with their
  data, use and extend Databricks services, and enable users to interact through
  single sign-on.`,
		GroupID: "apps",
		RunE:    root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newCreateSpace())
	cmd.AddCommand(newCreateUpdate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newDeleteSpace())
	cmd.AddCommand(newDeploy())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newGetDeployment())
	cmd.AddCommand(newGetPermissionLevels())
	cmd.AddCommand(newGetPermissions())
	cmd.AddCommand(newGetSpace())
	cmd.AddCommand(newGetSpaceOperation())
	cmd.AddCommand(newGetUpdate())
	cmd.AddCommand(newList())
	cmd.AddCommand(newListDeployments())
	cmd.AddCommand(newListSpaces())
	cmd.AddCommand(newSetPermissions())
	cmd.AddCommand(newStart())
	cmd.AddCommand(newStop())
	cmd.AddCommand(newUpdate())
	cmd.AddCommand(newUpdatePermissions())
	cmd.AddCommand(newUpdateSpace())

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
	cmd.Flags().Var(&createReq.App.ComputeSize, "compute-size", `Supported values: [LARGE, MEDIUM]`)
	// TODO: complex arg: compute_status
	cmd.Flags().StringVar(&createReq.App.Description, "description", createReq.App.Description, `The description of the app.`)
	// TODO: array: effective_user_api_scopes
	// TODO: complex arg: git_repository
	// TODO: complex arg: pending_deployment
	// TODO: array: resources
	cmd.Flags().StringVar(&createReq.App.Space, "space", createReq.App.Space, `Name of the space this app belongs to.`)
	cmd.Flags().StringVar(&createReq.App.UsagePolicyId, "usage-policy-id", createReq.App.UsagePolicyId, ``)
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
				err := cmdio.RenderDiagnostics(ctx, diags)
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

// start create-space command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createSpaceOverrides []func(
	*cobra.Command,
	*apps.CreateSpaceRequest,
)

func newCreateSpace() *cobra.Command {
	cmd := &cobra.Command{}

	var createSpaceReq apps.CreateSpaceRequest
	createSpaceReq.Space = apps.Space{}
	var createSpaceJson flags.JsonFlag

	var createSpaceSkipWait bool
	var createSpaceTimeout time.Duration

	cmd.Flags().BoolVar(&createSpaceSkipWait, "no-wait", createSpaceSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&createSpaceTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&createSpaceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createSpaceReq.Space.Description, "description", createSpaceReq.Space.Description, `The description of the app space.`)
	// TODO: array: effective_user_api_scopes
	// TODO: array: resources
	// TODO: complex arg: status
	cmd.Flags().StringVar(&createSpaceReq.Space.UsagePolicyId, "usage-policy-id", createSpaceReq.Space.UsagePolicyId, `The usage policy ID for managing cost at the space level.`)
	// TODO: array: user_api_scopes

	cmd.Use = "create-space NAME"
	cmd.Short = `Create an app space.`
	cmd.Long = `Create an app space.

  Creates a new app space.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-space-operation command.

  Arguments:
    NAME: The name of the app space. The name must contain only lowercase
      alphanumeric characters and hyphens. It must be unique within the
      workspace.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

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
			diags := createSpaceJson.Unmarshal(&createSpaceReq.Space)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		if !cmd.Flags().Changed("json") {
			createSpaceReq.Space.Name = args[0]
		}

		// Determine which mode to execute based on flags.
		switch {
		case createSpaceSkipWait:
			wait, err := w.Apps.CreateSpace(ctx, createSpaceReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			operation, err := w.Apps.GetSpaceOperation(ctx, apps.GetOperationRequest{
				Name: wait.Name(),
			})
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, operation)

		default:
			wait, err := w.Apps.CreateSpace(ctx, createSpaceReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			spinner := cmdio.Spinner(ctx)
			spinner <- "Waiting for create-space to complete..."

			// Wait for completion.
			opts := api.WithTimeout(createSpaceTimeout)
			response, err := wait.Wait(ctx, opts)
			if err != nil {
				return err
			}
			close(spinner)
			return cmdio.Render(ctx, response)
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createSpaceOverrides {
		fn(cmd, &createSpaceReq)
	}

	return cmd
}

// start create-update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createUpdateOverrides []func(
	*cobra.Command,
	*apps.AsyncUpdateAppRequest,
)

func newCreateUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var createUpdateReq apps.AsyncUpdateAppRequest
	var createUpdateJson flags.JsonFlag

	var createUpdateSkipWait bool
	var createUpdateTimeout time.Duration

	cmd.Flags().BoolVar(&createUpdateSkipWait, "no-wait", createUpdateSkipWait, `do not wait to reach SUCCEEDED state`)
	cmd.Flags().DurationVar(&createUpdateTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach SUCCEEDED state`)

	cmd.Flags().Var(&createUpdateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: app

	cmd.Use = "create-update APP_NAME UPDATE_MASK"
	cmd.Short = `Create an app update.`
	cmd.Long = `Create an app update.

  Creates an app update and starts the update process. The update process is
  asynchronous and the status of the update can be checked with the GetAppUpdate
  method.

  Arguments:
    APP_NAME:
    UPDATE_MASK: The field mask must be a single string, with multiple fields separated by
      commas (no spaces). The field path is relative to the resource object,
      using a dot (.) to navigate sub-fields (e.g., author.given_name).
      Specification of elements in sequence or map fields is not allowed, as
      only the entire collection field can be specified. Field names must
      exactly match the resource field names.

      A field mask of * indicates full replacement. It’s recommended to
      always explicitly list the fields being updated and avoid using *
      wildcards, as it can lead to unintended results if the API changes in the
      future.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only APP_NAME as positional arguments. Provide 'update_mask' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createUpdateJson.Unmarshal(&createUpdateReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		createUpdateReq.AppName = args[0]
		if !cmd.Flags().Changed("json") {
			createUpdateReq.UpdateMask = args[1]
		}

		wait, err := w.Apps.CreateUpdate(ctx, createUpdateReq)
		if err != nil {
			return err
		}
		if createUpdateSkipWait {
			return cmdio.Render(ctx, wait.Response)
		}
		spinner := cmdio.Spinner(ctx)
		info, err := wait.OnProgress(func(i *apps.AppUpdate) {
			if i.Status == nil {
				return
			}
			status := i.Status.State
			statusMessage := fmt.Sprintf("current status: %s", status)
			if i.Status != nil {
				statusMessage = i.Status.Message
			}
			spinner <- statusMessage
		}).GetWithTimeout(createUpdateTimeout)
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
	for _, fn := range createUpdateOverrides {
		fn(cmd, &createUpdateReq)
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

// start delete-space command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteSpaceOverrides []func(
	*cobra.Command,
	*apps.DeleteSpaceRequest,
)

func newDeleteSpace() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteSpaceReq apps.DeleteSpaceRequest

	var deleteSpaceSkipWait bool
	var deleteSpaceTimeout time.Duration

	cmd.Flags().BoolVar(&deleteSpaceSkipWait, "no-wait", deleteSpaceSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&deleteSpaceTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Use = "delete-space NAME"
	cmd.Short = `Delete an app space.`
	cmd.Long = `Delete an app space.

  Deletes an app space.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-space-operation command.

  Arguments:
    NAME: The name of the app space.`

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

		deleteSpaceReq.Name = args[0]

		// Determine which mode to execute based on flags.
		switch {
		case deleteSpaceSkipWait:
			wait, err := w.Apps.DeleteSpace(ctx, deleteSpaceReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			operation, err := w.Apps.GetSpaceOperation(ctx, apps.GetOperationRequest{
				Name: wait.Name(),
			})
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, operation)

		default:
			wait, err := w.Apps.DeleteSpace(ctx, deleteSpaceReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			spinner := cmdio.Spinner(ctx)
			spinner <- "Waiting for delete-space to complete..."

			// Wait for completion.
			opts := api.WithTimeout(deleteSpaceTimeout)

			err = wait.Wait(ctx, opts)
			if err != nil {
				return err
			}
			close(spinner)
			return nil
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteSpaceOverrides {
		fn(cmd, &deleteSpaceReq)
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

	// TODO: array: command
	// TODO: complex arg: deployment_artifacts
	cmd.Flags().StringVar(&deployReq.AppDeployment.DeploymentId, "deployment-id", deployReq.AppDeployment.DeploymentId, `The unique id of the deployment.`)
	// TODO: array: env_vars
	// TODO: complex arg: git_source
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
				err := cmdio.RenderDiagnostics(ctx, diags)
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

// start get-space command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getSpaceOverrides []func(
	*cobra.Command,
	*apps.GetSpaceRequest,
)

func newGetSpace() *cobra.Command {
	cmd := &cobra.Command{}

	var getSpaceReq apps.GetSpaceRequest

	cmd.Use = "get-space NAME"
	cmd.Short = `Get an app space.`
	cmd.Long = `Get an app space.

  Retrieves information for the app space with the supplied name.

  Arguments:
    NAME: The name of the app space.`

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

		getSpaceReq.Name = args[0]

		response, err := w.Apps.GetSpace(ctx, getSpaceReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getSpaceOverrides {
		fn(cmd, &getSpaceReq)
	}

	return cmd
}

// start get-space-operation command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getSpaceOperationOverrides []func(
	*cobra.Command,
	*apps.GetOperationRequest,
)

func newGetSpaceOperation() *cobra.Command {
	cmd := &cobra.Command{}

	var getSpaceOperationReq apps.GetOperationRequest

	cmd.Use = "get-space-operation NAME"
	cmd.Short = `Get the status of an app space operation.`
	cmd.Long = `Get the status of an app space operation.

  Gets the status of an app space update operation.

  Arguments:
    NAME: The name of the operation resource.`

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

		getSpaceOperationReq.Name = args[0]

		response, err := w.Apps.GetSpaceOperation(ctx, getSpaceOperationReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getSpaceOperationOverrides {
		fn(cmd, &getSpaceOperationReq)
	}

	return cmd
}

// start get-update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getUpdateOverrides []func(
	*cobra.Command,
	*apps.GetAppUpdateRequest,
)

func newGetUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var getUpdateReq apps.GetAppUpdateRequest

	cmd.Use = "get-update APP_NAME"
	cmd.Short = `Get an app update.`
	cmd.Long = `Get an app update.

  Gets the status of an app update.

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

		getUpdateReq.AppName = args[0]

		response, err := w.Apps.GetUpdate(ctx, getUpdateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getUpdateOverrides {
		fn(cmd, &getUpdateReq)
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
	cmd.Flags().StringVar(&listReq.Space, "space", listReq.Space, `Filter apps by app space name.`)

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

// start list-spaces command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listSpacesOverrides []func(
	*cobra.Command,
	*apps.ListSpacesRequest,
)

func newListSpaces() *cobra.Command {
	cmd := &cobra.Command{}

	var listSpacesReq apps.ListSpacesRequest

	cmd.Flags().IntVar(&listSpacesReq.PageSize, "page-size", listSpacesReq.PageSize, `Upper bound for items returned.`)
	cmd.Flags().StringVar(&listSpacesReq.PageToken, "page-token", listSpacesReq.PageToken, `Pagination token to go to the next page of app spaces.`)

	cmd.Use = "list-spaces"
	cmd.Short = `List app spaces.`
	cmd.Long = `List app spaces.

  Lists all app spaces in the workspace.`

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

		response := w.Apps.ListSpaces(ctx, listSpacesReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listSpacesOverrides {
		fn(cmd, &listSpacesReq)
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
				err := cmdio.RenderDiagnostics(ctx, diags)
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
	cmd.Flags().Var(&updateReq.App.ComputeSize, "compute-size", `Supported values: [LARGE, MEDIUM]`)
	// TODO: complex arg: compute_status
	cmd.Flags().StringVar(&updateReq.App.Description, "description", updateReq.App.Description, `The description of the app.`)
	// TODO: array: effective_user_api_scopes
	// TODO: complex arg: git_repository
	// TODO: complex arg: pending_deployment
	// TODO: array: resources
	cmd.Flags().StringVar(&updateReq.App.Space, "space", updateReq.App.Space, `Name of the space this app belongs to.`)
	cmd.Flags().StringVar(&updateReq.App.UsagePolicyId, "usage-policy-id", updateReq.App.UsagePolicyId, ``)
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
				err := cmdio.RenderDiagnostics(ctx, diags)
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
				err := cmdio.RenderDiagnostics(ctx, diags)
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

// start update-space command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateSpaceOverrides []func(
	*cobra.Command,
	*apps.UpdateSpaceRequest,
)

func newUpdateSpace() *cobra.Command {
	cmd := &cobra.Command{}

	var updateSpaceReq apps.UpdateSpaceRequest
	updateSpaceReq.Space = apps.Space{}
	var updateSpaceJson flags.JsonFlag

	var updateSpaceSkipWait bool
	var updateSpaceTimeout time.Duration

	cmd.Flags().BoolVar(&updateSpaceSkipWait, "no-wait", updateSpaceSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&updateSpaceTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&updateSpaceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateSpaceReq.Space.Description, "description", updateSpaceReq.Space.Description, `The description of the app space.`)
	// TODO: array: effective_user_api_scopes
	// TODO: array: resources
	// TODO: complex arg: status
	cmd.Flags().StringVar(&updateSpaceReq.Space.UsagePolicyId, "usage-policy-id", updateSpaceReq.Space.UsagePolicyId, `The usage policy ID for managing cost at the space level.`)
	// TODO: array: user_api_scopes

	cmd.Use = "update-space NAME UPDATE_MASK"
	cmd.Short = `Update an app space.`
	cmd.Long = `Update an app space.

  Updates an app space. The update process is asynchronous and the status of the
  update can be checked with the GetSpaceOperation method.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-space-operation command.

  Arguments:
    NAME: The name of the app space. The name must contain only lowercase
      alphanumeric characters and hyphens. It must be unique within the
      workspace.
    UPDATE_MASK: The field mask must be a single string, with multiple fields separated by
      commas (no spaces). The field path is relative to the resource object,
      using a dot (.) to navigate sub-fields (e.g., author.given_name).
      Specification of elements in sequence or map fields is not allowed, as
      only the entire collection field can be specified. Field names must
      exactly match the resource field names.

      A field mask of * indicates full replacement. It’s recommended to
      always explicitly list the fields being updated and avoid using *
      wildcards, as it can lead to unintended results if the API changes in the
      future.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateSpaceJson.Unmarshal(&updateSpaceReq.Space)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		}
		updateSpaceReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateSpaceReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}

		// Determine which mode to execute based on flags.
		switch {
		case updateSpaceSkipWait:
			wait, err := w.Apps.UpdateSpace(ctx, updateSpaceReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			operation, err := w.Apps.GetSpaceOperation(ctx, apps.GetOperationRequest{
				Name: wait.Name(),
			})
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, operation)

		default:
			wait, err := w.Apps.UpdateSpace(ctx, updateSpaceReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			spinner := cmdio.Spinner(ctx)
			spinner <- "Waiting for update-space to complete..."

			// Wait for completion.
			opts := api.WithTimeout(updateSpaceTimeout)
			response, err := wait.Wait(ctx, opts)
			if err != nil {
				return err
			}
			close(spinner)
			return cmdio.Render(ctx, response)
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateSpaceOverrides {
		fn(cmd, &updateSpaceReq)
	}

	return cmd
}

// end service Apps
