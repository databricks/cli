// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package permissions

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "permissions",
		Short: `Permissions API are used to create read, write, edit, update and manage access for various users on different objects and endpoints.`,
		Long: `Permissions API are used to create read, write, edit, update and manage access
  for various users on different objects and endpoints. * **[Apps
  permissions](:service:apps)** — Manage which users can manage or use apps. *
  **[Cluster permissions](:service:clusters)** — Manage which users can
  manage, restart, or attach to clusters. * **[Cluster policy
  permissions](:service:clusterpolicies)** — Manage which users can use
  cluster policies. * **[Delta Live Tables pipeline
  permissions](:service:pipelines)** — Manage which users can view, manage,
  run, cancel, or own a Delta Live Tables pipeline. * **[Job
  permissions](:service:jobs)** — Manage which users can view, manage,
  trigger, cancel, or own a job. * **[MLflow experiment
  permissions](:service:experiments)** — Manage which users can read, edit, or
  manage MLflow experiments. * **[MLflow registered model
  permissions](:service:modelregistry)** — Manage which users can read, edit,
  or manage MLflow registered models. * **[Instance Pool
  permissions](:service:instancepools)** — Manage which users can manage or
  attach to pools. * **[Repo permissions](repos)** — Manage which users can
  read, run, edit, or manage a repo. * **[Serving endpoint
  permissions](:service:servingendpoints)** — Manage which users can view,
  query, or manage a serving endpoint. * **[SQL warehouse
  permissions](:service:warehouses)** — Manage which users can use or manage
  SQL warehouses. * **[Token permissions](:service:tokenmanagement)** — Manage
  which users can create or use tokens. * **[Workspace object
  permissions](:service:workspace)** — Manage which users can read, run, edit,
  or manage alerts, dbsql-dashboards, directories, files, notebooks and queries.
  For the mapping of the required permissions for specific actions or abilities
  and other important information, see [Access Control]. Note that to manage
  access control on service principals, use **[Account Access Control
  Proxy](:service:accountaccesscontrolproxy)**.
  
  [Access Control]: https://docs.databricks.com/security/auth-authz/access-control/index.html`,
		GroupID: "iam",
		Annotations: map[string]string{
			"package": "iam",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newGet())
	cmd.AddCommand(newGetPermissionLevels())
	cmd.AddCommand(newSet())
	cmd.AddCommand(newUpdate())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*iam.GetPermissionRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq iam.GetPermissionRequest

	cmd.Use = "get REQUEST_OBJECT_TYPE REQUEST_OBJECT_ID"
	cmd.Short = `Get object permissions.`
	cmd.Long = `Get object permissions.
  
  Gets the permissions of an object. Objects can inherit permissions from their
  parent objects or root object.

  Arguments:
    REQUEST_OBJECT_TYPE: The type of the request object. Can be one of the following: alerts,
      alertsv2, authorization, clusters, cluster-policies, dashboards,
      dbsql-dashboards, directories, experiments, files, instance-pools, jobs,
      notebooks, pipelines, queries, registered-models, repos,
      serving-endpoints, or warehouses.
    REQUEST_OBJECT_ID: The id of the request object.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getReq.RequestObjectType = args[0]
		getReq.RequestObjectId = args[1]

		response, err := w.Permissions.Get(ctx, getReq)
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

// start get-permission-levels command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPermissionLevelsOverrides []func(
	*cobra.Command,
	*iam.GetPermissionLevelsRequest,
)

func newGetPermissionLevels() *cobra.Command {
	cmd := &cobra.Command{}

	var getPermissionLevelsReq iam.GetPermissionLevelsRequest

	cmd.Use = "get-permission-levels REQUEST_OBJECT_TYPE REQUEST_OBJECT_ID"
	cmd.Short = `Get object permission levels.`
	cmd.Long = `Get object permission levels.
  
  Gets the permission levels that a user can have on an object.

  Arguments:
    REQUEST_OBJECT_TYPE: The type of the request object. Can be one of the following: alerts,
      alertsv2, authorization, clusters, cluster-policies, dashboards,
      dbsql-dashboards, directories, experiments, files, instance-pools, jobs,
      notebooks, pipelines, queries, registered-models, repos,
      serving-endpoints, or warehouses.
    REQUEST_OBJECT_ID: `

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getPermissionLevelsReq.RequestObjectType = args[0]
		getPermissionLevelsReq.RequestObjectId = args[1]

		response, err := w.Permissions.GetPermissionLevels(ctx, getPermissionLevelsReq)
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

// start set command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var setOverrides []func(
	*cobra.Command,
	*iam.SetObjectPermissions,
)

func newSet() *cobra.Command {
	cmd := &cobra.Command{}

	var setReq iam.SetObjectPermissions
	var setJson flags.JsonFlag

	cmd.Flags().Var(&setJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "set REQUEST_OBJECT_TYPE REQUEST_OBJECT_ID"
	cmd.Short = `Set object permissions.`
	cmd.Long = `Set object permissions.
  
  Sets permissions on an object, replacing existing permissions if they exist.
  Deletes all direct permissions if none are specified. Objects can inherit
  permissions from their parent objects or root object.

  Arguments:
    REQUEST_OBJECT_TYPE: The type of the request object. Can be one of the following: alerts,
      alertsv2, authorization, clusters, cluster-policies, dashboards,
      dbsql-dashboards, directories, experiments, files, instance-pools, jobs,
      notebooks, pipelines, queries, registered-models, repos,
      serving-endpoints, or warehouses.
    REQUEST_OBJECT_ID: The id of the request object.`

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
			diags := setJson.Unmarshal(&setReq)
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
		setReq.RequestObjectType = args[0]
		setReq.RequestObjectId = args[1]

		response, err := w.Permissions.Set(ctx, setReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range setOverrides {
		fn(cmd, &setReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*iam.UpdateObjectPermissions,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq iam.UpdateObjectPermissions
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "update REQUEST_OBJECT_TYPE REQUEST_OBJECT_ID"
	cmd.Short = `Update object permissions.`
	cmd.Long = `Update object permissions.
  
  Updates the permissions on an object. Objects can inherit permissions from
  their parent objects or root object.

  Arguments:
    REQUEST_OBJECT_TYPE: The type of the request object. Can be one of the following: alerts,
      alertsv2, authorization, clusters, cluster-policies, dashboards,
      dbsql-dashboards, directories, experiments, files, instance-pools, jobs,
      notebooks, pipelines, queries, registered-models, repos,
      serving-endpoints, or warehouses.
    REQUEST_OBJECT_ID: The id of the request object.`

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
			diags := updateJson.Unmarshal(&updateReq)
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
		updateReq.RequestObjectType = args[0]
		updateReq.RequestObjectId = args[1]

		response, err := w.Permissions.Update(ctx, updateReq)
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

// end service Permissions
