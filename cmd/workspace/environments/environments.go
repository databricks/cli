// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package environments

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
	"github.com/databricks/databricks-sdk-go/service/environments"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "environments",
		Short: `APIs to manage environment resources.`,
		Long: `APIs to manage environment resources.
  
  The Environments API provides management capabilities for different types of
  environments including workspace-level base environments that define the
  environment version and dependencies to be used in serverless notebooks and
  jobs.`,
		GroupID: "environments",
		RunE:    root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreateWorkspaceBaseEnvironment())
	cmd.AddCommand(newDeleteWorkspaceBaseEnvironment())
	cmd.AddCommand(newGetDefaultWorkspaceBaseEnvironment())
	cmd.AddCommand(newGetOperation())
	cmd.AddCommand(newGetWorkspaceBaseEnvironment())
	cmd.AddCommand(newListWorkspaceBaseEnvironments())
	cmd.AddCommand(newRefreshWorkspaceBaseEnvironment())
	cmd.AddCommand(newUpdateDefaultWorkspaceBaseEnvironment())
	cmd.AddCommand(newUpdateWorkspaceBaseEnvironment())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-workspace-base-environment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createWorkspaceBaseEnvironmentOverrides []func(
	*cobra.Command,
	*environments.CreateWorkspaceBaseEnvironmentRequest,
)

func newCreateWorkspaceBaseEnvironment() *cobra.Command {
	cmd := &cobra.Command{}

	var createWorkspaceBaseEnvironmentReq environments.CreateWorkspaceBaseEnvironmentRequest
	createWorkspaceBaseEnvironmentReq.WorkspaceBaseEnvironment = environments.WorkspaceBaseEnvironment{}
	var createWorkspaceBaseEnvironmentJson flags.JsonFlag

	var createWorkspaceBaseEnvironmentSkipWait bool
	var createWorkspaceBaseEnvironmentTimeout time.Duration

	cmd.Flags().BoolVar(&createWorkspaceBaseEnvironmentSkipWait, "no-wait", createWorkspaceBaseEnvironmentSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&createWorkspaceBaseEnvironmentTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&createWorkspaceBaseEnvironmentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createWorkspaceBaseEnvironmentReq.RequestId, "request-id", createWorkspaceBaseEnvironmentReq.RequestId, `A unique identifier for this request.`)
	cmd.Flags().StringVar(&createWorkspaceBaseEnvironmentReq.WorkspaceBaseEnvironmentId, "workspace-base-environment-id", createWorkspaceBaseEnvironmentReq.WorkspaceBaseEnvironmentId, `The ID to use for the workspace base environment, which will become the final component of the resource name.`)
	cmd.Flags().Var(&createWorkspaceBaseEnvironmentReq.WorkspaceBaseEnvironment.BaseEnvironmentType, "base-environment-type", `The type of base environment (CPU or GPU). Supported values: [CPU, GPU]`)
	cmd.Flags().StringVar(&createWorkspaceBaseEnvironmentReq.WorkspaceBaseEnvironment.Filepath, "filepath", createWorkspaceBaseEnvironmentReq.WorkspaceBaseEnvironment.Filepath, `The WSFS or UC Volumes path to the environment YAML file.`)
	cmd.Flags().StringVar(&createWorkspaceBaseEnvironmentReq.WorkspaceBaseEnvironment.Name, "name", createWorkspaceBaseEnvironmentReq.WorkspaceBaseEnvironment.Name, `The resource name of the workspace base environment.`)

	cmd.Use = "create-workspace-base-environment DISPLAY_NAME"
	cmd.Short = `Create a workspace base environment.`
	cmd.Long = `Create a workspace base environment.
  
  Creates a new WorkspaceBaseEnvironment. This is a long-running operation. The
  operation will asynchronously generate a materialized environment to optimize
  dependency resolution and is only marked as done when the materialized
  environment has been successfully generated or has failed.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    DISPLAY_NAME: Human-readable display name for the workspace base environment.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'display_name' in your JSON input")
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
			diags := createWorkspaceBaseEnvironmentJson.Unmarshal(&createWorkspaceBaseEnvironmentReq.WorkspaceBaseEnvironment)
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
			createWorkspaceBaseEnvironmentReq.WorkspaceBaseEnvironment.DisplayName = args[0]
		}

		// Determine which mode to execute based on flags.
		switch {
		case createWorkspaceBaseEnvironmentSkipWait:
			wait, err := w.Environments.CreateWorkspaceBaseEnvironment(ctx, createWorkspaceBaseEnvironmentReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			operation, err := w.Environments.GetOperation(ctx, environments.GetOperationRequest{
				Name: wait.Name(),
			})
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, operation)

		default:
			wait, err := w.Environments.CreateWorkspaceBaseEnvironment(ctx, createWorkspaceBaseEnvironmentReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for create-workspace-base-environment to complete...")

			// Wait for completion.
			opts := api.WithTimeout(createWorkspaceBaseEnvironmentTimeout)
			response, err := wait.Wait(ctx, opts)
			if err != nil {
				return err
			}
			sp.Close()
			return cmdio.Render(ctx, response)
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createWorkspaceBaseEnvironmentOverrides {
		fn(cmd, &createWorkspaceBaseEnvironmentReq)
	}

	return cmd
}

// start delete-workspace-base-environment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteWorkspaceBaseEnvironmentOverrides []func(
	*cobra.Command,
	*environments.DeleteWorkspaceBaseEnvironmentRequest,
)

func newDeleteWorkspaceBaseEnvironment() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteWorkspaceBaseEnvironmentReq environments.DeleteWorkspaceBaseEnvironmentRequest

	cmd.Use = "delete-workspace-base-environment NAME"
	cmd.Short = `Delete a workspace base environment.`
	cmd.Long = `Delete a workspace base environment.
  
  Deletes a WorkspaceBaseEnvironment. Deleting a base environment may impact
  linked notebooks and jobs. This operation is irreversible and should be
  performed only when you are certain the environment is no longer needed.

  Arguments:
    NAME: Required. The resource name of the workspace base environment to delete.
      Format: workspace-base-environments/{workspace_base_environment}`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteWorkspaceBaseEnvironmentReq.Name = args[0]

		err = w.Environments.DeleteWorkspaceBaseEnvironment(ctx, deleteWorkspaceBaseEnvironmentReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteWorkspaceBaseEnvironmentOverrides {
		fn(cmd, &deleteWorkspaceBaseEnvironmentReq)
	}

	return cmd
}

// start get-default-workspace-base-environment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getDefaultWorkspaceBaseEnvironmentOverrides []func(
	*cobra.Command,
	*environments.GetDefaultWorkspaceBaseEnvironmentRequest,
)

func newGetDefaultWorkspaceBaseEnvironment() *cobra.Command {
	cmd := &cobra.Command{}

	var getDefaultWorkspaceBaseEnvironmentReq environments.GetDefaultWorkspaceBaseEnvironmentRequest

	cmd.Use = "get-default-workspace-base-environment NAME"
	cmd.Short = `Get the default workspace base environment configuration.`
	cmd.Long = `Get the default workspace base environment configuration.
  
  Gets the default WorkspaceBaseEnvironment configuration for the workspace.
  Returns the current default base environment settings for both CPU and GPU
  compute.

  Arguments:
    NAME: A static resource name of the default workspace base environment. Format:
      default-workspace-base-environment`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getDefaultWorkspaceBaseEnvironmentReq.Name = args[0]

		response, err := w.Environments.GetDefaultWorkspaceBaseEnvironment(ctx, getDefaultWorkspaceBaseEnvironmentReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getDefaultWorkspaceBaseEnvironmentOverrides {
		fn(cmd, &getDefaultWorkspaceBaseEnvironmentReq)
	}

	return cmd
}

// start get-operation command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOperationOverrides []func(
	*cobra.Command,
	*environments.GetOperationRequest,
)

func newGetOperation() *cobra.Command {
	cmd := &cobra.Command{}

	var getOperationReq environments.GetOperationRequest

	cmd.Use = "get-operation NAME"
	cmd.Short = `Get the status of a long-running operation.`
	cmd.Long = `Get the status of a long-running operation.
  
  Gets the status of a long-running operation. Clients can use this method to
  poll the operation result.

  Arguments:
    NAME: The name of the operation resource.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getOperationReq.Name = args[0]

		response, err := w.Environments.GetOperation(ctx, getOperationReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getOperationOverrides {
		fn(cmd, &getOperationReq)
	}

	return cmd
}

// start get-workspace-base-environment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getWorkspaceBaseEnvironmentOverrides []func(
	*cobra.Command,
	*environments.GetWorkspaceBaseEnvironmentRequest,
)

func newGetWorkspaceBaseEnvironment() *cobra.Command {
	cmd := &cobra.Command{}

	var getWorkspaceBaseEnvironmentReq environments.GetWorkspaceBaseEnvironmentRequest

	cmd.Use = "get-workspace-base-environment NAME"
	cmd.Short = `Get a workspace base environment.`
	cmd.Long = `Get a workspace base environment.
  
  Retrieves a WorkspaceBaseEnvironment by its name.

  Arguments:
    NAME: Required. The resource name of the workspace base environment to retrieve.
      Format: workspace-base-environments/{workspace_base_environment}`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getWorkspaceBaseEnvironmentReq.Name = args[0]

		response, err := w.Environments.GetWorkspaceBaseEnvironment(ctx, getWorkspaceBaseEnvironmentReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getWorkspaceBaseEnvironmentOverrides {
		fn(cmd, &getWorkspaceBaseEnvironmentReq)
	}

	return cmd
}

// start list-workspace-base-environments command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listWorkspaceBaseEnvironmentsOverrides []func(
	*cobra.Command,
	*environments.ListWorkspaceBaseEnvironmentsRequest,
)

func newListWorkspaceBaseEnvironments() *cobra.Command {
	cmd := &cobra.Command{}

	var listWorkspaceBaseEnvironmentsReq environments.ListWorkspaceBaseEnvironmentsRequest

	cmd.Flags().IntVar(&listWorkspaceBaseEnvironmentsReq.PageSize, "page-size", listWorkspaceBaseEnvironmentsReq.PageSize, `The maximum number of environments to return per page.`)
	cmd.Flags().StringVar(&listWorkspaceBaseEnvironmentsReq.PageToken, "page-token", listWorkspaceBaseEnvironmentsReq.PageToken, `Page token for pagination.`)

	cmd.Use = "list-workspace-base-environments"
	cmd.Short = `List workspace base environments.`
	cmd.Long = `List workspace base environments.
  
  Lists all WorkspaceBaseEnvironments in the workspace.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.Environments.ListWorkspaceBaseEnvironments(ctx, listWorkspaceBaseEnvironmentsReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listWorkspaceBaseEnvironmentsOverrides {
		fn(cmd, &listWorkspaceBaseEnvironmentsReq)
	}

	return cmd
}

// start refresh-workspace-base-environment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var refreshWorkspaceBaseEnvironmentOverrides []func(
	*cobra.Command,
	*environments.RefreshWorkspaceBaseEnvironmentRequest,
)

func newRefreshWorkspaceBaseEnvironment() *cobra.Command {
	cmd := &cobra.Command{}

	var refreshWorkspaceBaseEnvironmentReq environments.RefreshWorkspaceBaseEnvironmentRequest

	var refreshWorkspaceBaseEnvironmentSkipWait bool
	var refreshWorkspaceBaseEnvironmentTimeout time.Duration

	cmd.Flags().BoolVar(&refreshWorkspaceBaseEnvironmentSkipWait, "no-wait", refreshWorkspaceBaseEnvironmentSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&refreshWorkspaceBaseEnvironmentTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Use = "refresh-workspace-base-environment NAME"
	cmd.Short = `Refresh materialized workspace base environment.`
	cmd.Long = `Refresh materialized workspace base environment.
  
  Refreshes the materialized environment for a WorkspaceBaseEnvironment. This is
  a long-running operation. The operation will asynchronously regenerate the
  materialized environment and is only marked as done when the materialized
  environment has been successfully generated or has failed. The existing
  materialized environment remains available until it expires.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: Required. The resource name of the workspace base environment to delete.
      Format: workspace-base-environments/{workspace_base_environment}`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		refreshWorkspaceBaseEnvironmentReq.Name = args[0]

		// Determine which mode to execute based on flags.
		switch {
		case refreshWorkspaceBaseEnvironmentSkipWait:
			wait, err := w.Environments.RefreshWorkspaceBaseEnvironment(ctx, refreshWorkspaceBaseEnvironmentReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			operation, err := w.Environments.GetOperation(ctx, environments.GetOperationRequest{
				Name: wait.Name(),
			})
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, operation)

		default:
			wait, err := w.Environments.RefreshWorkspaceBaseEnvironment(ctx, refreshWorkspaceBaseEnvironmentReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for refresh-workspace-base-environment to complete...")

			// Wait for completion.
			opts := api.WithTimeout(refreshWorkspaceBaseEnvironmentTimeout)
			response, err := wait.Wait(ctx, opts)
			if err != nil {
				return err
			}
			sp.Close()
			return cmdio.Render(ctx, response)
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range refreshWorkspaceBaseEnvironmentOverrides {
		fn(cmd, &refreshWorkspaceBaseEnvironmentReq)
	}

	return cmd
}

// start update-default-workspace-base-environment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateDefaultWorkspaceBaseEnvironmentOverrides []func(
	*cobra.Command,
	*environments.UpdateDefaultWorkspaceBaseEnvironmentRequest,
)

func newUpdateDefaultWorkspaceBaseEnvironment() *cobra.Command {
	cmd := &cobra.Command{}

	var updateDefaultWorkspaceBaseEnvironmentReq environments.UpdateDefaultWorkspaceBaseEnvironmentRequest
	updateDefaultWorkspaceBaseEnvironmentReq.DefaultWorkspaceBaseEnvironment = environments.DefaultWorkspaceBaseEnvironment{}
	var updateDefaultWorkspaceBaseEnvironmentJson flags.JsonFlag

	cmd.Flags().Var(&updateDefaultWorkspaceBaseEnvironmentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateDefaultWorkspaceBaseEnvironmentReq.DefaultWorkspaceBaseEnvironment.CpuWorkspaceBaseEnvironment, "cpu-workspace-base-environment", updateDefaultWorkspaceBaseEnvironmentReq.DefaultWorkspaceBaseEnvironment.CpuWorkspaceBaseEnvironment, `The default workspace base environment for CPU compute.`)
	cmd.Flags().StringVar(&updateDefaultWorkspaceBaseEnvironmentReq.DefaultWorkspaceBaseEnvironment.GpuWorkspaceBaseEnvironment, "gpu-workspace-base-environment", updateDefaultWorkspaceBaseEnvironmentReq.DefaultWorkspaceBaseEnvironment.GpuWorkspaceBaseEnvironment, `The default workspace base environment for GPU compute.`)
	cmd.Flags().StringVar(&updateDefaultWorkspaceBaseEnvironmentReq.DefaultWorkspaceBaseEnvironment.Name, "name", updateDefaultWorkspaceBaseEnvironmentReq.DefaultWorkspaceBaseEnvironment.Name, `The resource name of this singleton resource.`)

	cmd.Use = "update-default-workspace-base-environment NAME UPDATE_MASK"
	cmd.Short = `Update the default workspace base environment configuration.`
	cmd.Long = `Update the default workspace base environment configuration.
  
  Updates the default WorkspaceBaseEnvironment configuration for the workspace.
  Sets the specified base environments as the workspace defaults for CPU and/or
  GPU compute.

  Arguments:
    NAME: The resource name of this singleton resource. Format:
      default-workspace-base-environment
    UPDATE_MASK: Field mask specifying which fields to update. Use comma as the separator
      for multiple fields (no space). The special value '*' indicates that all
      fields should be updated (full replacement). Valid field paths:
      cpu_workspace_base_environment, gpu_workspace_base_environment
      
      To unset one or both defaults, include the field path(s) in the mask and
      omit them from the request body. To unset both, you must list both paths
      explicitly — the wildcard '*' cannot be used to unset fields.`

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
			diags := updateDefaultWorkspaceBaseEnvironmentJson.Unmarshal(&updateDefaultWorkspaceBaseEnvironmentReq.DefaultWorkspaceBaseEnvironment)
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
		updateDefaultWorkspaceBaseEnvironmentReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateDefaultWorkspaceBaseEnvironmentReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}

		response, err := w.Environments.UpdateDefaultWorkspaceBaseEnvironment(ctx, updateDefaultWorkspaceBaseEnvironmentReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateDefaultWorkspaceBaseEnvironmentOverrides {
		fn(cmd, &updateDefaultWorkspaceBaseEnvironmentReq)
	}

	return cmd
}

// start update-workspace-base-environment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateWorkspaceBaseEnvironmentOverrides []func(
	*cobra.Command,
	*environments.UpdateWorkspaceBaseEnvironmentRequest,
)

func newUpdateWorkspaceBaseEnvironment() *cobra.Command {
	cmd := &cobra.Command{}

	var updateWorkspaceBaseEnvironmentReq environments.UpdateWorkspaceBaseEnvironmentRequest
	updateWorkspaceBaseEnvironmentReq.WorkspaceBaseEnvironment = environments.WorkspaceBaseEnvironment{}
	var updateWorkspaceBaseEnvironmentJson flags.JsonFlag

	var updateWorkspaceBaseEnvironmentSkipWait bool
	var updateWorkspaceBaseEnvironmentTimeout time.Duration

	cmd.Flags().BoolVar(&updateWorkspaceBaseEnvironmentSkipWait, "no-wait", updateWorkspaceBaseEnvironmentSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&updateWorkspaceBaseEnvironmentTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&updateWorkspaceBaseEnvironmentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().Var(&updateWorkspaceBaseEnvironmentReq.WorkspaceBaseEnvironment.BaseEnvironmentType, "base-environment-type", `The type of base environment (CPU or GPU). Supported values: [CPU, GPU]`)
	cmd.Flags().StringVar(&updateWorkspaceBaseEnvironmentReq.WorkspaceBaseEnvironment.Filepath, "filepath", updateWorkspaceBaseEnvironmentReq.WorkspaceBaseEnvironment.Filepath, `The WSFS or UC Volumes path to the environment YAML file.`)
	cmd.Flags().StringVar(&updateWorkspaceBaseEnvironmentReq.WorkspaceBaseEnvironment.Name, "name", updateWorkspaceBaseEnvironmentReq.WorkspaceBaseEnvironment.Name, `The resource name of the workspace base environment.`)

	cmd.Use = "update-workspace-base-environment NAME DISPLAY_NAME"
	cmd.Short = `Update a workspace base environment.`
	cmd.Long = `Update a workspace base environment.
  
  Updates an existing WorkspaceBaseEnvironment. This is a long-running
  operation. The operation will asynchronously regenerate the materialized
  environment and is only marked as done when the materialized environment has
  been successfully generated or has failed. The existing materialized
  environment remains available until it expires.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: 
    DISPLAY_NAME: Human-readable display name for the workspace base environment.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only NAME as positional arguments. Provide 'display_name' in your JSON input")
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
			diags := updateWorkspaceBaseEnvironmentJson.Unmarshal(&updateWorkspaceBaseEnvironmentReq.WorkspaceBaseEnvironment)
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
		updateWorkspaceBaseEnvironmentReq.Name = args[0]
		if !cmd.Flags().Changed("json") {
			updateWorkspaceBaseEnvironmentReq.WorkspaceBaseEnvironment.DisplayName = args[1]
		}

		// Determine which mode to execute based on flags.
		switch {
		case updateWorkspaceBaseEnvironmentSkipWait:
			wait, err := w.Environments.UpdateWorkspaceBaseEnvironment(ctx, updateWorkspaceBaseEnvironmentReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			operation, err := w.Environments.GetOperation(ctx, environments.GetOperationRequest{
				Name: wait.Name(),
			})
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, operation)

		default:
			wait, err := w.Environments.UpdateWorkspaceBaseEnvironment(ctx, updateWorkspaceBaseEnvironmentReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			sp := cmdio.NewSpinner(ctx)
			sp.Update("Waiting for update-workspace-base-environment to complete...")

			// Wait for completion.
			opts := api.WithTimeout(updateWorkspaceBaseEnvironmentTimeout)
			response, err := wait.Wait(ctx, opts)
			if err != nil {
				return err
			}
			sp.Close()
			return cmdio.Render(ctx, response)
		}
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateWorkspaceBaseEnvironmentOverrides {
		fn(cmd, &updateWorkspaceBaseEnvironmentReq)
	}

	return cmd
}

// end service Environments
