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

var Cmd = &cobra.Command{
	Use:   "serving-endpoints",
	Short: `The Serving Endpoints API allows you to create, update, and delete model serving endpoints.`,
	Long: `The Serving Endpoints API allows you to create, update, and delete model
  serving endpoints.
  
  You can use a serving endpoint to serve models from the Databricks Model
  Registry. Endpoints expose the underlying models as scalable REST API
  endpoints using serverless compute. This means the endpoints and associated
  compute resources are fully managed by Databricks and will not appear in your
  cloud account. A serving endpoint can consist of one or more MLflow models
  from the Databricks Model Registry, called served models. A serving endpoint
  can have at most ten served models. You can configure traffic settings to
  define how requests should be routed to your served models behind an endpoint.
  Additionally, you can configure the scale of resources that should be applied
  to each served model.`,
}

// start build-logs command

var buildLogsReq serving.BuildLogsRequest
var buildLogsJson flags.JsonFlag

func init() {
	Cmd.AddCommand(buildLogsCmd)
	// TODO: short flags
	buildLogsCmd.Flags().Var(&buildLogsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var buildLogsCmd = &cobra.Command{
	Use:   "build-logs NAME SERVED_MODEL_NAME",
	Short: `Retrieve the logs associated with building the model's environment for a given serving endpoint's served model.`,
	Long: `Retrieve the logs associated with building the model's environment for a given
  serving endpoint's served model.
  
  Retrieves the build logs associated with the provided served model.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = buildLogsJson.Unmarshal(&buildLogsReq)
			if err != nil {
				return err
			}
		} else {
			buildLogsReq.Name = args[0]
			buildLogsReq.ServedModelName = args[1]
		}

		response, err := w.ServingEndpoints.BuildLogs(ctx, buildLogsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start create command

var createReq serving.CreateServingEndpoint
var createJson flags.JsonFlag
var createSkipWait bool
var createTimeout time.Duration

func init() {
	Cmd.AddCommand(createCmd)

	createCmd.Flags().BoolVar(&createSkipWait, "no-wait", createSkipWait, `do not wait to reach NOT_UPDATING state`)
	createCmd.Flags().DurationVar(&createTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach NOT_UPDATING state`)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a new serving endpoint.`,
	Long:  `Create a new serving endpoint.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
			createReq.Name = args[0]
			_, err = fmt.Sscan(args[1], &createReq.Config)
			if err != nil {
				return fmt.Errorf("invalid CONFIG: %s", args[1])
			}
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
	},
}

// start delete command

var deleteReq serving.DeleteServingEndpointRequest
var deleteJson flags.JsonFlag

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags
	deleteCmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete NAME",
	Short: `Delete a serving endpoint.`,
	Long:  `Delete a serving endpoint.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = deleteJson.Unmarshal(&deleteReq)
			if err != nil {
				return err
			}
		} else {
			deleteReq.Name = args[0]
		}

		err = w.ServingEndpoints.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start export-metrics command

var exportMetricsReq serving.ExportMetricsRequest
var exportMetricsJson flags.JsonFlag

func init() {
	Cmd.AddCommand(exportMetricsCmd)
	// TODO: short flags
	exportMetricsCmd.Flags().Var(&exportMetricsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var exportMetricsCmd = &cobra.Command{
	Use:   "export-metrics NAME",
	Short: `Retrieve the metrics corresponding to a serving endpoint for the current time in Prometheus or OpenMetrics exposition format.`,
	Long: `Retrieve the metrics corresponding to a serving endpoint for the current time
  in Prometheus or OpenMetrics exposition format.
  
  Retrieves the metrics associated with the provided serving endpoint in either
  Prometheus or OpenMetrics exposition format.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = exportMetricsJson.Unmarshal(&exportMetricsReq)
			if err != nil {
				return err
			}
		} else {
			exportMetricsReq.Name = args[0]
		}

		err = w.ServingEndpoints.ExportMetrics(ctx, exportMetricsReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq serving.GetServingEndpointRequest
var getJson flags.JsonFlag

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags
	getCmd.Flags().Var(&getJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var getCmd = &cobra.Command{
	Use:   "get NAME",
	Short: `Get a single serving endpoint.`,
	Long: `Get a single serving endpoint.
  
  Retrieves the details for a single serving endpoint.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = getJson.Unmarshal(&getReq)
			if err != nil {
				return err
			}
		} else {
			getReq.Name = args[0]
		}

		response, err := w.ServingEndpoints.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start list command

func init() {
	Cmd.AddCommand(listCmd)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `Retrieve all serving endpoints.`,
	Long:  `Retrieve all serving endpoints.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		response, err := w.ServingEndpoints.ListAll(ctx)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start logs command

var logsReq serving.LogsRequest
var logsJson flags.JsonFlag

func init() {
	Cmd.AddCommand(logsCmd)
	// TODO: short flags
	logsCmd.Flags().Var(&logsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var logsCmd = &cobra.Command{
	Use:   "logs NAME SERVED_MODEL_NAME",
	Short: `Retrieve the most recent log lines associated with a given serving endpoint's served model.`,
	Long: `Retrieve the most recent log lines associated with a given serving endpoint's
  served model.
  
  Retrieves the service logs associated with the provided served model.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = logsJson.Unmarshal(&logsReq)
			if err != nil {
				return err
			}
		} else {
			logsReq.Name = args[0]
			logsReq.ServedModelName = args[1]
		}

		response, err := w.ServingEndpoints.Logs(ctx, logsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start query command

var queryReq serving.QueryRequest
var queryJson flags.JsonFlag

func init() {
	Cmd.AddCommand(queryCmd)
	// TODO: short flags
	queryCmd.Flags().Var(&queryJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var queryCmd = &cobra.Command{
	Use:   "query NAME",
	Short: `Query a serving endpoint with provided model input.`,
	Long:  `Query a serving endpoint with provided model input.`,

	Annotations: map[string]string{},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = queryJson.Unmarshal(&queryReq)
			if err != nil {
				return err
			}
		} else {
			queryReq.Name = args[0]
		}

		response, err := w.ServingEndpoints.Query(ctx, queryReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start update-config command

var updateConfigReq serving.EndpointCoreConfigInput
var updateConfigJson flags.JsonFlag
var updateConfigSkipWait bool
var updateConfigTimeout time.Duration

func init() {
	Cmd.AddCommand(updateConfigCmd)

	updateConfigCmd.Flags().BoolVar(&updateConfigSkipWait, "no-wait", updateConfigSkipWait, `do not wait to reach NOT_UPDATING state`)
	updateConfigCmd.Flags().DurationVar(&updateConfigTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach NOT_UPDATING state`)
	// TODO: short flags
	updateConfigCmd.Flags().Var(&updateConfigJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: traffic_config

}

var updateConfigCmd = &cobra.Command{
	Use:   "update-config",
	Short: `Update a serving endpoint with a new config.`,
	Long: `Update a serving endpoint with a new config.
  
  Updates any combination of the serving endpoint's served models, the compute
  configuration of those served models, and the endpoint's traffic config. An
  endpoint that already has an update in progress can not be updated until the
  current update completes or fails.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = updateConfigJson.Unmarshal(&updateConfigReq)
			if err != nil {
				return err
			}
		} else {
			_, err = fmt.Sscan(args[0], &updateConfigReq.ServedModels)
			if err != nil {
				return fmt.Errorf("invalid SERVED_MODELS: %s", args[0])
			}
			updateConfigReq.Name = args[1]
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
	},
}

// end service ServingEndpoints
