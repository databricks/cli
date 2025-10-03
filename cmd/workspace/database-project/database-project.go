// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package database_project

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/database"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "database-project",
		Short:   `Database Projects provide access to a database via REST API or direct SQL.`,
		Long:    `Database Projects provide access to a database via REST API or direct SQL.`,
		GroupID: "database",
		Annotations: map[string]string{
			"package": "database",
		},

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreateDatabaseBranch())
	cmd.AddCommand(newCreateDatabaseEndpoint())
	cmd.AddCommand(newCreateDatabaseProject())
	cmd.AddCommand(newDeleteDatabaseBranch())
	cmd.AddCommand(newDeleteDatabaseEndpoint())
	cmd.AddCommand(newDeleteDatabaseProject())
	cmd.AddCommand(newGetDatabaseBranch())
	cmd.AddCommand(newGetDatabaseEndpoint())
	cmd.AddCommand(newGetDatabaseProject())
	cmd.AddCommand(newListDatabaseBranches())
	cmd.AddCommand(newListDatabaseEndpoints())
	cmd.AddCommand(newListDatabaseProjects())
	cmd.AddCommand(newRestartDatabaseEndpoint())
	cmd.AddCommand(newUpdateDatabaseBranch())
	cmd.AddCommand(newUpdateDatabaseEndpoint())
	cmd.AddCommand(newUpdateDatabaseProject())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-database-branch command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createDatabaseBranchOverrides []func(
	*cobra.Command,
	*database.CreateDatabaseBranchRequest,
)

func newCreateDatabaseBranch() *cobra.Command {
	cmd := &cobra.Command{}

	var createDatabaseBranchReq database.CreateDatabaseBranchRequest
	createDatabaseBranchReq.DatabaseBranch = database.DatabaseBranch{}
	var createDatabaseBranchJson flags.JsonFlag

	cmd.Flags().Var(&createDatabaseBranchJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createDatabaseBranchReq.DatabaseBranch.BranchId, "branch-id", createDatabaseBranchReq.DatabaseBranch.BranchId, ``)
	cmd.Flags().BoolVar(&createDatabaseBranchReq.DatabaseBranch.Default, "default", createDatabaseBranchReq.DatabaseBranch.Default, `Whether the branch is the project's default branch.`)
	cmd.Flags().BoolVar(&createDatabaseBranchReq.DatabaseBranch.IsProtected, "is-protected", createDatabaseBranchReq.DatabaseBranch.IsProtected, `Whether the branch is protected.`)
	cmd.Flags().StringVar(&createDatabaseBranchReq.DatabaseBranch.ParentId, "parent-id", createDatabaseBranchReq.DatabaseBranch.ParentId, `The id of the parent branch.`)
	cmd.Flags().StringVar(&createDatabaseBranchReq.DatabaseBranch.ParentLsn, "parent-lsn", createDatabaseBranchReq.DatabaseBranch.ParentLsn, `The Log Sequence Number (LSN) on the parent branch from which this branch was created.`)
	cmd.Flags().StringVar(&createDatabaseBranchReq.DatabaseBranch.ParentTime, "parent-time", createDatabaseBranchReq.DatabaseBranch.ParentTime, `The point in time on the parent branch from which this branch was created.`)

	cmd.Use = "create-database-branch PROJECT_ID"
	cmd.Short = `Create a Database Branch.`
	cmd.Long = `Create a Database Branch.`

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
			diags := createDatabaseBranchJson.Unmarshal(&createDatabaseBranchReq.DatabaseBranch)
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
		createDatabaseBranchReq.ProjectId = args[0]

		response, err := w.DatabaseProject.CreateDatabaseBranch(ctx, createDatabaseBranchReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createDatabaseBranchOverrides {
		fn(cmd, &createDatabaseBranchReq)
	}

	return cmd
}

// start create-database-endpoint command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createDatabaseEndpointOverrides []func(
	*cobra.Command,
	*database.CreateDatabaseEndpointRequest,
)

func newCreateDatabaseEndpoint() *cobra.Command {
	cmd := &cobra.Command{}

	var createDatabaseEndpointReq database.CreateDatabaseEndpointRequest
	createDatabaseEndpointReq.DatabaseEndpoint = database.DatabaseEndpoint{}
	var createDatabaseEndpointJson flags.JsonFlag

	cmd.Flags().Var(&createDatabaseEndpointJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().Float64Var(&createDatabaseEndpointReq.DatabaseEndpoint.AutoscalingLimitMaxCu, "autoscaling-limit-max-cu", createDatabaseEndpointReq.DatabaseEndpoint.AutoscalingLimitMaxCu, `The maximum number of Compute Units.`)
	cmd.Flags().Float64Var(&createDatabaseEndpointReq.DatabaseEndpoint.AutoscalingLimitMinCu, "autoscaling-limit-min-cu", createDatabaseEndpointReq.DatabaseEndpoint.AutoscalingLimitMinCu, `The minimum number of Compute Units.`)
	cmd.Flags().BoolVar(&createDatabaseEndpointReq.DatabaseEndpoint.Disabled, "disabled", createDatabaseEndpointReq.DatabaseEndpoint.Disabled, `Whether to restrict connections to the compute endpoint.`)
	cmd.Flags().StringVar(&createDatabaseEndpointReq.DatabaseEndpoint.EndpointId, "endpoint-id", createDatabaseEndpointReq.DatabaseEndpoint.EndpointId, ``)
	cmd.Flags().Var(&createDatabaseEndpointReq.DatabaseEndpoint.PoolerMode, "pooler-mode", `Supported values: [TRANSACTION]`)
	// TODO: complex arg: settings
	cmd.Flags().StringVar(&createDatabaseEndpointReq.DatabaseEndpoint.SuspendTimeoutDuration, "suspend-timeout-duration", createDatabaseEndpointReq.DatabaseEndpoint.SuspendTimeoutDuration, `Duration of inactivity after which the compute endpoint is automatically suspended.`)
	cmd.Flags().Var(&createDatabaseEndpointReq.DatabaseEndpoint.Type, "type", `NOTE: if want type to default to some value set the server then an effective_type field OR make this field REQUIRED. Supported values: [READ_ONLY, READ_WRITE]`)

	cmd.Use = "create-database-endpoint PROJECT_ID BRANCH_ID"
	cmd.Short = `Create a Database Endpoint.`
	cmd.Long = `Create a Database Endpoint.`

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
			diags := createDatabaseEndpointJson.Unmarshal(&createDatabaseEndpointReq.DatabaseEndpoint)
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
		createDatabaseEndpointReq.ProjectId = args[0]
		createDatabaseEndpointReq.BranchId = args[1]

		response, err := w.DatabaseProject.CreateDatabaseEndpoint(ctx, createDatabaseEndpointReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createDatabaseEndpointOverrides {
		fn(cmd, &createDatabaseEndpointReq)
	}

	return cmd
}

// start create-database-project command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createDatabaseProjectOverrides []func(
	*cobra.Command,
	*database.CreateDatabaseProjectRequest,
)

func newCreateDatabaseProject() *cobra.Command {
	cmd := &cobra.Command{}

	var createDatabaseProjectReq database.CreateDatabaseProjectRequest
	createDatabaseProjectReq.DatabaseProject = database.DatabaseProject{}
	var createDatabaseProjectJson flags.JsonFlag

	cmd.Flags().Var(&createDatabaseProjectJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createDatabaseProjectReq.DatabaseProject.BudgetPolicyId, "budget-policy-id", createDatabaseProjectReq.DatabaseProject.BudgetPolicyId, `The desired budget policy to associate with the instance.`)
	// TODO: array: custom_tags
	// TODO: complex arg: default_endpoint_settings
	cmd.Flags().StringVar(&createDatabaseProjectReq.DatabaseProject.DisplayName, "display-name", createDatabaseProjectReq.DatabaseProject.DisplayName, `Human-readable project name.`)
	cmd.Flags().StringVar(&createDatabaseProjectReq.DatabaseProject.HistoryRetentionDuration, "history-retention-duration", createDatabaseProjectReq.DatabaseProject.HistoryRetentionDuration, `The number of seconds to retain the shared history for point in time recovery for all branches in this project.`)
	cmd.Flags().IntVar(&createDatabaseProjectReq.DatabaseProject.PgVersion, "pg-version", createDatabaseProjectReq.DatabaseProject.PgVersion, `The major Postgres version number.`)
	cmd.Flags().StringVar(&createDatabaseProjectReq.DatabaseProject.ProjectId, "project-id", createDatabaseProjectReq.DatabaseProject.ProjectId, ``)
	// TODO: complex arg: settings

	cmd.Use = "create-database-project"
	cmd.Short = `Create a Database Project.`
	cmd.Long = `Create a Database Project.`

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
			diags := createDatabaseProjectJson.Unmarshal(&createDatabaseProjectReq.DatabaseProject)
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

		response, err := w.DatabaseProject.CreateDatabaseProject(ctx, createDatabaseProjectReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createDatabaseProjectOverrides {
		fn(cmd, &createDatabaseProjectReq)
	}

	return cmd
}

// start delete-database-branch command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteDatabaseBranchOverrides []func(
	*cobra.Command,
	*database.DeleteDatabaseBranchRequest,
)

func newDeleteDatabaseBranch() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteDatabaseBranchReq database.DeleteDatabaseBranchRequest

	cmd.Use = "delete-database-branch PROJECT_ID BRANCH_ID"
	cmd.Short = `Delete a Database Branch.`
	cmd.Long = `Delete a Database Branch.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteDatabaseBranchReq.ProjectId = args[0]
		deleteDatabaseBranchReq.BranchId = args[1]

		err = w.DatabaseProject.DeleteDatabaseBranch(ctx, deleteDatabaseBranchReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteDatabaseBranchOverrides {
		fn(cmd, &deleteDatabaseBranchReq)
	}

	return cmd
}

// start delete-database-endpoint command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteDatabaseEndpointOverrides []func(
	*cobra.Command,
	*database.DeleteDatabaseEndpointRequest,
)

func newDeleteDatabaseEndpoint() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteDatabaseEndpointReq database.DeleteDatabaseEndpointRequest

	cmd.Use = "delete-database-endpoint PROJECT_ID BRANCH_ID ENDPOINT_ID"
	cmd.Short = `Delete a Database Endpoint.`
	cmd.Long = `Delete a Database Endpoint.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteDatabaseEndpointReq.ProjectId = args[0]
		deleteDatabaseEndpointReq.BranchId = args[1]
		deleteDatabaseEndpointReq.EndpointId = args[2]

		err = w.DatabaseProject.DeleteDatabaseEndpoint(ctx, deleteDatabaseEndpointReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteDatabaseEndpointOverrides {
		fn(cmd, &deleteDatabaseEndpointReq)
	}

	return cmd
}

// start delete-database-project command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteDatabaseProjectOverrides []func(
	*cobra.Command,
	*database.DeleteDatabaseProjectRequest,
)

func newDeleteDatabaseProject() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteDatabaseProjectReq database.DeleteDatabaseProjectRequest

	cmd.Use = "delete-database-project PROJECT_ID"
	cmd.Short = `Delete a Database Project.`
	cmd.Long = `Delete a Database Project.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteDatabaseProjectReq.ProjectId = args[0]

		err = w.DatabaseProject.DeleteDatabaseProject(ctx, deleteDatabaseProjectReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteDatabaseProjectOverrides {
		fn(cmd, &deleteDatabaseProjectReq)
	}

	return cmd
}

// start get-database-branch command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getDatabaseBranchOverrides []func(
	*cobra.Command,
	*database.GetDatabaseBranchRequest,
)

func newGetDatabaseBranch() *cobra.Command {
	cmd := &cobra.Command{}

	var getDatabaseBranchReq database.GetDatabaseBranchRequest

	cmd.Use = "get-database-branch PROJECT_ID BRANCH_ID"
	cmd.Short = `Get a Database Branch.`
	cmd.Long = `Get a Database Branch.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getDatabaseBranchReq.ProjectId = args[0]
		getDatabaseBranchReq.BranchId = args[1]

		response, err := w.DatabaseProject.GetDatabaseBranch(ctx, getDatabaseBranchReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getDatabaseBranchOverrides {
		fn(cmd, &getDatabaseBranchReq)
	}

	return cmd
}

// start get-database-endpoint command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getDatabaseEndpointOverrides []func(
	*cobra.Command,
	*database.GetDatabaseEndpointRequest,
)

func newGetDatabaseEndpoint() *cobra.Command {
	cmd := &cobra.Command{}

	var getDatabaseEndpointReq database.GetDatabaseEndpointRequest

	cmd.Use = "get-database-endpoint PROJECT_ID BRANCH_ID ENDPOINT_ID"
	cmd.Short = `Get a Database Endpoint.`
	cmd.Long = `Get a Database Endpoint.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getDatabaseEndpointReq.ProjectId = args[0]
		getDatabaseEndpointReq.BranchId = args[1]
		getDatabaseEndpointReq.EndpointId = args[2]

		response, err := w.DatabaseProject.GetDatabaseEndpoint(ctx, getDatabaseEndpointReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getDatabaseEndpointOverrides {
		fn(cmd, &getDatabaseEndpointReq)
	}

	return cmd
}

// start get-database-project command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getDatabaseProjectOverrides []func(
	*cobra.Command,
	*database.GetDatabaseProjectRequest,
)

func newGetDatabaseProject() *cobra.Command {
	cmd := &cobra.Command{}

	var getDatabaseProjectReq database.GetDatabaseProjectRequest

	cmd.Use = "get-database-project PROJECT_ID"
	cmd.Short = `Get a Database Project.`
	cmd.Long = `Get a Database Project.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getDatabaseProjectReq.ProjectId = args[0]

		response, err := w.DatabaseProject.GetDatabaseProject(ctx, getDatabaseProjectReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getDatabaseProjectOverrides {
		fn(cmd, &getDatabaseProjectReq)
	}

	return cmd
}

// start list-database-branches command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listDatabaseBranchesOverrides []func(
	*cobra.Command,
	*database.ListDatabaseBranchesRequest,
)

func newListDatabaseBranches() *cobra.Command {
	cmd := &cobra.Command{}

	var listDatabaseBranchesReq database.ListDatabaseBranchesRequest

	cmd.Flags().IntVar(&listDatabaseBranchesReq.PageSize, "page-size", listDatabaseBranchesReq.PageSize, `Upper bound for items returned.`)
	cmd.Flags().StringVar(&listDatabaseBranchesReq.PageToken, "page-token", listDatabaseBranchesReq.PageToken, `Pagination token to go to the next page of Database Branches.`)

	cmd.Use = "list-database-branches PROJECT_ID"
	cmd.Short = `List Database Branches.`
	cmd.Long = `List Database Branches.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listDatabaseBranchesReq.ProjectId = args[0]

		response := w.DatabaseProject.ListDatabaseBranches(ctx, listDatabaseBranchesReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listDatabaseBranchesOverrides {
		fn(cmd, &listDatabaseBranchesReq)
	}

	return cmd
}

// start list-database-endpoints command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listDatabaseEndpointsOverrides []func(
	*cobra.Command,
	*database.ListDatabaseEndpointsRequest,
)

func newListDatabaseEndpoints() *cobra.Command {
	cmd := &cobra.Command{}

	var listDatabaseEndpointsReq database.ListDatabaseEndpointsRequest

	cmd.Flags().IntVar(&listDatabaseEndpointsReq.PageSize, "page-size", listDatabaseEndpointsReq.PageSize, `Upper bound for items returned.`)
	cmd.Flags().StringVar(&listDatabaseEndpointsReq.PageToken, "page-token", listDatabaseEndpointsReq.PageToken, `Pagination token to go to the next page of Database Endpoints.`)

	cmd.Use = "list-database-endpoints PROJECT_ID BRANCH_ID"
	cmd.Short = `List Database Endpoints.`
	cmd.Long = `List Database Endpoints.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listDatabaseEndpointsReq.ProjectId = args[0]
		listDatabaseEndpointsReq.BranchId = args[1]

		response := w.DatabaseProject.ListDatabaseEndpoints(ctx, listDatabaseEndpointsReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listDatabaseEndpointsOverrides {
		fn(cmd, &listDatabaseEndpointsReq)
	}

	return cmd
}

// start list-database-projects command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listDatabaseProjectsOverrides []func(
	*cobra.Command,
	*database.ListDatabaseProjectsRequest,
)

func newListDatabaseProjects() *cobra.Command {
	cmd := &cobra.Command{}

	var listDatabaseProjectsReq database.ListDatabaseProjectsRequest

	cmd.Flags().IntVar(&listDatabaseProjectsReq.PageSize, "page-size", listDatabaseProjectsReq.PageSize, `Upper bound for items returned.`)
	cmd.Flags().StringVar(&listDatabaseProjectsReq.PageToken, "page-token", listDatabaseProjectsReq.PageToken, `Pagination token to go to the next page of Database Projects.`)

	cmd.Use = "list-database-projects"
	cmd.Short = `List Database Instances.`
	cmd.Long = `List Database Instances.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.DatabaseProject.ListDatabaseProjects(ctx, listDatabaseProjectsReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listDatabaseProjectsOverrides {
		fn(cmd, &listDatabaseProjectsReq)
	}

	return cmd
}

// start restart-database-endpoint command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var restartDatabaseEndpointOverrides []func(
	*cobra.Command,
	*database.RestartDatabaseEndpointRequest,
)

func newRestartDatabaseEndpoint() *cobra.Command {
	cmd := &cobra.Command{}

	var restartDatabaseEndpointReq database.RestartDatabaseEndpointRequest

	cmd.Use = "restart-database-endpoint PROJECT_ID BRANCH_ID ENDPOINT_ID"
	cmd.Short = `Restart a Database Endpoint.`
	cmd.Long = `Restart a Database Endpoint.
  
  Restart a Database Endpoint. TODO: should return
  databricks.longrunning.Operation`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		restartDatabaseEndpointReq.ProjectId = args[0]
		restartDatabaseEndpointReq.BranchId = args[1]
		restartDatabaseEndpointReq.EndpointId = args[2]

		response, err := w.DatabaseProject.RestartDatabaseEndpoint(ctx, restartDatabaseEndpointReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range restartDatabaseEndpointOverrides {
		fn(cmd, &restartDatabaseEndpointReq)
	}

	return cmd
}

// start update-database-branch command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateDatabaseBranchOverrides []func(
	*cobra.Command,
	*database.UpdateDatabaseBranchRequest,
)

func newUpdateDatabaseBranch() *cobra.Command {
	cmd := &cobra.Command{}

	var updateDatabaseBranchReq database.UpdateDatabaseBranchRequest
	updateDatabaseBranchReq.DatabaseBranch = database.DatabaseBranch{}
	var updateDatabaseBranchJson flags.JsonFlag

	cmd.Flags().Var(&updateDatabaseBranchJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateDatabaseBranchReq.DatabaseBranch.BranchId, "branch-id", updateDatabaseBranchReq.DatabaseBranch.BranchId, ``)
	cmd.Flags().BoolVar(&updateDatabaseBranchReq.DatabaseBranch.Default, "default", updateDatabaseBranchReq.DatabaseBranch.Default, `Whether the branch is the project's default branch.`)
	cmd.Flags().BoolVar(&updateDatabaseBranchReq.DatabaseBranch.IsProtected, "is-protected", updateDatabaseBranchReq.DatabaseBranch.IsProtected, `Whether the branch is protected.`)
	cmd.Flags().StringVar(&updateDatabaseBranchReq.DatabaseBranch.ParentId, "parent-id", updateDatabaseBranchReq.DatabaseBranch.ParentId, `The id of the parent branch.`)
	cmd.Flags().StringVar(&updateDatabaseBranchReq.DatabaseBranch.ParentLsn, "parent-lsn", updateDatabaseBranchReq.DatabaseBranch.ParentLsn, `The Log Sequence Number (LSN) on the parent branch from which this branch was created.`)
	cmd.Flags().StringVar(&updateDatabaseBranchReq.DatabaseBranch.ParentTime, "parent-time", updateDatabaseBranchReq.DatabaseBranch.ParentTime, `The point in time on the parent branch from which this branch was created.`)

	cmd.Use = "update-database-branch PROJECT_ID BRANCH_ID UPDATE_MASK"
	cmd.Short = `Update a Database Branch.`
	cmd.Long = `Update a Database Branch.

  Arguments:
    PROJECT_ID: 
    BRANCH_ID: 
    UPDATE_MASK: The list of fields to update. If unspecified, all fields will be updated
      when possible.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateDatabaseBranchJson.Unmarshal(&updateDatabaseBranchReq.DatabaseBranch)
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
		updateDatabaseBranchReq.ProjectId = args[0]
		updateDatabaseBranchReq.BranchId = args[1]
		updateDatabaseBranchReq.UpdateMask = args[2]

		response, err := w.DatabaseProject.UpdateDatabaseBranch(ctx, updateDatabaseBranchReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateDatabaseBranchOverrides {
		fn(cmd, &updateDatabaseBranchReq)
	}

	return cmd
}

// start update-database-endpoint command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateDatabaseEndpointOverrides []func(
	*cobra.Command,
	*database.UpdateDatabaseEndpointRequest,
)

func newUpdateDatabaseEndpoint() *cobra.Command {
	cmd := &cobra.Command{}

	var updateDatabaseEndpointReq database.UpdateDatabaseEndpointRequest
	updateDatabaseEndpointReq.DatabaseEndpoint = database.DatabaseEndpoint{}
	var updateDatabaseEndpointJson flags.JsonFlag

	cmd.Flags().Var(&updateDatabaseEndpointJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().Float64Var(&updateDatabaseEndpointReq.DatabaseEndpoint.AutoscalingLimitMaxCu, "autoscaling-limit-max-cu", updateDatabaseEndpointReq.DatabaseEndpoint.AutoscalingLimitMaxCu, `The maximum number of Compute Units.`)
	cmd.Flags().Float64Var(&updateDatabaseEndpointReq.DatabaseEndpoint.AutoscalingLimitMinCu, "autoscaling-limit-min-cu", updateDatabaseEndpointReq.DatabaseEndpoint.AutoscalingLimitMinCu, `The minimum number of Compute Units.`)
	cmd.Flags().BoolVar(&updateDatabaseEndpointReq.DatabaseEndpoint.Disabled, "disabled", updateDatabaseEndpointReq.DatabaseEndpoint.Disabled, `Whether to restrict connections to the compute endpoint.`)
	cmd.Flags().StringVar(&updateDatabaseEndpointReq.DatabaseEndpoint.EndpointId, "endpoint-id", updateDatabaseEndpointReq.DatabaseEndpoint.EndpointId, ``)
	cmd.Flags().Var(&updateDatabaseEndpointReq.DatabaseEndpoint.PoolerMode, "pooler-mode", `Supported values: [TRANSACTION]`)
	// TODO: complex arg: settings
	cmd.Flags().StringVar(&updateDatabaseEndpointReq.DatabaseEndpoint.SuspendTimeoutDuration, "suspend-timeout-duration", updateDatabaseEndpointReq.DatabaseEndpoint.SuspendTimeoutDuration, `Duration of inactivity after which the compute endpoint is automatically suspended.`)
	cmd.Flags().Var(&updateDatabaseEndpointReq.DatabaseEndpoint.Type, "type", `NOTE: if want type to default to some value set the server then an effective_type field OR make this field REQUIRED. Supported values: [READ_ONLY, READ_WRITE]`)

	cmd.Use = "update-database-endpoint PROJECT_ID BRANCH_ID ENDPOINT_ID UPDATE_MASK"
	cmd.Short = `Update a Database Endpoint.`
	cmd.Long = `Update a Database Endpoint.
  
  Update a Database Endpoint. TODO: should return
  databricks.longrunning.Operation {

  Arguments:
    PROJECT_ID: 
    BRANCH_ID: 
    ENDPOINT_ID: 
    UPDATE_MASK: The list of fields to update. If unspecified, all fields will be updated
      when possible.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(4)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateDatabaseEndpointJson.Unmarshal(&updateDatabaseEndpointReq.DatabaseEndpoint)
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
		updateDatabaseEndpointReq.ProjectId = args[0]
		updateDatabaseEndpointReq.BranchId = args[1]
		updateDatabaseEndpointReq.EndpointId = args[2]
		updateDatabaseEndpointReq.UpdateMask = args[3]

		response, err := w.DatabaseProject.UpdateDatabaseEndpoint(ctx, updateDatabaseEndpointReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateDatabaseEndpointOverrides {
		fn(cmd, &updateDatabaseEndpointReq)
	}

	return cmd
}

// start update-database-project command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateDatabaseProjectOverrides []func(
	*cobra.Command,
	*database.UpdateDatabaseProjectRequest,
)

func newUpdateDatabaseProject() *cobra.Command {
	cmd := &cobra.Command{}

	var updateDatabaseProjectReq database.UpdateDatabaseProjectRequest
	updateDatabaseProjectReq.DatabaseProject = database.DatabaseProject{}
	var updateDatabaseProjectJson flags.JsonFlag

	cmd.Flags().Var(&updateDatabaseProjectJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateDatabaseProjectReq.DatabaseProject.BudgetPolicyId, "budget-policy-id", updateDatabaseProjectReq.DatabaseProject.BudgetPolicyId, `The desired budget policy to associate with the instance.`)
	// TODO: array: custom_tags
	// TODO: complex arg: default_endpoint_settings
	cmd.Flags().StringVar(&updateDatabaseProjectReq.DatabaseProject.DisplayName, "display-name", updateDatabaseProjectReq.DatabaseProject.DisplayName, `Human-readable project name.`)
	cmd.Flags().StringVar(&updateDatabaseProjectReq.DatabaseProject.HistoryRetentionDuration, "history-retention-duration", updateDatabaseProjectReq.DatabaseProject.HistoryRetentionDuration, `The number of seconds to retain the shared history for point in time recovery for all branches in this project.`)
	cmd.Flags().IntVar(&updateDatabaseProjectReq.DatabaseProject.PgVersion, "pg-version", updateDatabaseProjectReq.DatabaseProject.PgVersion, `The major Postgres version number.`)
	cmd.Flags().StringVar(&updateDatabaseProjectReq.DatabaseProject.ProjectId, "project-id", updateDatabaseProjectReq.DatabaseProject.ProjectId, ``)
	// TODO: complex arg: settings

	cmd.Use = "update-database-project PROJECT_ID UPDATE_MASK"
	cmd.Short = `Update a Database Project.`
	cmd.Long = `Update a Database Project.

  Arguments:
    PROJECT_ID: 
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
			diags := updateDatabaseProjectJson.Unmarshal(&updateDatabaseProjectReq.DatabaseProject)
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
		updateDatabaseProjectReq.ProjectId = args[0]
		updateDatabaseProjectReq.UpdateMask = args[1]

		response, err := w.DatabaseProject.UpdateDatabaseProject(ctx, updateDatabaseProjectReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateDatabaseProjectOverrides {
		fn(cmd, &updateDatabaseProjectReq)
	}

	return cmd
}

// end service DatabaseProject
