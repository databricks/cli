// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package apps

import (
	"errors"
	"fmt"
	"time"

	"maps"
	"net/url"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/options/lro"
	"github.com/spf13/cobra"

	accessmanagement "github.com/databricks/cli/libs/sdktypes/accessmanagement/v1"
	apps "github.com/databricks/cli/libs/sdktypes/apps/v1"
	sdkaccessmanagement "github.com/databricks/sdk-go/accessmanagement/v1"
	sdkapps "github.com/databricks/sdk-go/apps/v1"
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

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newCreateSpace())
	cmd.AddCommand(newCreateUpdate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newDeleteAppThumbnail())
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
	cmd.AddCommand(newUpdateAppThumbnail())
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
	createReq.App = &apps.App{}

	var createJson flags.JsonFlag

	var createSkipWait bool
	var createTimeout time.Duration

	cmd.Flags().BoolVar(&createSkipWait, "no-wait", createSkipWait, `do not wait to reach ACTIVE state`)
	cmd.Flags().DurationVar(&createTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach ACTIVE state`)

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.BoolPtrVar(cmd.Flags(), &createReq.NoCompute, "no-compute", `If true, the app will not be started after creation.`)
	// TODO: complex arg: active_deployment
	// TODO: complex arg: app_status
	flags.StringPtrVar(cmd.Flags(), &createReq.App.BudgetPolicyId, "budget-policy-id", ``)
	flags.IntPtrVar(cmd.Flags(), &createReq.App.ComputeMaxInstances, "compute-max-instances", `Maximum number of app instances the app is configured to run.`)
	flags.IntPtrVar(cmd.Flags(), &createReq.App.ComputeMinInstances, "compute-min-instances", `Minimum number of app instances the app is configured to run.`)
	flags.EnumVar(cmd.Flags(), &createReq.App.ComputeSize, "compute-size", `Supported values: [LARGE, MEDIUM, XLARGE]`, "ComputeSize", "LARGE", "MEDIUM", "XLARGE")
	// TODO: complex arg: compute_status
	// TODO: complex arg: default_git_source
	flags.StringPtrVar(cmd.Flags(), &createReq.App.Description, "description", `The description of the app.`)
	// TODO: array: effective_user_api_scopes
	flags.BoolPtrVar(cmd.Flags(), &createReq.App.ForwardUserAccessToken, "forward-user-access-token", `Forward the user's access token to the app.`)
	// TODO: complex arg: git_repository
	// TODO: complex arg: git_source
	// TODO: complex arg: pending_deployment
	// TODO: array: resources
	flags.StringPtrVar(cmd.Flags(), &createReq.App.SourceCodePath, "source-code-path", ``)
	flags.StringPtrVar(cmd.Flags(), &createReq.App.Space, "space", `Name of the space this app belongs to.`)
	// TODO: array: telemetry_export_destinations
	flags.StringPtrVar(cmd.Flags(), &createReq.App.UsagePolicyId, "usage-policy-id", ``)
	// TODO: array: user_api_scopes

	cmd.Use = "create NAME"
	cmd.Short = `Create an app.`
	cmd.Long = `Create an app.

  Creates a new app.

  Arguments:
    NAME: The name of the app. The name must contain only lowercase alphanumeric
      characters and hyphens. It must be unique within the workspace.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name' in your JSON input")
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
		_ = w

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
			createReq.App.Name = new(args[0])
		}

		sdkReq, err := apps.CreateAppRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		if createSkipWait {
			api, err := apiclient.New(w.Config)
			if err != nil {
				return err
			}
			var current apps.App
			err = api.Do(ctx, "POST", "/api/2.0/apps", auth.WorkspaceIDHeaders(w.Config), map[string]any{"no_compute": createReq.NoCompute}, createReq.App, &current)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, current)
		}
		wait, err := w.Apps.CreateApp(ctx, *sdkReq)
		if err != nil {
			return err
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for create to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(createTimeout))
		sp.Close()
		if err != nil {
			return &cmdgen.WaitError{Err: err}
		}

		out, err := apps.AppFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	createSpaceReq.Space = &apps.Space{}

	var createSpaceJson flags.JsonFlag

	var createSpaceSkipWait bool
	var createSpaceTimeout time.Duration

	cmd.Flags().BoolVar(&createSpaceSkipWait, "no-wait", createSpaceSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&createSpaceTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&createSpaceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createSpaceReq.Space.AssumeGroupId, "assume-group-id", `The group whose permissions users assume via Role Authorization for apps in this space.`)
	flags.StringPtrVar(cmd.Flags(), &createSpaceReq.Space.Description, "description", `The description of the app space.`)
	// TODO: array: effective_user_api_scopes
	// TODO: array: resources
	// TODO: complex arg: status
	flags.StringPtrVar(cmd.Flags(), &createSpaceReq.Space.UsagePolicyId, "usage-policy-id", `The usage policy ID for managing cost at the space level.`)
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
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name' in your JSON input")
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
		_ = w

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
			createSpaceReq.Space.Name = new(args[0])
		}

		sdkReq, err := apps.CreateSpaceRequestToSDK(&createSpaceReq)
		if err != nil {
			return err
		}
		// Determine which mode to execute based on flags.
		switch {
		case createSpaceSkipWait:
			wait, err := w.Apps.CreateSpace(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Apps.CreateSpace(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for create-space to complete...")

			// Wait for completion.
			response, err := wait.Wait(ctx, lro.WithTimeout(createSpaceTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			out, err := apps.SpaceFromSDK(response)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, out)
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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only APP_NAME as positional arguments. Provide 'update_mask' in your JSON input")
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
		_ = w

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
		createUpdateReq.AppName = new(args[0])
		if !cmd.Flags().Changed("json") {
			createUpdateReq.UpdateMask = new(args[1])
		}

		sdkReq, err := apps.AsyncUpdateAppRequestToSDK(&createUpdateReq)
		if err != nil {
			return err
		}
		if createUpdateSkipWait {
			api, err := apiclient.New(w.Config)
			if err != nil {
				return err
			}
			var current apps.AppUpdate
			err = api.Do(ctx, "POST", fmt.Sprintf("/api/2.0/apps/%s/update", url.PathEscape(fmt.Sprint(utils.Deref(createUpdateReq.AppName)))), auth.WorkspaceIDHeaders(w.Config), nil, &createUpdateReq, &current)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, current)
		}
		wait, err := w.Apps.AsyncUpdateApp(ctx, *sdkReq)
		if err != nil {
			return err
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for create-update to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(createUpdateTimeout))
		sp.Close()
		if err != nil {
			return &cmdgen.WaitError{Err: err}
		}

		out, err := apps.AppUpdateFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkapps.DeleteAppRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq sdkapps.DeleteAppRequest

	cmd.Use = "delete NAME"
	cmd.Short = `Delete an app.`
	cmd.Long = `Delete an app.

  Deletes an app.

  Arguments:
    NAME: The name of the app.`

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

		deleteReq.Name = new(args[0])

		sdkReq := &deleteReq
		response, err := w.Apps.DeleteApp(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := apps.AppFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start delete-app-thumbnail command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteAppThumbnailOverrides []func(
	*cobra.Command,
	*sdkapps.DeleteAppThumbnailRequest,
)

func newDeleteAppThumbnail() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteAppThumbnailReq sdkapps.DeleteAppThumbnailRequest

	cmd.Use = "delete-app-thumbnail NAME"
	cmd.Short = `Delete an app thumbnail.`
	cmd.Long = `Delete an app thumbnail.

  Deletes the thumbnail for an app.

  Arguments:
    NAME: The name of the app.`

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

		deleteAppThumbnailReq.Name = new(args[0])

		sdkReq := &deleteAppThumbnailReq
		err = w.Apps.DeleteAppThumbnail(ctx, *sdkReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteAppThumbnailOverrides {
		fn(cmd, &deleteAppThumbnailReq)
	}

	return cmd
}

// start delete-space command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteSpaceOverrides []func(
	*cobra.Command,
	*sdkapps.DeleteSpaceRequest,
)

func newDeleteSpace() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteSpaceReq sdkapps.DeleteSpaceRequest

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
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteSpaceReq.Name = new(args[0])

		sdkReq := &deleteSpaceReq
		// Determine which mode to execute based on flags.
		switch {
		case deleteSpaceSkipWait:
			wait, err := w.Apps.DeleteSpace(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Apps.DeleteSpace(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for delete-space to complete...")

			// Wait for completion.
			response := struct{}{}
			err = wait.Wait(ctx, lro.WithTimeout(deleteSpaceTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			_ = response
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
	deployReq.AppDeployment = &apps.AppDeployment{}

	var deployJson flags.JsonFlag

	var deploySkipWait bool
	var deployTimeout time.Duration

	cmd.Flags().BoolVar(&deploySkipWait, "no-wait", deploySkipWait, `do not wait to reach SUCCEEDED state`)
	cmd.Flags().DurationVar(&deployTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach SUCCEEDED state`)

	cmd.Flags().Var(&deployJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: command
	// TODO: complex arg: deployment_artifacts
	flags.StringPtrVar(cmd.Flags(), &deployReq.AppDeployment.DeploymentId, "deployment-id", `The unique id of the deployment.`)
	// TODO: array: env_vars
	// TODO: complex arg: git_source
	// TODO: complex arg: health_check
	flags.EnumVar(cmd.Flags(), &deployReq.AppDeployment.Mode, "mode", `The mode of which the deployment will manage the source code. Supported values: [AUTO_SYNC, SNAPSHOT]`, "AppDeploymentMode", "AUTO_SYNC", "SNAPSHOT")
	flags.StringPtrVar(cmd.Flags(), &deployReq.AppDeployment.SourceCodePath, "source-code-path", `The workspace file system path of the source code used to create the app deployment.`)
	// TODO: complex arg: status

	cmd.Use = "deploy APP_NAME"
	cmd.Short = `Create an app deployment.`
	cmd.Long = `Create an app deployment.

  Creates an app deployment for the app with the supplied name.

  Arguments:
    APP_NAME: The name of the app.`

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
		deployReq.AppName = new(args[0])

		sdkReq, err := apps.CreateAppDeploymentRequestToSDK(&deployReq)
		if err != nil {
			return err
		}
		if deploySkipWait {
			api, err := apiclient.New(w.Config)
			if err != nil {
				return err
			}
			var current apps.AppDeployment
			err = api.Do(ctx, "POST", fmt.Sprintf("/api/2.0/apps/%s/deployments", url.PathEscape(fmt.Sprint(utils.Deref(deployReq.AppName)))), auth.WorkspaceIDHeaders(w.Config), nil, deployReq.AppDeployment, &current)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, current)
		}
		wait, err := w.Apps.CreateAppDeployment(ctx, *sdkReq)
		if err != nil {
			return err
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for deploy to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(deployTimeout))
		sp.Close()
		if err != nil {
			return &cmdgen.WaitError{Err: err}
		}

		out, err := apps.AppDeploymentFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkapps.GetAppRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}
	var getReq sdkapps.GetAppRequest

	cmd.Use = "get NAME"
	cmd.Short = `Get an app.`
	cmd.Long = `Get an app.

  Retrieves information for the app with the supplied name.

  Arguments:
    NAME: The name of the app.`

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

		getReq.Name = new(args[0])

		sdkReq := &getReq
		response, err := w.Apps.GetApp(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := apps.AppFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkapps.GetAppDeploymentRequest,
)

func newGetDeployment() *cobra.Command {
	cmd := &cobra.Command{}
	var getDeploymentReq sdkapps.GetAppDeploymentRequest

	cmd.Use = "get-deployment APP_NAME DEPLOYMENT_ID"
	cmd.Short = `Get an app deployment.`
	cmd.Long = `Get an app deployment.

  Retrieves information for the app deployment with the supplied name and
  deployment id.

  Arguments:
    APP_NAME: The name of the app.
    DEPLOYMENT_ID: The unique id of the deployment.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getDeploymentReq.AppName = new(args[0])
		getDeploymentReq.DeploymentId = new(args[1])

		sdkReq := &getDeploymentReq
		response, err := w.Apps.GetAppDeployment(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := apps.AppDeploymentFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkaccessmanagement.ListPermissionLevelsRequest,
)

func newGetPermissionLevels() *cobra.Command {
	cmd := &cobra.Command{}
	var getPermissionLevelsReq sdkaccessmanagement.ListPermissionLevelsRequest

	cmd.Use = "get-permission-levels APP_NAME"
	cmd.Short = `Get app permission levels.`
	cmd.Long = `Get app permission levels.

  Gets the permission levels that a user can have on an object.

  Arguments:
    APP_NAME: The app for which to get or manage permissions.`

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

		getPermissionLevelsReq.RequestObjectId = new(args[0])

		getPermissionLevelsReq.RequestObjectType = new("apps")

		sdkReq := &getPermissionLevelsReq
		response, err := w.AccessManagement.ListPermissionLevels(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.ListPermissionLevelsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkaccessmanagement.GetObjectPermissionsRequest,
)

func newGetPermissions() *cobra.Command {
	cmd := &cobra.Command{}
	var getPermissionsReq sdkaccessmanagement.GetObjectPermissionsRequest

	cmd.Use = "get-permissions APP_NAME"
	cmd.Short = `Get app permissions.`
	cmd.Long = `Get app permissions.

  Gets the permissions of an app. Apps can inherit permissions from their root
  object.

  Arguments:
    APP_NAME: The app for which to get or manage permissions.`

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

		getPermissionsReq.RequestObjectId = new(args[0])

		getPermissionsReq.RequestObjectType = new("apps")

		sdkReq := &getPermissionsReq
		response, err := w.AccessManagement.GetObjectPermissions(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.PermissionsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkapps.GetSpaceRequest,
)

func newGetSpace() *cobra.Command {
	cmd := &cobra.Command{}
	var getSpaceReq sdkapps.GetSpaceRequest

	cmd.Use = "get-space NAME"
	cmd.Short = `Get an app space.`
	cmd.Long = `Get an app space.

  Retrieves information for the app space with the supplied name.

  Arguments:
    NAME: The name of the app space.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getSpaceReq.Name = new(args[0])

		sdkReq := &getSpaceReq
		response, err := w.Apps.GetSpace(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := apps.SpaceFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*map[string]any,
)

func newGetSpaceOperation() *cobra.Command {
	cmd := &cobra.Command{}
	var getSpaceOperationReq map[string]any
	getSpaceOperationReq = map[string]any{}
	var getSpaceOperationNameRaw string

	cmd.Use = "get-space-operation NAME"
	cmd.Short = `Get the status of an app space operation.`
	cmd.Long = `Get the status of an app space operation.

  Gets the status of an app space update operation.

  Arguments:
    NAME: The name of the operation resource.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getSpaceOperationNameRaw = args[0]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/app-spaces/%s/operation", fmt.Sprint(getSpaceOperationNameRaw))

		maps.Copy(rawQuery, getSpaceOperationReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
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
	*sdkapps.GetAppUpdateRequest,
)

func newGetUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var getUpdateReq sdkapps.GetAppUpdateRequest

	cmd.Use = "get-update APP_NAME"
	cmd.Short = `Get an app update.`
	cmd.Long = `Get an app update.

  Gets the status of an app update.

  Arguments:
    APP_NAME: The name of the app.`

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

		getUpdateReq.AppName = new(args[0])

		sdkReq := &getUpdateReq
		response, err := w.Apps.GetAppUpdate(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := apps.AppUpdateFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	flags.IntPtrVar(cmd.Flags(), &listReq.PageSize, "page-size", `Upper bound for items returned.`)
	flags.StringPtrVar(cmd.Flags(), &listReq.Space, "space", `Filter apps by app space name.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list"
	cmd.Short = `List apps.`
	cmd.Long = `List apps.

  Lists all apps in the workspace.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := apps.ListAppsRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.Apps.ListAppsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, apps.AppFromSDK))
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

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listDeploymentsLimit int

	flags.IntPtrVar(cmd.Flags(), &listDeploymentsReq.PageSize, "page-size", `Upper bound for items returned.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listDeploymentsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listDeploymentsReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-deployments APP_NAME"
	cmd.Short = `List app deployments.`
	cmd.Long = `List app deployments.

  Lists all app deployments for the app with the supplied name.

  Arguments:
    APP_NAME: The name of the app.`

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

		listDeploymentsReq.AppName = new(args[0])

		sdkReq, err := apps.ListAppDeploymentsRequestToSDK(&listDeploymentsReq)
		if err != nil {
			return err
		}
		if listDeploymentsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listDeploymentsLimit)
		}
		if listDeploymentsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listDeploymentsLimit)
		}
		response := w.Apps.ListAppDeploymentsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, apps.AppDeploymentFromSDK))
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

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listSpacesLimit int

	flags.IntPtrVar(cmd.Flags(), &listSpacesReq.PageSize, "page-size", `Upper bound for items returned.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listSpacesLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listSpacesReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-spaces"
	cmd.Short = `List app spaces.`
	cmd.Long = `List app spaces.

  Lists all app spaces in the workspace.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := apps.ListSpacesRequestToSDK(&listSpacesReq)
		if err != nil {
			return err
		}
		if listSpacesLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listSpacesLimit)
		}
		if listSpacesLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listSpacesLimit)
		}
		response := w.Apps.ListSpacesIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, apps.SpaceFromSDK))
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
	*accessmanagement.SetObjectPermissionsRequest,
)

func newSetPermissions() *cobra.Command {
	cmd := &cobra.Command{}
	var setPermissionsReq accessmanagement.SetObjectPermissionsRequest

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
		setPermissionsReq.RequestObjectId = new(args[0])

		setPermissionsReq.RequestObjectType = new("apps")

		sdkReq, err := accessmanagement.SetObjectPermissionsRequestToSDK(&setPermissionsReq)
		if err != nil {
			return err
		}
		response, err := w.AccessManagement.SetObjectPermissions(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.PermissionsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

		startReq.Name = new(args[0])

		sdkReq, err := apps.StartAppRequestToSDK(&startReq)
		if err != nil {
			return err
		}
		if startSkipWait {
			api, err := apiclient.New(w.Config)
			if err != nil {
				return err
			}
			var current apps.App
			err = api.Do(ctx, "POST", fmt.Sprintf("/api/2.0/apps/%s/start", url.PathEscape(fmt.Sprint(utils.Deref(startReq.Name)))), auth.WorkspaceIDHeaders(w.Config), nil, &startReq, &current)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, current)
		}
		wait, err := w.Apps.StartApp(ctx, *sdkReq)
		if err != nil {
			return err
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for start to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(startTimeout))
		sp.Close()
		if err != nil {
			return &cmdgen.WaitError{Err: err}
		}

		out, err := apps.AppFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

		stopReq.Name = new(args[0])

		sdkReq, err := apps.StopAppRequestToSDK(&stopReq)
		if err != nil {
			return err
		}
		if stopSkipWait {
			api, err := apiclient.New(w.Config)
			if err != nil {
				return err
			}
			var current apps.App
			err = api.Do(ctx, "POST", fmt.Sprintf("/api/2.0/apps/%s/stop", url.PathEscape(fmt.Sprint(utils.Deref(stopReq.Name)))), auth.WorkspaceIDHeaders(w.Config), nil, &stopReq, &current)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, current)
		}
		wait, err := w.Apps.StopApp(ctx, *sdkReq)
		if err != nil {
			return err
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for stop to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(stopTimeout))
		sp.Close()
		if err != nil {
			return &cmdgen.WaitError{Err: err}
		}

		out, err := apps.AppFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	updateReq.App = &apps.App{}

	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: active_deployment
	// TODO: complex arg: app_status
	flags.StringPtrVar(cmd.Flags(), &updateReq.App.BudgetPolicyId, "budget-policy-id", ``)
	flags.IntPtrVar(cmd.Flags(), &updateReq.App.ComputeMaxInstances, "compute-max-instances", `Maximum number of app instances the app is configured to run.`)
	flags.IntPtrVar(cmd.Flags(), &updateReq.App.ComputeMinInstances, "compute-min-instances", `Minimum number of app instances the app is configured to run.`)
	flags.EnumVar(cmd.Flags(), &updateReq.App.ComputeSize, "compute-size", `Supported values: [LARGE, MEDIUM, XLARGE]`, "ComputeSize", "LARGE", "MEDIUM", "XLARGE")
	// TODO: complex arg: compute_status
	// TODO: complex arg: default_git_source
	flags.StringPtrVar(cmd.Flags(), &updateReq.App.Description, "description", `The description of the app.`)
	// TODO: array: effective_user_api_scopes
	flags.BoolPtrVar(cmd.Flags(), &updateReq.App.ForwardUserAccessToken, "forward-user-access-token", `Forward the user's access token to the app.`)
	// TODO: complex arg: git_repository
	// TODO: complex arg: git_source
	// TODO: complex arg: pending_deployment
	// TODO: array: resources
	flags.StringPtrVar(cmd.Flags(), &updateReq.App.SourceCodePath, "source-code-path", ``)
	flags.StringPtrVar(cmd.Flags(), &updateReq.App.Space, "space", `Name of the space this app belongs to.`)
	// TODO: array: telemetry_export_destinations
	flags.StringPtrVar(cmd.Flags(), &updateReq.App.UsagePolicyId, "usage-policy-id", ``)
	// TODO: array: user_api_scopes

	cmd.Use = "update NAME"
	cmd.Short = `Update an app.`
	cmd.Long = `Update an app.

  Updates the app with the supplied name. This is a full replacement: fields
  omitted from the request are cleared, so send the complete app.

  Arguments:
    NAME: The name of the app. The name must contain only lowercase alphanumeric
      characters and hyphens. It must be unique within the workspace.`

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
		updateReq.App.Name = new(args[0])

		sdkReq, err := apps.UpdateAppRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := w.Apps.UpdateApp(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := apps.AppFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start update-app-thumbnail command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateAppThumbnailOverrides []func(
	*cobra.Command,
	*apps.UpdateAppThumbnailRequest,
)

func newUpdateAppThumbnail() *cobra.Command {
	cmd := &cobra.Command{}
	var updateAppThumbnailReq apps.UpdateAppThumbnailRequest

	var updateAppThumbnailJson flags.JsonFlag

	cmd.Flags().Var(&updateAppThumbnailJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: app_thumbnail

	cmd.Use = "update-app-thumbnail NAME"
	cmd.Short = `Update an app thumbnail.`
	cmd.Long = `Update an app thumbnail.

  Updates the thumbnail for an app.

  Arguments:
    NAME: The name of the app.`

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

		if cmd.Flags().Changed("json") {
			diags := updateAppThumbnailJson.Unmarshal(&updateAppThumbnailReq)
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
		updateAppThumbnailReq.Name = new(args[0])

		sdkReq, err := apps.UpdateAppThumbnailRequestToSDK(&updateAppThumbnailReq)
		if err != nil {
			return err
		}
		response, err := w.Apps.UpdateAppThumbnail(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := apps.AppThumbnailFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateAppThumbnailOverrides {
		fn(cmd, &updateAppThumbnailReq)
	}

	return cmd
}

// start update-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updatePermissionsOverrides []func(
	*cobra.Command,
	*accessmanagement.UpdateObjectPermissionsRequest,
)

func newUpdatePermissions() *cobra.Command {
	cmd := &cobra.Command{}
	var updatePermissionsReq accessmanagement.UpdateObjectPermissionsRequest

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
		updatePermissionsReq.RequestObjectId = new(args[0])

		updatePermissionsReq.RequestObjectType = new("apps")

		sdkReq, err := accessmanagement.UpdateObjectPermissionsRequestToSDK(&updatePermissionsReq)
		if err != nil {
			return err
		}
		response, err := w.AccessManagement.UpdateObjectPermissions(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.PermissionsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	updateSpaceReq.Space = &apps.Space{}

	var updateSpaceJson flags.JsonFlag

	var updateSpaceSkipWait bool
	var updateSpaceTimeout time.Duration

	cmd.Flags().BoolVar(&updateSpaceSkipWait, "no-wait", updateSpaceSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&updateSpaceTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&updateSpaceJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateSpaceReq.Space.AssumeGroupId, "assume-group-id", `The group whose permissions users assume via Role Authorization for apps in this space.`)
	flags.StringPtrVar(cmd.Flags(), &updateSpaceReq.Space.Description, "description", `The description of the app space.`)
	// TODO: array: effective_user_api_scopes
	// TODO: array: resources
	// TODO: complex arg: status
	flags.StringPtrVar(cmd.Flags(), &updateSpaceReq.Space.UsagePolicyId, "usage-policy-id", `The usage policy ID for managing cost at the space level.`)
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
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

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
		updateSpaceReq.Space.Name = new(args[0])
		if args[1] != "" {
			updateSpaceReq.UpdateMask = new(args[1])
		}

		sdkReq, err := apps.UpdateSpaceRequestToSDK(&updateSpaceReq)
		if err != nil {
			return err
		}
		// Determine which mode to execute based on flags.
		switch {
		case updateSpaceSkipWait:
			wait, err := w.Apps.UpdateSpace(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			return cmdio.Render(ctx, map[string]any{"name": wait.Name()})

		default:
			wait, err := w.Apps.UpdateSpace(ctx, *sdkReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for update-space to complete...")

			// Wait for completion.
			response, err := wait.Wait(ctx, lro.WithTimeout(updateSpaceTimeout))
			if err != nil {
				return &cmdgen.WaitError{Err: err}
			}
			sp.Close()

			out, err := apps.SpaceFromSDK(response)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, out)
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
