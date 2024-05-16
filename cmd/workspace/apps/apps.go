// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package apps

import (
	"fmt"
	"time"

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
		Short: `Apps run directly on a customer’s Databricks instance, integrate with their data, use and extend Databricks services, and enable users to interact through single sign-on.`,
		Long: `Apps run directly on a customer’s Databricks instance, integrate with their
  data, use and extend Databricks services, and enable users to interact through
  single sign-on.`,
		GroupID: "serving",
		Annotations: map[string]string{
			"package": "serving",
		},

		// This service is being previewed; hide from help output.
		Hidden: true,
	}

	// Add methods
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newCreateDeployment())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newGetDeployment())
	cmd.AddCommand(newGetEnvironment())
	cmd.AddCommand(newList())
	cmd.AddCommand(newListDeployments())
	cmd.AddCommand(newStop())
	cmd.AddCommand(newUpdate())

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
	*serving.CreateAppRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq serving.CreateAppRequest
	var createJson flags.JsonFlag

	var createSkipWait bool
	var createTimeout time.Duration

	cmd.Flags().BoolVar(&createSkipWait, "no-wait", createSkipWait, `do not wait to reach IDLE state`)
	cmd.Flags().DurationVar(&createTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach IDLE state`)
	// TODO: short flags
	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createReq.Description, "description", createReq.Description, `The description of the app.`)

	cmd.Use = "create NAME"
	cmd.Short = `Create an App.`
	cmd.Long = `Create an App.
  
  Creates a new app.

  Arguments:
    NAME: The name of the app. The name must contain only lowercase alphanumeric
      characters and hyphens and be between 2 and 30 characters long. It must be
      unique within the workspace.`

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
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		}
		if !cmd.Flags().Changed("json") {
			createReq.Name = args[0]
		}

		wait, err := w.Apps.Create(ctx, createReq)
		if err != nil {
			return err
		}
		if createSkipWait {
			return cmdio.Render(ctx, wait.Response)
		}
		spinner := cmdio.Spinner(ctx)
		info, err := wait.OnProgress(func(i *serving.App) {
			if i.Status == nil {
				return
			}
			status := i.Status.State
			statusMessage := fmt.Sprintf("current status: %s", status)
			if i.Status != nil {
				statusMessage = i.Status.Message
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

// start create-deployment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createDeploymentOverrides []func(
	*cobra.Command,
	*serving.CreateAppDeploymentRequest,
)

func newCreateDeployment() *cobra.Command {
	cmd := &cobra.Command{}

	var createDeploymentReq serving.CreateAppDeploymentRequest
	var createDeploymentJson flags.JsonFlag

	var createDeploymentSkipWait bool
	var createDeploymentTimeout time.Duration

	cmd.Flags().BoolVar(&createDeploymentSkipWait, "no-wait", createDeploymentSkipWait, `do not wait to reach SUCCEEDED state`)
	cmd.Flags().DurationVar(&createDeploymentTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach SUCCEEDED state`)
	// TODO: short flags
	cmd.Flags().Var(&createDeploymentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create-deployment APP_NAME SOURCE_CODE_PATH"
	cmd.Short = `Create an App Deployment.`
	cmd.Long = `Create an App Deployment.
  
  Creates an app deployment for the app with the supplied name.

  Arguments:
    APP_NAME: The name of the app.
    SOURCE_CODE_PATH: The source code path of the deployment.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only APP_NAME as positional arguments. Provide 'source_code_path' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = createDeploymentJson.Unmarshal(&createDeploymentReq)
			if err != nil {
				return err
			}
		}
		createDeploymentReq.AppName = args[0]
		if !cmd.Flags().Changed("json") {
			createDeploymentReq.SourceCodePath = args[1]
		}

		wait, err := w.Apps.CreateDeployment(ctx, createDeploymentReq)
		if err != nil {
			return err
		}
		if createDeploymentSkipWait {
			return cmdio.Render(ctx, wait.Response)
		}
		spinner := cmdio.Spinner(ctx)
		info, err := wait.OnProgress(func(i *serving.AppDeployment) {
			if i.Status == nil {
				return
			}
			status := i.Status.State
			statusMessage := fmt.Sprintf("current status: %s", status)
			if i.Status != nil {
				statusMessage = i.Status.Message
			}
			spinner <- statusMessage
		}).GetWithTimeout(createDeploymentTimeout)
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
	for _, fn := range createDeploymentOverrides {
		fn(cmd, &createDeploymentReq)
	}

	return cmd
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*serving.DeleteAppRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq serving.DeleteAppRequest

	// TODO: short flags

	cmd.Use = "delete NAME"
	cmd.Short = `Delete an App.`
	cmd.Long = `Delete an App.
  
  Deletes an app.

  Arguments:
    NAME: The name of the app.`

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
		w := root.WorkspaceClient(ctx)

		deleteReq.Name = args[0]

		err = w.Apps.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
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

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*serving.GetAppRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq serving.GetAppRequest

	// TODO: short flags

	cmd.Use = "get NAME"
	cmd.Short = `Get an App.`
	cmd.Long = `Get an App.
  
  Retrieves information for the app with the supplied name.

  Arguments:
    NAME: The name of the app.`

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
		w := root.WorkspaceClient(ctx)

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
	*serving.GetAppDeploymentRequest,
)

func newGetDeployment() *cobra.Command {
	cmd := &cobra.Command{}

	var getDeploymentReq serving.GetAppDeploymentRequest

	// TODO: short flags

	cmd.Use = "get-deployment APP_NAME DEPLOYMENT_ID"
	cmd.Short = `Get an App Deployment.`
	cmd.Long = `Get an App Deployment.
  
  Retrieves information for the app deployment with the supplied name and
  deployment id.

  Arguments:
    APP_NAME: The name of the app.
    DEPLOYMENT_ID: The unique id of the deployment.`

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
		w := root.WorkspaceClient(ctx)

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

// start get-environment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getEnvironmentOverrides []func(
	*cobra.Command,
	*serving.GetAppEnvironmentRequest,
)

func newGetEnvironment() *cobra.Command {
	cmd := &cobra.Command{}

	var getEnvironmentReq serving.GetAppEnvironmentRequest

	// TODO: short flags

	cmd.Use = "get-environment NAME"
	cmd.Short = `Get App Environment.`
	cmd.Long = `Get App Environment.
  
  Retrieves app environment.

  Arguments:
    NAME: The name of the app.`

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
		w := root.WorkspaceClient(ctx)

		getEnvironmentReq.Name = args[0]

		response, err := w.Apps.GetEnvironment(ctx, getEnvironmentReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getEnvironmentOverrides {
		fn(cmd, &getEnvironmentReq)
	}

	return cmd
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*serving.ListAppsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq serving.ListAppsRequest

	// TODO: short flags

	cmd.Flags().IntVar(&listReq.PageSize, "page-size", listReq.PageSize, `Upper bound for items returned.`)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Pagination token to go to the next page of apps.`)

	cmd.Use = "list"
	cmd.Short = `List Apps.`
	cmd.Long = `List Apps.
  
  Lists all apps in the workspace.`

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
		w := root.WorkspaceClient(ctx)

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
	*serving.ListAppDeploymentsRequest,
)

func newListDeployments() *cobra.Command {
	cmd := &cobra.Command{}

	var listDeploymentsReq serving.ListAppDeploymentsRequest

	// TODO: short flags

	cmd.Flags().IntVar(&listDeploymentsReq.PageSize, "page-size", listDeploymentsReq.PageSize, `Upper bound for items returned.`)
	cmd.Flags().StringVar(&listDeploymentsReq.PageToken, "page-token", listDeploymentsReq.PageToken, `Pagination token to go to the next page of apps.`)

	cmd.Use = "list-deployments APP_NAME"
	cmd.Short = `List App Deployments.`
	cmd.Long = `List App Deployments.
  
  Lists all app deployments for the app with the supplied name.

  Arguments:
    APP_NAME: The name of the app.`

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
		w := root.WorkspaceClient(ctx)

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

// start stop command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var stopOverrides []func(
	*cobra.Command,
	*serving.StopAppRequest,
)

func newStop() *cobra.Command {
	cmd := &cobra.Command{}

	var stopReq serving.StopAppRequest

	// TODO: short flags

	cmd.Use = "stop NAME"
	cmd.Short = `Stop an App.`
	cmd.Long = `Stop an App.
  
  Stops the active deployment of the app in the workspace.

  Arguments:
    NAME: The name of the app.`

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
		w := root.WorkspaceClient(ctx)

		stopReq.Name = args[0]

		err = w.Apps.Stop(ctx, stopReq)
		if err != nil {
			return err
		}
		return nil
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
	*serving.UpdateAppRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq serving.UpdateAppRequest
	var updateJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateReq.Description, "description", updateReq.Description, `The description of the app.`)

	cmd.Use = "update NAME"
	cmd.Short = `Update an App.`
	cmd.Long = `Update an App.
  
  Updates the app with the supplied name.

  Arguments:
    NAME: The name of the app. The name must contain only lowercase alphanumeric
      characters and hyphens and be between 2 and 30 characters long. It must be
      unique within the workspace.`

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
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
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

// end service Apps
