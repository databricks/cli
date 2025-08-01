// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package serving_endpoints

import (
	"fmt"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
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
		Use:   "serving-endpoints",
		Short: `The Serving Endpoints API allows you to create, update, and delete model serving endpoints.`,
		Long: `The Serving Endpoints API allows you to create, update, and delete model
  serving endpoints.
  
  You can use a serving endpoint to serve models from the Databricks Model
  Registry or from Unity Catalog. Endpoints expose the underlying models as
  scalable REST API endpoints using serverless compute. This means the endpoints
  and associated compute resources are fully managed by Databricks and will not
  appear in your cloud account. A serving endpoint can consist of one or more
  MLflow models from the Databricks Model Registry, called served entities. A
  serving endpoint can have at most ten served entities. You can configure
  traffic settings to define how requests should be routed to your served
  entities behind an endpoint. Additionally, you can configure the scale of
  resources that should be applied to each served entity.`,
		GroupID: "serving",
		Annotations: map[string]string{
			"package": "serving",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newBuildLogs())
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newCreateProvisionedThroughputEndpoint())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newExportMetrics())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newGetOpenApi())
	cmd.AddCommand(newGetPermissionLevels())
	cmd.AddCommand(newGetPermissions())
	cmd.AddCommand(newHttpRequest())
	cmd.AddCommand(newList())
	cmd.AddCommand(newLogs())
	cmd.AddCommand(newPatch())
	cmd.AddCommand(newPut())
	cmd.AddCommand(newPutAiGateway())
	cmd.AddCommand(newQuery())
	cmd.AddCommand(newSetPermissions())
	cmd.AddCommand(newUpdateConfig())
	cmd.AddCommand(newUpdatePermissions())
	cmd.AddCommand(newUpdateProvisionedThroughputEndpointConfig())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start build-logs command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var buildLogsOverrides []func(
	*cobra.Command,
	*serving.BuildLogsRequest,
)

func newBuildLogs() *cobra.Command {
	cmd := &cobra.Command{}

	var buildLogsReq serving.BuildLogsRequest

	cmd.Use = "build-logs NAME SERVED_MODEL_NAME"
	cmd.Short = `Get build logs for a served model.`
	cmd.Long = `Get build logs for a served model.
  
  Retrieves the build logs associated with the provided served model.

  Arguments:
    NAME: The name of the serving endpoint that the served model belongs to. This
      field is required.
    SERVED_MODEL_NAME: The name of the served model that build logs will be retrieved for. This
      field is required.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		buildLogsReq.Name = args[0]
		buildLogsReq.ServedModelName = args[1]

		response, err := w.ServingEndpoints.BuildLogs(ctx, buildLogsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range buildLogsOverrides {
		fn(cmd, &buildLogsReq)
	}

	return cmd
}

// start create command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOverrides []func(
	*cobra.Command,
	*serving.CreateServingEndpoint,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq serving.CreateServingEndpoint
	var createJson flags.JsonFlag

	var createSkipWait bool
	var createTimeout time.Duration

	cmd.Flags().BoolVar(&createSkipWait, "no-wait", createSkipWait, `do not wait to reach NOT_UPDATING state`)
	cmd.Flags().DurationVar(&createTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach NOT_UPDATING state`)

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: ai_gateway
	cmd.Flags().StringVar(&createReq.BudgetPolicyId, "budget-policy-id", createReq.BudgetPolicyId, `The budget policy to be applied to the serving endpoint.`)
	// TODO: complex arg: config
	cmd.Flags().StringVar(&createReq.Description, "description", createReq.Description, ``)
	// TODO: array: rate_limits
	cmd.Flags().BoolVar(&createReq.RouteOptimized, "route-optimized", createReq.RouteOptimized, `Enable route optimization for the serving endpoint.`)
	// TODO: array: tags

	cmd.Use = "create NAME"
	cmd.Short = `Create a new serving endpoint.`
	cmd.Long = `Create a new serving endpoint.

  Arguments:
    NAME: The name of the serving endpoint. This field is required and must be
      unique across a Databricks workspace. An endpoint name can consist of
      alphanumeric characters, dashes, and underscores.`

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
			diags := createJson.Unmarshal(&createReq)
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
			createReq.Name = args[0]
		}

		wait, err := w.ServingEndpoints.Create(ctx, createReq)
		if err != nil {
			return err
		}
		if createSkipWait {
			return cmdio.Render(ctx, wait.Response)
		}
		spinner := cmdio.Spinner(ctx)
		info, err := wait.OnProgress(func(i *serving.ServingEndpointDetailed) {
			status := i.State.ConfigUpdate
			statusMessage := fmt.Sprintf("current status: %s", status)
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

// start create-provisioned-throughput-endpoint command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createProvisionedThroughputEndpointOverrides []func(
	*cobra.Command,
	*serving.CreatePtEndpointRequest,
)

func newCreateProvisionedThroughputEndpoint() *cobra.Command {
	cmd := &cobra.Command{}

	var createProvisionedThroughputEndpointReq serving.CreatePtEndpointRequest
	var createProvisionedThroughputEndpointJson flags.JsonFlag

	var createProvisionedThroughputEndpointSkipWait bool
	var createProvisionedThroughputEndpointTimeout time.Duration

	cmd.Flags().BoolVar(&createProvisionedThroughputEndpointSkipWait, "no-wait", createProvisionedThroughputEndpointSkipWait, `do not wait to reach NOT_UPDATING state`)
	cmd.Flags().DurationVar(&createProvisionedThroughputEndpointTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach NOT_UPDATING state`)

	cmd.Flags().Var(&createProvisionedThroughputEndpointJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: ai_gateway
	cmd.Flags().StringVar(&createProvisionedThroughputEndpointReq.BudgetPolicyId, "budget-policy-id", createProvisionedThroughputEndpointReq.BudgetPolicyId, `The budget policy associated with the endpoint.`)
	// TODO: array: tags

	cmd.Use = "create-provisioned-throughput-endpoint"
	cmd.Short = `Create a new PT serving endpoint.`
	cmd.Long = `Create a new PT serving endpoint.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createProvisionedThroughputEndpointJson.Unmarshal(&createProvisionedThroughputEndpointReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

		wait, err := w.ServingEndpoints.CreateProvisionedThroughputEndpoint(ctx, createProvisionedThroughputEndpointReq)
		if err != nil {
			return err
		}
		if createProvisionedThroughputEndpointSkipWait {
			return cmdio.Render(ctx, wait.Response)
		}
		spinner := cmdio.Spinner(ctx)
		info, err := wait.OnProgress(func(i *serving.ServingEndpointDetailed) {
			status := i.State.ConfigUpdate
			statusMessage := fmt.Sprintf("current status: %s", status)
			spinner <- statusMessage
		}).GetWithTimeout(createProvisionedThroughputEndpointTimeout)
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
	for _, fn := range createProvisionedThroughputEndpointOverrides {
		fn(cmd, &createProvisionedThroughputEndpointReq)
	}

	return cmd
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*serving.DeleteServingEndpointRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq serving.DeleteServingEndpointRequest

	cmd.Use = "delete NAME"
	cmd.Short = `Delete a serving endpoint.`
	cmd.Long = `Delete a serving endpoint.`

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

		err = w.ServingEndpoints.Delete(ctx, deleteReq)
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

// start export-metrics command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var exportMetricsOverrides []func(
	*cobra.Command,
	*serving.ExportMetricsRequest,
)

func newExportMetrics() *cobra.Command {
	cmd := &cobra.Command{}

	var exportMetricsReq serving.ExportMetricsRequest

	cmd.Use = "export-metrics NAME"
	cmd.Short = `Get metrics of a serving endpoint.`
	cmd.Long = `Get metrics of a serving endpoint.
  
  Retrieves the metrics associated with the provided serving endpoint in either
  Prometheus or OpenMetrics exposition format.

  Arguments:
    NAME: The name of the serving endpoint to retrieve metrics for. This field is
      required.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		exportMetricsReq.Name = args[0]

		response, err := w.ServingEndpoints.ExportMetrics(ctx, exportMetricsReq)
		if err != nil {
			return err
		}
		defer response.Contents.Close()
		return cmdio.Render(ctx, response.Contents)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range exportMetricsOverrides {
		fn(cmd, &exportMetricsReq)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*serving.GetServingEndpointRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq serving.GetServingEndpointRequest

	cmd.Use = "get NAME"
	cmd.Short = `Get a single serving endpoint.`
	cmd.Long = `Get a single serving endpoint.
  
  Retrieves the details for a single serving endpoint.

  Arguments:
    NAME: The name of the serving endpoint. This field is required.`

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

		response, err := w.ServingEndpoints.Get(ctx, getReq)
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

// start get-open-api command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOpenApiOverrides []func(
	*cobra.Command,
	*serving.GetOpenApiRequest,
)

func newGetOpenApi() *cobra.Command {
	cmd := &cobra.Command{}

	var getOpenApiReq serving.GetOpenApiRequest

	cmd.Use = "get-open-api NAME"
	cmd.Short = `Get the schema for a serving endpoint.`
	cmd.Long = `Get the schema for a serving endpoint.
  
  Get the query schema of the serving endpoint in OpenAPI format. The schema
  contains information for the supported paths, input and output format and
  datatypes.

  Arguments:
    NAME: The name of the serving endpoint that the served model belongs to. This
      field is required.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getOpenApiReq.Name = args[0]

		response, err := w.ServingEndpoints.GetOpenApi(ctx, getOpenApiReq)
		if err != nil {
			return err
		}
		defer response.Contents.Close()
		return cmdio.Render(ctx, response.Contents)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getOpenApiOverrides {
		fn(cmd, &getOpenApiReq)
	}

	return cmd
}

// start get-permission-levels command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPermissionLevelsOverrides []func(
	*cobra.Command,
	*serving.GetServingEndpointPermissionLevelsRequest,
)

func newGetPermissionLevels() *cobra.Command {
	cmd := &cobra.Command{}

	var getPermissionLevelsReq serving.GetServingEndpointPermissionLevelsRequest

	cmd.Use = "get-permission-levels SERVING_ENDPOINT_ID"
	cmd.Short = `Get serving endpoint permission levels.`
	cmd.Long = `Get serving endpoint permission levels.
  
  Gets the permission levels that a user can have on an object.

  Arguments:
    SERVING_ENDPOINT_ID: The serving endpoint for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getPermissionLevelsReq.ServingEndpointId = args[0]

		response, err := w.ServingEndpoints.GetPermissionLevels(ctx, getPermissionLevelsReq)
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
	*serving.GetServingEndpointPermissionsRequest,
)

func newGetPermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var getPermissionsReq serving.GetServingEndpointPermissionsRequest

	cmd.Use = "get-permissions SERVING_ENDPOINT_ID"
	cmd.Short = `Get serving endpoint permissions.`
	cmd.Long = `Get serving endpoint permissions.
  
  Gets the permissions of a serving endpoint. Serving endpoints can inherit
  permissions from their root object.

  Arguments:
    SERVING_ENDPOINT_ID: The serving endpoint for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getPermissionsReq.ServingEndpointId = args[0]

		response, err := w.ServingEndpoints.GetPermissions(ctx, getPermissionsReq)
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

// start http-request command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var httpRequestOverrides []func(
	*cobra.Command,
	*serving.ExternalFunctionRequest,
)

func newHttpRequest() *cobra.Command {
	cmd := &cobra.Command{}

	var httpRequestReq serving.ExternalFunctionRequest

	cmd.Flags().StringVar(&httpRequestReq.Headers, "headers", httpRequestReq.Headers, `Additional headers for the request.`)
	cmd.Flags().StringVar(&httpRequestReq.Json, "json", httpRequestReq.Json, `The JSON payload to send in the request body.`)
	cmd.Flags().StringVar(&httpRequestReq.Params, "params", httpRequestReq.Params, `Query parameters for the request.`)

	cmd.Use = "http-request CONNECTION_NAME METHOD PATH"
	cmd.Short = `Make external services call using the credentials stored in UC Connection.`
	cmd.Long = `Make external services call using the credentials stored in UC Connection.

  Arguments:
    CONNECTION_NAME: The connection name to use. This is required to identify the external
      connection.
    METHOD: The HTTP method to use (e.g., 'GET', 'POST'). 
      Supported values: [DELETE, GET, PATCH, POST, PUT]
    PATH: The relative path for the API endpoint. This is required.`

	// This command is being previewed; hide from help output.
	cmd.Hidden = true

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		httpRequestReq.ConnectionName = args[0]
		_, err = fmt.Sscan(args[1], &httpRequestReq.Method)
		if err != nil {
			return fmt.Errorf("invalid METHOD: %s", args[1])
		}
		httpRequestReq.Path = args[2]

		response, err := w.ServingEndpoints.HttpRequest(ctx, httpRequestReq)
		if err != nil {
			return err
		}
		defer response.Contents.Close()
		return cmdio.Render(ctx, response.Contents)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range httpRequestOverrides {
		fn(cmd, &httpRequestReq)
	}

	return cmd
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "list"
	cmd.Short = `Get all serving endpoints.`
	cmd.Long = `Get all serving endpoints.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		response := w.ServingEndpoints.List(ctx)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOverrides {
		fn(cmd)
	}

	return cmd
}

// start logs command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var logsOverrides []func(
	*cobra.Command,
	*serving.LogsRequest,
)

func newLogs() *cobra.Command {
	cmd := &cobra.Command{}

	var logsReq serving.LogsRequest

	cmd.Use = "logs NAME SERVED_MODEL_NAME"
	cmd.Short = `Get the latest logs for a served model.`
	cmd.Long = `Get the latest logs for a served model.
  
  Retrieves the service logs associated with the provided served model.

  Arguments:
    NAME: The name of the serving endpoint that the served model belongs to. This
      field is required.
    SERVED_MODEL_NAME: The name of the served model that logs will be retrieved for. This field
      is required.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		logsReq.Name = args[0]
		logsReq.ServedModelName = args[1]

		response, err := w.ServingEndpoints.Logs(ctx, logsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range logsOverrides {
		fn(cmd, &logsReq)
	}

	return cmd
}

// start patch command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var patchOverrides []func(
	*cobra.Command,
	*serving.PatchServingEndpointTags,
)

func newPatch() *cobra.Command {
	cmd := &cobra.Command{}

	var patchReq serving.PatchServingEndpointTags
	var patchJson flags.JsonFlag

	cmd.Flags().Var(&patchJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: add_tags
	// TODO: array: delete_tags

	cmd.Use = "patch NAME"
	cmd.Short = `Update tags of a serving endpoint.`
	cmd.Long = `Update tags of a serving endpoint.
  
  Used to batch add and delete tags from a serving endpoint with a single API
  call.

  Arguments:
    NAME: The name of the serving endpoint who's tags to patch. This field is
      required.`

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
			diags := patchJson.Unmarshal(&patchReq)
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
		patchReq.Name = args[0]

		response, err := w.ServingEndpoints.Patch(ctx, patchReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range patchOverrides {
		fn(cmd, &patchReq)
	}

	return cmd
}

// start put command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var putOverrides []func(
	*cobra.Command,
	*serving.PutRequest,
)

func newPut() *cobra.Command {
	cmd := &cobra.Command{}

	var putReq serving.PutRequest
	var putJson flags.JsonFlag

	cmd.Flags().Var(&putJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: rate_limits

	cmd.Use = "put NAME"
	cmd.Short = `Update rate limits of a serving endpoint.`
	cmd.Long = `Update rate limits of a serving endpoint.
  
  Deprecated: Please use AI Gateway to manage rate limits instead.

  Arguments:
    NAME: The name of the serving endpoint whose rate limits are being updated. This
      field is required.`

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
			diags := putJson.Unmarshal(&putReq)
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
		putReq.Name = args[0]

		response, err := w.ServingEndpoints.Put(ctx, putReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range putOverrides {
		fn(cmd, &putReq)
	}

	return cmd
}

// start put-ai-gateway command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var putAiGatewayOverrides []func(
	*cobra.Command,
	*serving.PutAiGatewayRequest,
)

func newPutAiGateway() *cobra.Command {
	cmd := &cobra.Command{}

	var putAiGatewayReq serving.PutAiGatewayRequest
	var putAiGatewayJson flags.JsonFlag

	cmd.Flags().Var(&putAiGatewayJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: fallback_config
	// TODO: complex arg: guardrails
	// TODO: complex arg: inference_table_config
	// TODO: array: rate_limits
	// TODO: complex arg: usage_tracking_config

	cmd.Use = "put-ai-gateway NAME"
	cmd.Short = `Update AI Gateway of a serving endpoint.`
	cmd.Long = `Update AI Gateway of a serving endpoint.
  
  Used to update the AI Gateway of a serving endpoint. NOTE: External model,
  provisioned throughput, and pay-per-token endpoints are fully supported; agent
  endpoints currently only support inference tables.

  Arguments:
    NAME: The name of the serving endpoint whose AI Gateway is being updated. This
      field is required.`

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
			diags := putAiGatewayJson.Unmarshal(&putAiGatewayReq)
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
		putAiGatewayReq.Name = args[0]

		response, err := w.ServingEndpoints.PutAiGateway(ctx, putAiGatewayReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range putAiGatewayOverrides {
		fn(cmd, &putAiGatewayReq)
	}

	return cmd
}

// start query command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var queryOverrides []func(
	*cobra.Command,
	*serving.QueryEndpointInput,
)

func newQuery() *cobra.Command {
	cmd := &cobra.Command{}

	var queryReq serving.QueryEndpointInput
	var queryJson flags.JsonFlag

	cmd.Flags().Var(&queryJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: dataframe_records
	// TODO: complex arg: dataframe_split
	// TODO: map via StringToStringVar: extra_params
	// TODO: any: input
	// TODO: any: inputs
	// TODO: array: instances
	cmd.Flags().IntVar(&queryReq.MaxTokens, "max-tokens", queryReq.MaxTokens, `The max tokens field used ONLY for __completions__ and __chat external & foundation model__ serving endpoints.`)
	// TODO: array: messages
	cmd.Flags().IntVar(&queryReq.N, "n", queryReq.N, `The n (number of candidates) field used ONLY for __completions__ and __chat external & foundation model__ serving endpoints.`)
	// TODO: any: prompt
	// TODO: array: stop
	cmd.Flags().BoolVar(&queryReq.Stream, "stream", queryReq.Stream, `The stream field used ONLY for __completions__ and __chat external & foundation model__ serving endpoints.`)
	cmd.Flags().Float64Var(&queryReq.Temperature, "temperature", queryReq.Temperature, `The temperature field used ONLY for __completions__ and __chat external & foundation model__ serving endpoints.`)

	cmd.Use = "query NAME"
	cmd.Short = `Query a serving endpoint.`
	cmd.Long = `Query a serving endpoint.

  Arguments:
    NAME: The name of the serving endpoint. This field is required.`

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
			diags := queryJson.Unmarshal(&queryReq)
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
		queryReq.Name = args[0]

		response, err := w.ServingEndpoints.Query(ctx, queryReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range queryOverrides {
		fn(cmd, &queryReq)
	}

	return cmd
}

// start set-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var setPermissionsOverrides []func(
	*cobra.Command,
	*serving.ServingEndpointPermissionsRequest,
)

func newSetPermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var setPermissionsReq serving.ServingEndpointPermissionsRequest
	var setPermissionsJson flags.JsonFlag

	cmd.Flags().Var(&setPermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "set-permissions SERVING_ENDPOINT_ID"
	cmd.Short = `Set serving endpoint permissions.`
	cmd.Long = `Set serving endpoint permissions.
  
  Sets permissions on an object, replacing existing permissions if they exist.
  Deletes all direct permissions if none are specified. Objects can inherit
  permissions from their root object.

  Arguments:
    SERVING_ENDPOINT_ID: The serving endpoint for which to get or manage permissions.`

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
		setPermissionsReq.ServingEndpointId = args[0]

		response, err := w.ServingEndpoints.SetPermissions(ctx, setPermissionsReq)
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

// start update-config command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateConfigOverrides []func(
	*cobra.Command,
	*serving.EndpointCoreConfigInput,
)

func newUpdateConfig() *cobra.Command {
	cmd := &cobra.Command{}

	var updateConfigReq serving.EndpointCoreConfigInput
	var updateConfigJson flags.JsonFlag

	var updateConfigSkipWait bool
	var updateConfigTimeout time.Duration

	cmd.Flags().BoolVar(&updateConfigSkipWait, "no-wait", updateConfigSkipWait, `do not wait to reach NOT_UPDATING state`)
	cmd.Flags().DurationVar(&updateConfigTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach NOT_UPDATING state`)

	cmd.Flags().Var(&updateConfigJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: auto_capture_config
	// TODO: array: served_entities
	// TODO: array: served_models
	// TODO: complex arg: traffic_config

	cmd.Use = "update-config NAME"
	cmd.Short = `Update config of a serving endpoint.`
	cmd.Long = `Update config of a serving endpoint.
  
  Updates any combination of the serving endpoint's served entities, the compute
  configuration of those served entities, and the endpoint's traffic config. An
  endpoint that already has an update in progress can not be updated until the
  current update completes or fails.

  Arguments:
    NAME: The name of the serving endpoint to update. This field is required.`

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
			diags := updateConfigJson.Unmarshal(&updateConfigReq)
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
		updateConfigReq.Name = args[0]

		wait, err := w.ServingEndpoints.UpdateConfig(ctx, updateConfigReq)
		if err != nil {
			return err
		}
		if updateConfigSkipWait {
			return cmdio.Render(ctx, wait.Response)
		}
		spinner := cmdio.Spinner(ctx)
		info, err := wait.OnProgress(func(i *serving.ServingEndpointDetailed) {
			status := i.State.ConfigUpdate
			statusMessage := fmt.Sprintf("current status: %s", status)
			spinner <- statusMessage
		}).GetWithTimeout(updateConfigTimeout)
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
	for _, fn := range updateConfigOverrides {
		fn(cmd, &updateConfigReq)
	}

	return cmd
}

// start update-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updatePermissionsOverrides []func(
	*cobra.Command,
	*serving.ServingEndpointPermissionsRequest,
)

func newUpdatePermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var updatePermissionsReq serving.ServingEndpointPermissionsRequest
	var updatePermissionsJson flags.JsonFlag

	cmd.Flags().Var(&updatePermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "update-permissions SERVING_ENDPOINT_ID"
	cmd.Short = `Update serving endpoint permissions.`
	cmd.Long = `Update serving endpoint permissions.
  
  Updates the permissions on a serving endpoint. Serving endpoints can inherit
  permissions from their root object.

  Arguments:
    SERVING_ENDPOINT_ID: The serving endpoint for which to get or manage permissions.`

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
		updatePermissionsReq.ServingEndpointId = args[0]

		response, err := w.ServingEndpoints.UpdatePermissions(ctx, updatePermissionsReq)
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

// start update-provisioned-throughput-endpoint-config command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateProvisionedThroughputEndpointConfigOverrides []func(
	*cobra.Command,
	*serving.UpdateProvisionedThroughputEndpointConfigRequest,
)

func newUpdateProvisionedThroughputEndpointConfig() *cobra.Command {
	cmd := &cobra.Command{}

	var updateProvisionedThroughputEndpointConfigReq serving.UpdateProvisionedThroughputEndpointConfigRequest
	var updateProvisionedThroughputEndpointConfigJson flags.JsonFlag

	var updateProvisionedThroughputEndpointConfigSkipWait bool
	var updateProvisionedThroughputEndpointConfigTimeout time.Duration

	cmd.Flags().BoolVar(&updateProvisionedThroughputEndpointConfigSkipWait, "no-wait", updateProvisionedThroughputEndpointConfigSkipWait, `do not wait to reach NOT_UPDATING state`)
	cmd.Flags().DurationVar(&updateProvisionedThroughputEndpointConfigTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach NOT_UPDATING state`)

	cmd.Flags().Var(&updateProvisionedThroughputEndpointConfigJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "update-provisioned-throughput-endpoint-config NAME"
	cmd.Short = `Update config of a PT serving endpoint.`
	cmd.Long = `Update config of a PT serving endpoint.
  
  Updates any combination of the pt endpoint's served entities, the compute
  configuration of those served entities, and the endpoint's traffic config.
  Updates are instantaneous and endpoint should be updated instantly

  Arguments:
    NAME: The name of the pt endpoint to update. This field is required.`

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
			diags := updateProvisionedThroughputEndpointConfigJson.Unmarshal(&updateProvisionedThroughputEndpointConfigReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}
		updateProvisionedThroughputEndpointConfigReq.Name = args[0]

		wait, err := w.ServingEndpoints.UpdateProvisionedThroughputEndpointConfig(ctx, updateProvisionedThroughputEndpointConfigReq)
		if err != nil {
			return err
		}
		if updateProvisionedThroughputEndpointConfigSkipWait {
			return cmdio.Render(ctx, wait.Response)
		}
		spinner := cmdio.Spinner(ctx)
		info, err := wait.OnProgress(func(i *serving.ServingEndpointDetailed) {
			status := i.State.ConfigUpdate
			statusMessage := fmt.Sprintf("current status: %s", status)
			spinner <- statusMessage
		}).GetWithTimeout(updateProvisionedThroughputEndpointConfigTimeout)
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
	for _, fn := range updateProvisionedThroughputEndpointConfigOverrides {
		fn(cmd, &updateProvisionedThroughputEndpointConfigReq)
	}

	return cmd
}

// end service ServingEndpoints
