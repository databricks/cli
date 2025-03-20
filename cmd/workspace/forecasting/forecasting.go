// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package forecasting

import (
	"fmt"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/spf13/cobra"
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
		Annotations: map[string]string{
			"package": "ml",
		},

		// This service is being previewed; hide from help output.
		Hidden: true,
	}

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
	*ml.CreateForecastingExperimentRequest,
)

func newCreateExperiment() *cobra.Command {
	cmd := &cobra.Command{}

	var createExperimentReq ml.CreateForecastingExperimentRequest
	var createExperimentJson flags.JsonFlag

	var createExperimentSkipWait bool
	var createExperimentTimeout time.Duration

	cmd.Flags().BoolVar(&createExperimentSkipWait, "no-wait", createExperimentSkipWait, `do not wait to reach SUCCEEDED state`)
	cmd.Flags().DurationVar(&createExperimentTimeout, "timeout", 120*time.Minute, `maximum amount of time to reach SUCCEEDED state`)
	// TODO: short flags
	cmd.Flags().Var(&createExperimentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createExperimentReq.CustomWeightsColumn, "custom-weights-column", createExperimentReq.CustomWeightsColumn, `Name of the column in the input training table used to customize the weight for each time series to calculate weighted metrics.`)
	cmd.Flags().Int64Var(&createExperimentReq.DataGranularityQuantity, "data-granularity-quantity", createExperimentReq.DataGranularityQuantity, `The quantity of the input data granularity.`)
	cmd.Flags().StringVar(&createExperimentReq.ExperimentPath, "experiment-path", createExperimentReq.ExperimentPath, `The path to the created experiment.`)
	// TODO: array: holiday_regions
	cmd.Flags().Int64Var(&createExperimentReq.MaxRuntime, "max-runtime", createExperimentReq.MaxRuntime, `The maximum duration in minutes for which the experiment is allowed to run.`)
	cmd.Flags().StringVar(&createExperimentReq.PredictionDataPath, "prediction-data-path", createExperimentReq.PredictionDataPath, `The three-level (fully qualified) path to a unity catalog table.`)
	cmd.Flags().StringVar(&createExperimentReq.PrimaryMetric, "primary-metric", createExperimentReq.PrimaryMetric, `The evaluation metric used to optimize the forecasting model.`)
	cmd.Flags().StringVar(&createExperimentReq.RegisterTo, "register-to", createExperimentReq.RegisterTo, `The three-level (fully qualified) path to a unity catalog model.`)
	cmd.Flags().StringVar(&createExperimentReq.SplitColumn, "split-column", createExperimentReq.SplitColumn, `Name of the column in the input training table used for custom data splits.`)
	// TODO: array: timeseries_identifier_columns
	// TODO: array: training_frameworks

	cmd.Use = "create-experiment TRAIN_DATA_PATH TARGET_COLUMN TIME_COLUMN DATA_GRANULARITY_UNIT FORECAST_HORIZON"
	cmd.Short = `Create a forecasting experiment.`
	cmd.Long = `Create a forecasting experiment.
  
  Creates a serverless forecasting experiment. Returns the experiment ID.

  Arguments:
    TRAIN_DATA_PATH: The three-level (fully qualified) name of a unity catalog table. This
      table serves as the training data for the forecasting model.
    TARGET_COLUMN: Name of the column in the input training table that serves as the
      prediction target. The values in this column will be used as the ground
      truth for model training.
    TIME_COLUMN: Name of the column in the input training table that represents the
      timestamp of each row.
    DATA_GRANULARITY_UNIT: The time unit of the input data granularity. Together with
      data_granularity_quantity field, this defines the time interval between
      consecutive rows in the time series data. Possible values: * 'W' (weeks) *
      'D' / 'days' / 'day' * 'hours' / 'hour' / 'hr' / 'h' * 'm' / 'minute' /
      'min' / 'minutes' / 'T' * 'S' / 'seconds' / 'sec' / 'second' * 'M' /
      'month' / 'months' * 'Q' / 'quarter' / 'quarters' * 'Y' / 'year' / 'years'
    FORECAST_HORIZON: The number of time steps into the future for which predictions should be
      made. This value represents a multiple of data_granularity_unit and
      data_granularity_quantity determining how far ahead the model will
      forecast.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'train_data_path', 'target_column', 'time_column', 'data_granularity_unit', 'forecast_horizon' in your JSON input")
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

		if cmd.Flags().Changed("json") {
			diags := createExperimentJson.Unmarshal(&createExperimentReq)
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
			createExperimentReq.TrainDataPath = args[0]
		}
		if !cmd.Flags().Changed("json") {
			createExperimentReq.TargetColumn = args[1]
		}
		if !cmd.Flags().Changed("json") {
			createExperimentReq.TimeColumn = args[2]
		}
		if !cmd.Flags().Changed("json") {
			createExperimentReq.DataGranularityUnit = args[3]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[4], &createExperimentReq.ForecastHorizon)
			if err != nil {
				return fmt.Errorf("invalid FORECAST_HORIZON: %s", args[4])
			}
		}

		wait, err := w.Forecasting.CreateExperiment(ctx, createExperimentReq)
		if err != nil {
			return err
		}
		if createExperimentSkipWait {
			return cmdio.Render(ctx, wait.Response)
		}
		spinner := cmdio.Spinner(ctx)
		info, err := wait.OnProgress(func(i *ml.ForecastingExperiment) {
			status := i.State
			statusMessage := fmt.Sprintf("current status: %s", status)
			spinner <- statusMessage
		}).GetWithTimeout(createExperimentTimeout)
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
	*ml.GetForecastingExperimentRequest,
)

func newGetExperiment() *cobra.Command {
	cmd := &cobra.Command{}

	var getExperimentReq ml.GetForecastingExperimentRequest

	// TODO: short flags

	cmd.Use = "get-experiment EXPERIMENT_ID"
	cmd.Short = `Get a forecasting experiment.`
	cmd.Long = `Get a forecasting experiment.
  
  Public RPC to get forecasting experiment

  Arguments:
    EXPERIMENT_ID: The unique ID of a forecasting experiment`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getExperimentReq.ExperimentId = args[0]

		response, err := w.Forecasting.GetExperiment(ctx, getExperimentReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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

// end service forecasting
