// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package postgres

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
	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "postgres",
		Short: `Use the Postgres API to create and manage Lakebase Autoscaling Postgres infrastructure, including projects, branches, compute endpoints, and roles.`,
		Long: `Use the Postgres API to create and manage Lakebase Autoscaling Postgres
  infrastructure, including projects, branches, compute endpoints, and roles.

  This API manages database infrastructure only. To query or modify data, use
  the Data API or direct SQL connections.

  **About resource IDs and names**

  Resources are identified by hierarchical resource names like
  projects/{project_id}/branches/{branch_id}/endpoints/{endpoint_id}. The
  name field on each resource contains this full path and is output-only. Note
  that name refers to this resource path, not the user-visible display_name.`,
		GroupID: "postgres",
		RunE:    root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreateBranch())
	cmd.AddCommand(newCreateEndpoint())
	cmd.AddCommand(newCreateProject())
	cmd.AddCommand(newCreateRole())
	cmd.AddCommand(newDeleteBranch())
	cmd.AddCommand(newDeleteEndpoint())
	cmd.AddCommand(newDeleteProject())
	cmd.AddCommand(newDeleteRole())
	cmd.AddCommand(newGenerateDatabaseCredential())
	cmd.AddCommand(newGetBranch())
	cmd.AddCommand(newGetEndpoint())
	cmd.AddCommand(newGetOperation())
	cmd.AddCommand(newGetProject())
	cmd.AddCommand(newGetRole())
	cmd.AddCommand(newListBranches())
	cmd.AddCommand(newListEndpoints())
	cmd.AddCommand(newListProjects())
	cmd.AddCommand(newListRoles())
	cmd.AddCommand(newUpdateBranch())
	cmd.AddCommand(newUpdateEndpoint())
	cmd.AddCommand(newUpdateProject())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-branch command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createBranchOverrides []func(
	*cobra.Command,
	*postgres.CreateBranchRequest,
)

func newCreateBranch() *cobra.Command {
	cmd := &cobra.Command{}

	var createBranchReq postgres.CreateBranchRequest
	createBranchReq.Branch = postgres.Branch{}
	var createBranchJson flags.JsonFlag

	var createBranchSkipWait bool
	var createBranchTimeout time.Duration

	cmd.Flags().BoolVar(&createBranchSkipWait, "no-wait", createBranchSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&createBranchTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&createBranchJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createBranchReq.Branch.Name, "name", createBranchReq.Branch.Name, `Output only.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "create-branch PARENT BRANCH_ID"
	cmd.Short = `Create a Branch.`
	cmd.Long = `Create a Branch.

  Creates a new database branch in the project.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    PARENT: The Project where this Branch will be created. Format:
      projects/{project_id}
    BRANCH_ID: The ID to use for the Branch. This becomes the final component of the
      branch's resource name. The ID is required and must be 1-63 characters
      long, start with a lowercase letter, and contain only lowercase letters,
      numbers, and hyphens. For example, development becomes
      projects/my-app/branches/development.`

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
			diags := createBranchJson.Unmarshal(&createBranchReq.Branch)
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
		createBranchReq.Parent = args[0]
		createBranchReq.BranchId = args[1]

		// Determine which mode to execute based on flags.
		switch {
		case createBranchSkipWait:
			wait, err := w.Postgres.CreateBranch(ctx, createBranchReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			operation, err := w.Postgres.GetOperation(ctx, postgres.GetOperationRequest{
				Name: wait.Name(),
			})
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, operation)

		default:
			wait, err := w.Postgres.CreateBranch(ctx, createBranchReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			spinner := cmdio.Spinner(ctx)
			spinner <- "Waiting for create-branch to complete..."

			// Wait for completion.
			opts := api.WithTimeout(createBranchTimeout)
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
	for _, fn := range createBranchOverrides {
		fn(cmd, &createBranchReq)
	}

	return cmd
}

// start create-endpoint command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createEndpointOverrides []func(
	*cobra.Command,
	*postgres.CreateEndpointRequest,
)

func newCreateEndpoint() *cobra.Command {
	cmd := &cobra.Command{}

	var createEndpointReq postgres.CreateEndpointRequest
	createEndpointReq.Endpoint = postgres.Endpoint{}
	var createEndpointJson flags.JsonFlag

	var createEndpointSkipWait bool
	var createEndpointTimeout time.Duration

	cmd.Flags().BoolVar(&createEndpointSkipWait, "no-wait", createEndpointSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&createEndpointTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&createEndpointJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createEndpointReq.Endpoint.Name, "name", createEndpointReq.Endpoint.Name, `Output only.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "create-endpoint PARENT ENDPOINT_ID"
	cmd.Short = `Create an Endpoint.`
	cmd.Long = `Create an Endpoint.

  Creates a new compute endpoint in the branch.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    PARENT: The Branch where this Endpoint will be created. Format:
      projects/{project_id}/branches/{branch_id}
    ENDPOINT_ID: The ID to use for the Endpoint. This becomes the final component of the
      endpoint's resource name. The ID is required and must be 1-63 characters
      long, start with a lowercase letter, and contain only lowercase letters,
      numbers, and hyphens. For example, primary becomes
      projects/my-app/branches/development/endpoints/primary.`

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
			diags := createEndpointJson.Unmarshal(&createEndpointReq.Endpoint)
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
		createEndpointReq.Parent = args[0]
		createEndpointReq.EndpointId = args[1]

		// Determine which mode to execute based on flags.
		switch {
		case createEndpointSkipWait:
			wait, err := w.Postgres.CreateEndpoint(ctx, createEndpointReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			operation, err := w.Postgres.GetOperation(ctx, postgres.GetOperationRequest{
				Name: wait.Name(),
			})
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, operation)

		default:
			wait, err := w.Postgres.CreateEndpoint(ctx, createEndpointReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			spinner := cmdio.Spinner(ctx)
			spinner <- "Waiting for create-endpoint to complete..."

			// Wait for completion.
			opts := api.WithTimeout(createEndpointTimeout)
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
	for _, fn := range createEndpointOverrides {
		fn(cmd, &createEndpointReq)
	}

	return cmd
}

// start create-project command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createProjectOverrides []func(
	*cobra.Command,
	*postgres.CreateProjectRequest,
)

func newCreateProject() *cobra.Command {
	cmd := &cobra.Command{}

	var createProjectReq postgres.CreateProjectRequest
	createProjectReq.Project = postgres.Project{}
	var createProjectJson flags.JsonFlag

	var createProjectSkipWait bool
	var createProjectTimeout time.Duration

	cmd.Flags().BoolVar(&createProjectSkipWait, "no-wait", createProjectSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&createProjectTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&createProjectJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: initial_endpoint_spec
	cmd.Flags().StringVar(&createProjectReq.Project.Name, "name", createProjectReq.Project.Name, `Output only.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "create-project PROJECT_ID"
	cmd.Short = `Create a Project.`
	cmd.Long = `Create a Project.

  Creates a new Lakebase Autoscaling Postgres database project, which contains
  branches and compute endpoints.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    PROJECT_ID: The ID to use for the Project. This becomes the final component of the
      project's resource name. The ID is required and must be 1-63 characters
      long, start with a lowercase letter, and contain only lowercase letters,
      numbers, and hyphens. For example, my-app becomes projects/my-app.`

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
			diags := createProjectJson.Unmarshal(&createProjectReq.Project)
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
		createProjectReq.ProjectId = args[0]

		// Determine which mode to execute based on flags.
		switch {
		case createProjectSkipWait:
			wait, err := w.Postgres.CreateProject(ctx, createProjectReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			operation, err := w.Postgres.GetOperation(ctx, postgres.GetOperationRequest{
				Name: wait.Name(),
			})
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, operation)

		default:
			wait, err := w.Postgres.CreateProject(ctx, createProjectReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			spinner := cmdio.Spinner(ctx)
			spinner <- "Waiting for create-project to complete..."

			// Wait for completion.
			opts := api.WithTimeout(createProjectTimeout)
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
	for _, fn := range createProjectOverrides {
		fn(cmd, &createProjectReq)
	}

	return cmd
}

// start create-role command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createRoleOverrides []func(
	*cobra.Command,
	*postgres.CreateRoleRequest,
)

func newCreateRole() *cobra.Command {
	cmd := &cobra.Command{}

	var createRoleReq postgres.CreateRoleRequest
	createRoleReq.Role = postgres.Role{}
	var createRoleJson flags.JsonFlag

	var createRoleSkipWait bool
	var createRoleTimeout time.Duration

	cmd.Flags().BoolVar(&createRoleSkipWait, "no-wait", createRoleSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&createRoleTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&createRoleJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createRoleReq.RoleId, "role-id", createRoleReq.RoleId, `The ID to use for the Role, which will become the final component of the role's resource name.`)
	cmd.Flags().StringVar(&createRoleReq.Role.Name, "name", createRoleReq.Role.Name, `Output only.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "create-role PARENT"
	cmd.Short = `Create a postgres role for a branch.`
	cmd.Long = `Create a postgres role for a branch.

  Creates a new Postgres role in the branch.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    PARENT: The Branch where this Role is created. Format:
      projects/{project_id}/branches/{branch_id}`

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
			diags := createRoleJson.Unmarshal(&createRoleReq.Role)
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
		createRoleReq.Parent = args[0]

		// Determine which mode to execute based on flags.
		switch {
		case createRoleSkipWait:
			wait, err := w.Postgres.CreateRole(ctx, createRoleReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			operation, err := w.Postgres.GetOperation(ctx, postgres.GetOperationRequest{
				Name: wait.Name(),
			})
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, operation)

		default:
			wait, err := w.Postgres.CreateRole(ctx, createRoleReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			spinner := cmdio.Spinner(ctx)
			spinner <- "Waiting for create-role to complete..."

			// Wait for completion.
			opts := api.WithTimeout(createRoleTimeout)
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
	for _, fn := range createRoleOverrides {
		fn(cmd, &createRoleReq)
	}

	return cmd
}

// start delete-branch command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteBranchOverrides []func(
	*cobra.Command,
	*postgres.DeleteBranchRequest,
)

func newDeleteBranch() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteBranchReq postgres.DeleteBranchRequest

	var deleteBranchSkipWait bool
	var deleteBranchTimeout time.Duration

	cmd.Flags().BoolVar(&deleteBranchSkipWait, "no-wait", deleteBranchSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&deleteBranchTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Use = "delete-branch NAME"
	cmd.Short = `Delete a Branch.`
	cmd.Long = `Delete a Branch.

  Deletes the specified database branch.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: The full resource path of the branch to delete. Format:
      projects/{project_id}/branches/{branch_id}`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteBranchReq.Name = args[0]

		// Determine which mode to execute based on flags.
		switch {
		case deleteBranchSkipWait:
			wait, err := w.Postgres.DeleteBranch(ctx, deleteBranchReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			operation, err := w.Postgres.GetOperation(ctx, postgres.GetOperationRequest{
				Name: wait.Name(),
			})
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, operation)

		default:
			wait, err := w.Postgres.DeleteBranch(ctx, deleteBranchReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			spinner := cmdio.Spinner(ctx)
			spinner <- "Waiting for delete-branch to complete..."

			// Wait for completion.
			opts := api.WithTimeout(deleteBranchTimeout)

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
	for _, fn := range deleteBranchOverrides {
		fn(cmd, &deleteBranchReq)
	}

	return cmd
}

// start delete-endpoint command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteEndpointOverrides []func(
	*cobra.Command,
	*postgres.DeleteEndpointRequest,
)

func newDeleteEndpoint() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteEndpointReq postgres.DeleteEndpointRequest

	var deleteEndpointSkipWait bool
	var deleteEndpointTimeout time.Duration

	cmd.Flags().BoolVar(&deleteEndpointSkipWait, "no-wait", deleteEndpointSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&deleteEndpointTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Use = "delete-endpoint NAME"
	cmd.Short = `Delete an Endpoint.`
	cmd.Long = `Delete an Endpoint.

  Deletes the specified compute endpoint.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: The full resource path of the endpoint to delete. Format:
      projects/{project_id}/branches/{branch_id}/endpoints/{endpoint_id}`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteEndpointReq.Name = args[0]

		// Determine which mode to execute based on flags.
		switch {
		case deleteEndpointSkipWait:
			wait, err := w.Postgres.DeleteEndpoint(ctx, deleteEndpointReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			operation, err := w.Postgres.GetOperation(ctx, postgres.GetOperationRequest{
				Name: wait.Name(),
			})
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, operation)

		default:
			wait, err := w.Postgres.DeleteEndpoint(ctx, deleteEndpointReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			spinner := cmdio.Spinner(ctx)
			spinner <- "Waiting for delete-endpoint to complete..."

			// Wait for completion.
			opts := api.WithTimeout(deleteEndpointTimeout)

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
	for _, fn := range deleteEndpointOverrides {
		fn(cmd, &deleteEndpointReq)
	}

	return cmd
}

// start delete-project command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteProjectOverrides []func(
	*cobra.Command,
	*postgres.DeleteProjectRequest,
)

func newDeleteProject() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteProjectReq postgres.DeleteProjectRequest

	var deleteProjectSkipWait bool
	var deleteProjectTimeout time.Duration

	cmd.Flags().BoolVar(&deleteProjectSkipWait, "no-wait", deleteProjectSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&deleteProjectTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Use = "delete-project NAME"
	cmd.Short = `Delete a Project.`
	cmd.Long = `Delete a Project.

  Deletes the specified database project.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: The full resource path of the project to delete. Format:
      projects/{project_id}`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteProjectReq.Name = args[0]

		// Determine which mode to execute based on flags.
		switch {
		case deleteProjectSkipWait:
			wait, err := w.Postgres.DeleteProject(ctx, deleteProjectReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			operation, err := w.Postgres.GetOperation(ctx, postgres.GetOperationRequest{
				Name: wait.Name(),
			})
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, operation)

		default:
			wait, err := w.Postgres.DeleteProject(ctx, deleteProjectReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			spinner := cmdio.Spinner(ctx)
			spinner <- "Waiting for delete-project to complete..."

			// Wait for completion.
			opts := api.WithTimeout(deleteProjectTimeout)

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
	for _, fn := range deleteProjectOverrides {
		fn(cmd, &deleteProjectReq)
	}

	return cmd
}

// start delete-role command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteRoleOverrides []func(
	*cobra.Command,
	*postgres.DeleteRoleRequest,
)

func newDeleteRole() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteRoleReq postgres.DeleteRoleRequest

	var deleteRoleSkipWait bool
	var deleteRoleTimeout time.Duration

	cmd.Flags().BoolVar(&deleteRoleSkipWait, "no-wait", deleteRoleSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&deleteRoleTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().StringVar(&deleteRoleReq.ReassignOwnedTo, "reassign-owned-to", deleteRoleReq.ReassignOwnedTo, `Reassign objects.`)

	cmd.Use = "delete-role NAME"
	cmd.Short = `Delete a postgres role in a branch.`
	cmd.Long = `Delete a postgres role in a branch.

  Deletes the specified Postgres role.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: The full resource path of the role to delete. Format:
      projects/{project_id}/branches/{branch_id}/roles/{role_id}`

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

		deleteRoleReq.Name = args[0]

		// Determine which mode to execute based on flags.
		switch {
		case deleteRoleSkipWait:
			wait, err := w.Postgres.DeleteRole(ctx, deleteRoleReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			operation, err := w.Postgres.GetOperation(ctx, postgres.GetOperationRequest{
				Name: wait.Name(),
			})
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, operation)

		default:
			wait, err := w.Postgres.DeleteRole(ctx, deleteRoleReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			spinner := cmdio.Spinner(ctx)
			spinner <- "Waiting for delete-role to complete..."

			// Wait for completion.
			opts := api.WithTimeout(deleteRoleTimeout)

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
	for _, fn := range deleteRoleOverrides {
		fn(cmd, &deleteRoleReq)
	}

	return cmd
}

// start generate-database-credential command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var generateDatabaseCredentialOverrides []func(
	*cobra.Command,
	*postgres.GenerateDatabaseCredentialRequest,
)

func newGenerateDatabaseCredential() *cobra.Command {
	cmd := &cobra.Command{}

	var generateDatabaseCredentialReq postgres.GenerateDatabaseCredentialRequest
	var generateDatabaseCredentialJson flags.JsonFlag

	cmd.Flags().Var(&generateDatabaseCredentialJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: claims

	cmd.Use = "generate-database-credential ENDPOINT"
	cmd.Short = `Generate OAuth credentials for a Postgres database.`
	cmd.Long = `Generate OAuth credentials for a Postgres database.

  Arguments:
    ENDPOINT: This field is not yet supported. The endpoint for which this credential
      will be generated. Format:
      projects/{project_id}/branches/{branch_id}/endpoints/{endpoint_id}`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'endpoint' in your JSON input")
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
			diags := generateDatabaseCredentialJson.Unmarshal(&generateDatabaseCredentialReq)
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
			generateDatabaseCredentialReq.Endpoint = args[0]
		}

		response, err := w.Postgres.GenerateDatabaseCredential(ctx, generateDatabaseCredentialReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range generateDatabaseCredentialOverrides {
		fn(cmd, &generateDatabaseCredentialReq)
	}

	return cmd
}

// start get-branch command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getBranchOverrides []func(
	*cobra.Command,
	*postgres.GetBranchRequest,
)

func newGetBranch() *cobra.Command {
	cmd := &cobra.Command{}

	var getBranchReq postgres.GetBranchRequest

	cmd.Use = "get-branch NAME"
	cmd.Short = `Get a Branch.`
	cmd.Long = `Get a Branch.

  Retrieves information about the specified database branch.

  Arguments:
    NAME: The full resource path of the branch to retrieve. Format:
      projects/{project_id}/branches/{branch_id}`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getBranchReq.Name = args[0]

		response, err := w.Postgres.GetBranch(ctx, getBranchReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getBranchOverrides {
		fn(cmd, &getBranchReq)
	}

	return cmd
}

// start get-endpoint command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getEndpointOverrides []func(
	*cobra.Command,
	*postgres.GetEndpointRequest,
)

func newGetEndpoint() *cobra.Command {
	cmd := &cobra.Command{}

	var getEndpointReq postgres.GetEndpointRequest

	cmd.Use = "get-endpoint NAME"
	cmd.Short = `Get an Endpoint.`
	cmd.Long = `Get an Endpoint.

  Retrieves information about the specified compute endpoint, including its
  connection details and operational state.

  Arguments:
    NAME: The full resource path of the endpoint to retrieve. Format:
      projects/{project_id}/branches/{branch_id}/endpoints/{endpoint_id}`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getEndpointReq.Name = args[0]

		response, err := w.Postgres.GetEndpoint(ctx, getEndpointReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getEndpointOverrides {
		fn(cmd, &getEndpointReq)
	}

	return cmd
}

// start get-operation command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOperationOverrides []func(
	*cobra.Command,
	*postgres.GetOperationRequest,
)

func newGetOperation() *cobra.Command {
	cmd := &cobra.Command{}

	var getOperationReq postgres.GetOperationRequest

	cmd.Use = "get-operation NAME"
	cmd.Short = `Get an Operation.`
	cmd.Long = `Get an Operation.

  Retrieves the status of a long-running operation.

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

		response, err := w.Postgres.GetOperation(ctx, getOperationReq)
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

// start get-project command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getProjectOverrides []func(
	*cobra.Command,
	*postgres.GetProjectRequest,
)

func newGetProject() *cobra.Command {
	cmd := &cobra.Command{}

	var getProjectReq postgres.GetProjectRequest

	cmd.Use = "get-project NAME"
	cmd.Short = `Get a Project.`
	cmd.Long = `Get a Project.

  Retrieves information about the specified database project.

  Arguments:
    NAME: The full resource path of the project to retrieve. Format:
      projects/{project_id}`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getProjectReq.Name = args[0]

		response, err := w.Postgres.GetProject(ctx, getProjectReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getProjectOverrides {
		fn(cmd, &getProjectReq)
	}

	return cmd
}

// start get-role command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getRoleOverrides []func(
	*cobra.Command,
	*postgres.GetRoleRequest,
)

func newGetRole() *cobra.Command {
	cmd := &cobra.Command{}

	var getRoleReq postgres.GetRoleRequest

	cmd.Use = "get-role NAME"
	cmd.Short = `Get a postgres role in a branch.`
	cmd.Long = `Get a postgres role in a branch.

  Retrieves information about the specified Postgres role, including its
  authentication method and permissions.

  Arguments:
    NAME: The full resource path of the role to retrieve. Format:
      projects/{project_id}/branches/{branch_id}/roles/{role_id}`

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

		getRoleReq.Name = args[0]

		response, err := w.Postgres.GetRole(ctx, getRoleReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getRoleOverrides {
		fn(cmd, &getRoleReq)
	}

	return cmd
}

// start list-branches command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listBranchesOverrides []func(
	*cobra.Command,
	*postgres.ListBranchesRequest,
)

func newListBranches() *cobra.Command {
	cmd := &cobra.Command{}

	var listBranchesReq postgres.ListBranchesRequest

	cmd.Flags().IntVar(&listBranchesReq.PageSize, "page-size", listBranchesReq.PageSize, `Upper bound for items returned.`)
	cmd.Flags().StringVar(&listBranchesReq.PageToken, "page-token", listBranchesReq.PageToken, `Page token from a previous response.`)

	cmd.Use = "list-branches PARENT"
	cmd.Short = `List Branches.`
	cmd.Long = `List Branches.

  Returns a paginated list of database branches in the project.

  Arguments:
    PARENT: The Project that owns this collection of branches. Format:
      projects/{project_id}`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listBranchesReq.Parent = args[0]

		response := w.Postgres.ListBranches(ctx, listBranchesReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listBranchesOverrides {
		fn(cmd, &listBranchesReq)
	}

	return cmd
}

// start list-endpoints command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listEndpointsOverrides []func(
	*cobra.Command,
	*postgres.ListEndpointsRequest,
)

func newListEndpoints() *cobra.Command {
	cmd := &cobra.Command{}

	var listEndpointsReq postgres.ListEndpointsRequest

	cmd.Flags().IntVar(&listEndpointsReq.PageSize, "page-size", listEndpointsReq.PageSize, `Upper bound for items returned.`)
	cmd.Flags().StringVar(&listEndpointsReq.PageToken, "page-token", listEndpointsReq.PageToken, `Page token from a previous response.`)

	cmd.Use = "list-endpoints PARENT"
	cmd.Short = `List Endpoints.`
	cmd.Long = `List Endpoints.

  Returns a paginated list of compute endpoints in the branch.

  Arguments:
    PARENT: The Branch that owns this collection of endpoints. Format:
      projects/{project_id}/branches/{branch_id}`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listEndpointsReq.Parent = args[0]

		response := w.Postgres.ListEndpoints(ctx, listEndpointsReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listEndpointsOverrides {
		fn(cmd, &listEndpointsReq)
	}

	return cmd
}

// start list-projects command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listProjectsOverrides []func(
	*cobra.Command,
	*postgres.ListProjectsRequest,
)

func newListProjects() *cobra.Command {
	cmd := &cobra.Command{}

	var listProjectsReq postgres.ListProjectsRequest

	cmd.Flags().IntVar(&listProjectsReq.PageSize, "page-size", listProjectsReq.PageSize, `Upper bound for items returned.`)
	cmd.Flags().StringVar(&listProjectsReq.PageToken, "page-token", listProjectsReq.PageToken, `Page token from a previous response.`)

	cmd.Use = "list-projects"
	cmd.Short = `List Projects.`
	cmd.Long = `List Projects.

  Returns a paginated list of database projects in the workspace that the user
  has permission to access.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.Postgres.ListProjects(ctx, listProjectsReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listProjectsOverrides {
		fn(cmd, &listProjectsReq)
	}

	return cmd
}

// start list-roles command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listRolesOverrides []func(
	*cobra.Command,
	*postgres.ListRolesRequest,
)

func newListRoles() *cobra.Command {
	cmd := &cobra.Command{}

	var listRolesReq postgres.ListRolesRequest

	cmd.Flags().IntVar(&listRolesReq.PageSize, "page-size", listRolesReq.PageSize, `Upper bound for items returned.`)
	cmd.Flags().StringVar(&listRolesReq.PageToken, "page-token", listRolesReq.PageToken, `Page token from a previous response.`)

	cmd.Use = "list-roles PARENT"
	cmd.Short = `List postgres roles in a branch.`
	cmd.Long = `List postgres roles in a branch.

  Returns a paginated list of Postgres roles in the branch.

  Arguments:
    PARENT: The Branch that owns this collection of roles. Format:
      projects/{project_id}/branches/{branch_id}`

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

		listRolesReq.Parent = args[0]

		response := w.Postgres.ListRoles(ctx, listRolesReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listRolesOverrides {
		fn(cmd, &listRolesReq)
	}

	return cmd
}

// start update-branch command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateBranchOverrides []func(
	*cobra.Command,
	*postgres.UpdateBranchRequest,
)

func newUpdateBranch() *cobra.Command {
	cmd := &cobra.Command{}

	var updateBranchReq postgres.UpdateBranchRequest
	updateBranchReq.Branch = postgres.Branch{}
	var updateBranchJson flags.JsonFlag

	var updateBranchSkipWait bool
	var updateBranchTimeout time.Duration

	cmd.Flags().BoolVar(&updateBranchSkipWait, "no-wait", updateBranchSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&updateBranchTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&updateBranchJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateBranchReq.Branch.Name, "name", updateBranchReq.Branch.Name, `Output only.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "update-branch NAME UPDATE_MASK"
	cmd.Short = `Update a Branch.`
	cmd.Long = `Update a Branch.

  Updates the specified database branch. You can set this branch as the
  project's default branch, or protect/unprotect it.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: Output only. The full resource path of the branch. Format:
      projects/{project_id}/branches/{branch_id}
    UPDATE_MASK: The list of fields to update. If unspecified, all fields will be updated
      when possible.`

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
			diags := updateBranchJson.Unmarshal(&updateBranchReq.Branch)
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
		updateBranchReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateBranchReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}

		// Determine which mode to execute based on flags.
		switch {
		case updateBranchSkipWait:
			wait, err := w.Postgres.UpdateBranch(ctx, updateBranchReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			operation, err := w.Postgres.GetOperation(ctx, postgres.GetOperationRequest{
				Name: wait.Name(),
			})
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, operation)

		default:
			wait, err := w.Postgres.UpdateBranch(ctx, updateBranchReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			spinner := cmdio.Spinner(ctx)
			spinner <- "Waiting for update-branch to complete..."

			// Wait for completion.
			opts := api.WithTimeout(updateBranchTimeout)
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
	for _, fn := range updateBranchOverrides {
		fn(cmd, &updateBranchReq)
	}

	return cmd
}

// start update-endpoint command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateEndpointOverrides []func(
	*cobra.Command,
	*postgres.UpdateEndpointRequest,
)

func newUpdateEndpoint() *cobra.Command {
	cmd := &cobra.Command{}

	var updateEndpointReq postgres.UpdateEndpointRequest
	updateEndpointReq.Endpoint = postgres.Endpoint{}
	var updateEndpointJson flags.JsonFlag

	var updateEndpointSkipWait bool
	var updateEndpointTimeout time.Duration

	cmd.Flags().BoolVar(&updateEndpointSkipWait, "no-wait", updateEndpointSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&updateEndpointTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&updateEndpointJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateEndpointReq.Endpoint.Name, "name", updateEndpointReq.Endpoint.Name, `Output only.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "update-endpoint NAME UPDATE_MASK"
	cmd.Short = `Update an Endpoint.`
	cmd.Long = `Update an Endpoint.

  Updates the specified compute endpoint. You can update autoscaling limits,
  suspend timeout, or enable/disable the compute endpoint.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: Output only. The full resource path of the endpoint. Format:
      projects/{project_id}/branches/{branch_id}/endpoints/{endpoint_id}
    UPDATE_MASK: The list of fields to update. If unspecified, all fields will be updated
      when possible.`

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
			diags := updateEndpointJson.Unmarshal(&updateEndpointReq.Endpoint)
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
		updateEndpointReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateEndpointReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}

		// Determine which mode to execute based on flags.
		switch {
		case updateEndpointSkipWait:
			wait, err := w.Postgres.UpdateEndpoint(ctx, updateEndpointReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			operation, err := w.Postgres.GetOperation(ctx, postgres.GetOperationRequest{
				Name: wait.Name(),
			})
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, operation)

		default:
			wait, err := w.Postgres.UpdateEndpoint(ctx, updateEndpointReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			spinner := cmdio.Spinner(ctx)
			spinner <- "Waiting for update-endpoint to complete..."

			// Wait for completion.
			opts := api.WithTimeout(updateEndpointTimeout)
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
	for _, fn := range updateEndpointOverrides {
		fn(cmd, &updateEndpointReq)
	}

	return cmd
}

// start update-project command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateProjectOverrides []func(
	*cobra.Command,
	*postgres.UpdateProjectRequest,
)

func newUpdateProject() *cobra.Command {
	cmd := &cobra.Command{}

	var updateProjectReq postgres.UpdateProjectRequest
	updateProjectReq.Project = postgres.Project{}
	var updateProjectJson flags.JsonFlag

	var updateProjectSkipWait bool
	var updateProjectTimeout time.Duration

	cmd.Flags().BoolVar(&updateProjectSkipWait, "no-wait", updateProjectSkipWait, `do not wait to reach DONE state`)
	cmd.Flags().DurationVar(&updateProjectTimeout, "timeout", 0, `maximum amount of time to reach DONE state`)

	cmd.Flags().Var(&updateProjectJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: initial_endpoint_spec
	cmd.Flags().StringVar(&updateProjectReq.Project.Name, "name", updateProjectReq.Project.Name, `Output only.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "update-project NAME UPDATE_MASK"
	cmd.Short = `Update a Project.`
	cmd.Long = `Update a Project.

  Updates the specified database project.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: Output only. The full resource path of the project. Format:
      projects/{project_id}
    UPDATE_MASK: The list of fields to update. If unspecified, all fields will be updated
      when possible.`

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
			diags := updateProjectJson.Unmarshal(&updateProjectReq.Project)
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
		updateProjectReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateProjectReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}

		// Determine which mode to execute based on flags.
		switch {
		case updateProjectSkipWait:
			wait, err := w.Postgres.UpdateProject(ctx, updateProjectReq)
			if err != nil {
				return err
			}

			// Return operation immediately without waiting.
			operation, err := w.Postgres.GetOperation(ctx, postgres.GetOperationRequest{
				Name: wait.Name(),
			})
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, operation)

		default:
			wait, err := w.Postgres.UpdateProject(ctx, updateProjectReq)
			if err != nil {
				return err
			}

			// Show spinner while waiting for completion.
			spinner := cmdio.Spinner(ctx)
			spinner <- "Waiting for update-project to complete..."

			// Wait for completion.
			opts := api.WithTimeout(updateProjectTimeout)
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
	for _, fn := range updateProjectOverrides {
		fn(cmd, &updateProjectReq)
	}

	return cmd
}

// end service Postgres
