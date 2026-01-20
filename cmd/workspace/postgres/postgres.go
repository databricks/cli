// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package postgres

import (
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
		Short: `The Postgres API provides access to a Postgres database via REST API or direct SQL.`,
		Long: `The Postgres API provides access to a Postgres database via REST API or direct
  SQL.`,
		GroupID: "postgres",

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
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

	cmd.Flags().StringVar(&createBranchReq.BranchId, "branch-id", createBranchReq.BranchId, `The ID to use for the Branch, which will become the final component of the branch's resource name.`)
	cmd.Flags().StringVar(&createBranchReq.Branch.Name, "name", createBranchReq.Branch.Name, `The resource name of the branch.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "create-branch PARENT"
	cmd.Short = `Create a Branch.`
	cmd.Long = `Create a Branch.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    PARENT: The Project where this Branch will be created. Format:
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

		if cmd.Flags().Changed("json") {
			diags := createBranchJson.Unmarshal(&createBranchReq.Branch)
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
		createBranchReq.Parent = args[0]

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

	cmd.Flags().StringVar(&createEndpointReq.EndpointId, "endpoint-id", createEndpointReq.EndpointId, `The ID to use for the Endpoint, which will become the final component of the endpoint's resource name.`)
	cmd.Flags().StringVar(&createEndpointReq.Endpoint.Name, "name", createEndpointReq.Endpoint.Name, `The resource name of the endpoint.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "create-endpoint PARENT"
	cmd.Short = `Create an Endpoint.`
	cmd.Long = `Create an Endpoint.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    PARENT: The Branch where this Endpoint will be created. Format:
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

		if cmd.Flags().Changed("json") {
			diags := createEndpointJson.Unmarshal(&createEndpointReq.Endpoint)
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
		createEndpointReq.Parent = args[0]

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

	cmd.Flags().StringVar(&createProjectReq.ProjectId, "project-id", createProjectReq.ProjectId, `The ID to use for the Project, which will become the final component of the project's resource name.`)
	cmd.Flags().StringVar(&createProjectReq.Project.Name, "name", createProjectReq.Project.Name, `The resource name of the project.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "create-project"
	cmd.Short = `Create a Project.`
	cmd.Long = `Create a Project.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.`

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
			diags := createProjectJson.Unmarshal(&createProjectReq.Project)
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

	cmd.Flags().StringVar(&createRoleReq.Role.Name, "name", createRoleReq.Role.Name, `The resource name of the role.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "create-role PARENT ROLE_ID"
	cmd.Short = `Create a postgres role for a branch.`
	cmd.Long = `Create a postgres role for a branch.

  Create a role for a branch.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    PARENT: The Branch where this Role is created. Format:
      projects/{project_id}/branches/{branch_id}
    ROLE_ID: The ID to use for the Role, which will become the final component of the
      branch's resource name. This ID becomes the role in postgres.

      This value should be 4-63 characters, and only use characters available in
      DNS names, as defined by RFC-1123`

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
			diags := createRoleJson.Unmarshal(&createRoleReq.Role)
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
		createRoleReq.Parent = args[0]
		createRoleReq.RoleId = args[1]

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

	cmd.Use = "delete-branch NAME"
	cmd.Short = `Delete a Branch.`
	cmd.Long = `Delete a Branch.

  Arguments:
    NAME: The name of the Branch to delete. Format:
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

		err = w.Postgres.DeleteBranch(ctx, deleteBranchReq)
		if err != nil {
			return err
		}
		return nil
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

	cmd.Use = "delete-endpoint NAME"
	cmd.Short = `Delete an Endpoint.`
	cmd.Long = `Delete an Endpoint.

  Arguments:
    NAME: The name of the Endpoint to delete. Format:
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

		err = w.Postgres.DeleteEndpoint(ctx, deleteEndpointReq)
		if err != nil {
			return err
		}
		return nil
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

	cmd.Use = "delete-project NAME"
	cmd.Short = `Delete a Project.`
	cmd.Long = `Delete a Project.

  Arguments:
    NAME: The name of the Project to delete. Format: projects/{project_id}`

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

		err = w.Postgres.DeleteProject(ctx, deleteProjectReq)
		if err != nil {
			return err
		}
		return nil
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

  Delete a role in a branch.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: The resource name of the postgres role. Format:
      projects/{project_id}/branch/{branch_id}/roles/{role_id}`

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

  Arguments:
    NAME: The name of the Branch to retrieve. Format:
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

  Arguments:
    NAME: The name of the Endpoint to retrieve. Format:
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

  Arguments:
    NAME: The name of the Project to retrieve. Format: projects/{project_id}`

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

  Get a Role.

  Arguments:
    NAME: The name of the Role to retrieve. Format:
      projects/{project_id}/branches/{branch_id}/roles/{role_id}`

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
	cmd.Flags().StringVar(&listBranchesReq.PageToken, "page-token", listBranchesReq.PageToken, `Pagination token to go to the next page of Branches.`)

	cmd.Use = "list-branches PARENT"
	cmd.Short = `List Branches.`
	cmd.Long = `List Branches.

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
	cmd.Flags().StringVar(&listEndpointsReq.PageToken, "page-token", listEndpointsReq.PageToken, `Pagination token to go to the next page of Endpoints.`)

	cmd.Use = "list-endpoints PARENT"
	cmd.Short = `List Endpoints.`
	cmd.Long = `List Endpoints.

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
	cmd.Flags().StringVar(&listProjectsReq.PageToken, "page-token", listProjectsReq.PageToken, `Pagination token to go to the next page of Projects.`)

	cmd.Use = "list-projects"
	cmd.Short = `List Projects.`
	cmd.Long = `List Projects.`

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
	cmd.Flags().StringVar(&listRolesReq.PageToken, "page-token", listRolesReq.PageToken, `Pagination token to go to the next page of Roles.`)

	cmd.Use = "list-roles PARENT"
	cmd.Short = `List postgres roles in a branch.`
	cmd.Long = `List postgres roles in a branch.

  List Roles.

  Arguments:
    PARENT: The Branch that owns this collection of roles. Format:
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

	cmd.Flags().StringVar(&updateBranchReq.Branch.Name, "name", updateBranchReq.Branch.Name, `The resource name of the branch.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "update-branch NAME UPDATE_MASK"
	cmd.Short = `Update a Branch.`
	cmd.Long = `Update a Branch.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: The resource name of the branch. Format:
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
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
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

	cmd.Flags().StringVar(&updateEndpointReq.Endpoint.Name, "name", updateEndpointReq.Endpoint.Name, `The resource name of the endpoint.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "update-endpoint NAME UPDATE_MASK"
	cmd.Short = `Update an Endpoint.`
	cmd.Long = `Update an Endpoint.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: The resource name of the endpoint. Format:
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
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
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

	cmd.Flags().StringVar(&updateProjectReq.Project.Name, "name", updateProjectReq.Project.Name, `The resource name of the project.`)
	// TODO: complex arg: spec
	// TODO: complex arg: status

	cmd.Use = "update-project NAME UPDATE_MASK"
	cmd.Short = `Update a Project.`
	cmd.Long = `Update a Project.

  This is a long-running operation. By default, the command waits for the
  operation to complete. Use --no-wait to return immediately with the raw
  operation details. The operation's 'name' field can then be used to poll for
  completion using the get-operation command.

  Arguments:
    NAME: The resource name of the project. Format: projects/{project_id}
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
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
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
