// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package forecasting

import (
	"errors"
	"fmt"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/sdk-go/options/lro"
	"github.com/spf13/cobra"

	forecasting "github.com/databricks/cli/libs/sdktypes/forecasting/v1"
	sdkforecasting "github.com/databricks/sdk-go/forecasting/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "forecasting",
		Short: `The Forecasting API allows you to create and get serverless forecasting experiments.`,
		Long: `The Forecasting API allows you to create and get serverless forecasting
  experiments`,
		GroupID: "ml",

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	// Add methods
	cmd.AddCommand(newCreateExperiment())
	cmd.AddCommand(newGetExperiment())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-experiment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createExperimentOverrides []func(
	*cobra.Command,
	*forecasting.CreateForecastingExperimentRequest,
)

func newCreateExperiment() *cobra.Command {
	cmd := &cobra.Command{}
	var createExperimentReq forecasting.CreateForecastingExperimentRequest

	var createExperimentJson flags.JsonFlag

	var createExperimentSkipWait bool
	var createExperimentTimeout time.Duration

	cmd.Flags().BoolVar(&createExperimentSkipWait, "no-wait", createExperimentSkipWait, `do not wait to reach SUCCEEDED state`)
	cmd.Flags().DurationVar(&createExperimentTimeout, "timeout", 120*time.Minute, `maximum amount of time to reach SUCCEEDED state`)

	cmd.Flags().Var(&createExperimentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createExperimentReq.CustomWeightsColumn, "custom-weights-column", `The column in the training table used to customize weights for each time series.`)
	flags.StringPtrVar(cmd.Flags(), &createExperimentReq.ExperimentPath, "experiment-path", `The path in the workspace to store the created experiment.`)
	flags.StringPtrVar(cmd.Flags(), &createExperimentReq.FutureFeatureDataPath, "future-feature-data-path", `The fully qualified path of a Unity Catalog table, formatted as catalog_name.schema_name.table_name, used to store future feature data for predictions.`)
	// TODO: array: holiday_regions
	// TODO: array: include_features
	flags.Int64PtrVar(cmd.Flags(), &createExperimentReq.MaxRuntime, "max-runtime", `The maximum duration for the experiment in minutes.`)
	flags.StringPtrVar(cmd.Flags(), &createExperimentReq.PredictionDataPath, "prediction-data-path", `The fully qualified path of a Unity Catalog table, formatted as catalog_name.schema_name.table_name, used to store predictions.`)
	flags.StringPtrVar(cmd.Flags(), &createExperimentReq.PrimaryMetric, "primary-metric", `The evaluation metric used to optimize the forecasting model.`)
	flags.StringPtrVar(cmd.Flags(), &createExperimentReq.RegisterTo, "register-to", `The fully qualified path of a Unity Catalog model, formatted as catalog_name.schema_name.model_name, used to store the best model.`)
	flags.StringPtrVar(cmd.Flags(), &createExperimentReq.SplitColumn, "split-column", `// The column in the training table used for custom data splits.`)
	// TODO: array: timeseries_identifier_columns
	// TODO: array: training_frameworks

	cmd.Use = "create-experiment TRAIN_DATA_PATH TARGET_COLUMN TIME_COLUMN FORECAST_GRANULARITY FORECAST_HORIZON"
	cmd.Short = `Create a forecasting experiment.`
	cmd.Long = `Create a forecasting experiment.

  Creates a serverless forecasting experiment. Returns the experiment ID.

  Arguments:
    TRAIN_DATA_PATH: The fully qualified path of a Unity Catalog table, formatted as
      catalog_name.schema_name.table_name, used as training data for the
      forecasting model.
    TARGET_COLUMN: The column in the input training table used as the prediction target for
      model training. The values in this column are used as the ground truth for
      model training.
    TIME_COLUMN: The column in the input training table that represents each row's
      timestamp.
    FORECAST_GRANULARITY: The time interval between consecutive rows in the time series data.
      Possible values include: '1 second', '1 minute', '5 minutes', '10
      minutes', '15 minutes', '30 minutes', 'Hourly', 'Daily', 'Weekly',
      'Monthly', 'Quarterly', 'Yearly'.
    FORECAST_HORIZON: The number of time steps into the future to make predictions, calculated
      as a multiple of forecast_granularity. This value represents how far ahead
      the model should forecast.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'train_data_path', 'target_column', 'time_column', 'forecast_granularity', 'forecast_horizon' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(5)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createExperimentJson.Unmarshal(&createExperimentReq)
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
			createExperimentReq.TrainDataPath = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			createExperimentReq.TargetColumn = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			createExperimentReq.TimeColumn = new(args[2])
		}
		if !cmd.Flags().Changed("json") {
			createExperimentReq.ForecastGranularity = new(args[3])
		}
		if !cmd.Flags().Changed("json") {
			var createExperimentForecastHorizonArg int64
			_, err = fmt.Sscan(args[4], &createExperimentForecastHorizonArg)
			if err != nil {
				return fmt.Errorf("invalid FORECAST_HORIZON: %s", args[4])
			}
			createExperimentReq.ForecastHorizon = &createExperimentForecastHorizonArg
		}

		sdkReq, err := forecasting.CreateForecastingExperimentRequestToSDK(&createExperimentReq)
		if err != nil {
			return err
		}
		if createExperimentSkipWait {
			api, err := apiclient.New(w.Config)
			if err != nil {
				return err
			}
			var current forecasting.CreateForecastingExperimentResponse
			err = api.Do(ctx, "POST", "/api/2.0/automl/create-forecasting-experiment", auth.WorkspaceIDHeaders(w.Config), nil, &createExperimentReq, &current)
			if err != nil {
				return err
			}
			return cmdio.Render(ctx, current)
		}
		wait, err := w.Forecasting.CreateForecastingExperiment(ctx, *sdkReq)
		if err != nil {
			return err
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for create-experiment to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(createExperimentTimeout))
		sp.Close()
		if err != nil {
			return &cmdgen.WaitError{Err: err}
		}

		out, err := forecasting.ForecastingExperimentFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createExperimentOverrides {
		fn(cmd, &createExperimentReq)
	}

	return cmd
}

// start get-experiment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getExperimentOverrides []func(
	*cobra.Command,
	*sdkforecasting.GetForecastingExperimentRequest,
)

func newGetExperiment() *cobra.Command {
	cmd := &cobra.Command{}
	var getExperimentReq sdkforecasting.GetForecastingExperimentRequest

	cmd.Use = "get-experiment EXPERIMENT_ID"
	cmd.Short = `Get a forecasting experiment.`
	cmd.Long = `Get a forecasting experiment.

  Public RPC to get forecasting experiment

  Arguments:
    EXPERIMENT_ID: The unique ID of a forecasting experiment`

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

		getExperimentReq.ExperimentId = new(args[0])

		sdkReq := &getExperimentReq
		response, err := w.Forecasting.GetForecastingExperiment(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := forecasting.ForecastingExperimentFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getExperimentOverrides {
		fn(cmd, &getExperimentReq)
	}

	return cmd
}

// end service Forecasting
