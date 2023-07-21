// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package serving_endpoints

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
		Use:   "serving-endpoints",
		Short: `The Serving Endpoints API allows you to create, update, and delete model serving endpoints.`,
		Long: `The Serving Endpoints API allows you to create, update, and delete model
  serving endpoints.
  
  You can use a serving endpoint to serve models from the Databricks Model
  Registry or from Unity Catalog. Endpoints expose the underlying models as
  scalable REST API endpoints using serverless compute. This means the endpoints
  and associated compute resources are fully managed by Databricks and will not
  appear in your cloud account. A serving endpoint can consist of one or more
  MLflow models from the Databricks Model Registry, called served models. A
  serving endpoint can have at most ten served models. You can configure traffic
  settings to define how requests should be routed to your served models behind
  an endpoint. Additionally, you can configure the scale of resources that
  should be applied to each served model.`,
		GroupID: "serving",
		Annotations: map[string]string{
			"package": "serving",
		},
	}

	cmd.AddCommand(newBuildLogs())
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newExportMetrics())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newList())
	cmd.AddCommand(newLogs())
	cmd.AddCommand(newQuery())
	cmd.AddCommand(newUpdateConfig())

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

	// TODO: short flags

	cmd.Use = "build-logs NAME SERVED_MODEL_NAME"
	cmd.Short = `Retrieve the logs associated with building the model's environment for a given serving endpoint's served model.`
	cmd.Long = `Retrieve the logs associated with building the model's environment for a given
  serving endpoint's served model.
  
  Retrieves the build logs associated with the provided served model.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

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
	// TODO: short flags
	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create"
	cmd.Short = `Create a new serving endpoint.`
	cmd.Long = `Create a new serving endpoint.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
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

	// TODO: short flags

	cmd.Use = "delete NAME"
	cmd.Short = `Delete a serving endpoint.`
	cmd.Long = `Delete a serving endpoint.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

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

	// TODO: short flags

	cmd.Use = "export-metrics NAME"
	cmd.Short = `Retrieve the metrics associated with a serving endpoint.`
	cmd.Long = `Retrieve the metrics associated with a serving endpoint.
  
  Retrieves the metrics associated with the provided serving endpoint in either
  Prometheus or OpenMetrics exposition format.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		exportMetricsReq.Name = args[0]

		err = w.ServingEndpoints.ExportMetrics(ctx, exportMetricsReq)
		if err != nil {
			return err
		}
		return nil
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

	// TODO: short flags

	cmd.Use = "get NAME"
	cmd.Short = `Get a single serving endpoint.`
	cmd.Long = `Get a single serving endpoint.
  
  Retrieves the details for a single serving endpoint.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

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

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	cmd.Use = "list"
	cmd.Short = `Retrieve all serving endpoints.`
	cmd.Long = `Retrieve all serving endpoints.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		response, err := w.ServingEndpoints.ListAll(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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

	// TODO: short flags

	cmd.Use = "logs NAME SERVED_MODEL_NAME"
	cmd.Short = `Retrieve the most recent log lines associated with a given serving endpoint's served model.`
	cmd.Long = `Retrieve the most recent log lines associated with a given serving endpoint's
  served model.
  
  Retrieves the service logs associated with the provided served model.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

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

// start query command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var queryOverrides []func(
	*cobra.Command,
	*serving.QueryRequest,
)

func newQuery() *cobra.Command {
	cmd := &cobra.Command{}

	var queryReq serving.QueryRequest

	// TODO: short flags

	cmd.Use = "query NAME"
	cmd.Short = `Query a serving endpoint with provided model input.`
	cmd.Long = `Query a serving endpoint with provided model input.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

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
	// TODO: short flags
	cmd.Flags().Var(&updateConfigJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: traffic_config

	cmd.Use = "update-config"
	cmd.Short = `Update a serving endpoint with a new config.`
	cmd.Long = `Update a serving endpoint with a new config.
  
  Updates any combination of the serving endpoint's served models, the compute
  configuration of those served models, and the endpoint's traffic config. An
  endpoint that already has an update in progress can not be updated until the
  current update completes or fails.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = updateConfigJson.Unmarshal(&updateConfigReq)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

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

// end service ServingEndpoints
