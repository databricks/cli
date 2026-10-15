// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package experiments

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	accessmanagement "github.com/databricks/cli/libs/sdktypes/accessmanagement/v1"
	experiments "github.com/databricks/cli/libs/sdktypes/experiments/v1"
	sdkaccessmanagement "github.com/databricks/sdk-go/accessmanagement/v1"
	sdkexperiments "github.com/databricks/sdk-go/experiments/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "experiments",
		Short: `Experiments are the primary unit of organization in MLflow; all MLflow runs belong to an experiment.`,
		Long: `Experiments are the primary unit of organization in MLflow; all MLflow runs
  belong to an experiment. Each experiment lets you visualize, search, and
  compare runs, as well as download run artifacts or metadata for analysis in
  other tools. Experiments are maintained in a Databricks hosted MLflow tracking
  server.

  Experiments are located in the workspace file tree. You manage experiments
  using the same tools you use to manage other workspace objects such as
  folders, notebooks, and libraries.`,
		GroupID: "ml",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	// Add methods
	cmd.AddCommand(newCreateExperiment())
	cmd.AddCommand(newCreateLoggedModel())
	cmd.AddCommand(newCreateRun())
	cmd.AddCommand(newDeleteExperiment())
	cmd.AddCommand(newDeleteLoggedModel())
	cmd.AddCommand(newDeleteLoggedModelTag())
	cmd.AddCommand(newDeleteRun())
	cmd.AddCommand(newDeleteRuns())
	cmd.AddCommand(newDeleteTag())
	cmd.AddCommand(newFinalizeLoggedModel())
	cmd.AddCommand(newGetByName())
	cmd.AddCommand(newGetExperiment())
	cmd.AddCommand(newGetHistory())
	cmd.AddCommand(newGetLoggedModel())
	cmd.AddCommand(newGetPermissionLevels())
	cmd.AddCommand(newGetPermissions())
	cmd.AddCommand(newGetRun())
	cmd.AddCommand(newListArtifacts())
	cmd.AddCommand(newListExperiments())
	cmd.AddCommand(newLogBatch())
	cmd.AddCommand(newLogInputs())
	cmd.AddCommand(newLogLoggedModelParams())
	cmd.AddCommand(newLogMetric())
	cmd.AddCommand(newLogModel())
	cmd.AddCommand(newLogOutputs())
	cmd.AddCommand(newLogParam())
	cmd.AddCommand(newRestoreExperiment())
	cmd.AddCommand(newRestoreRun())
	cmd.AddCommand(newRestoreRuns())
	cmd.AddCommand(newSearchExperiments())
	cmd.AddCommand(newSearchLoggedModels())
	cmd.AddCommand(newSearchRuns())
	cmd.AddCommand(newSetExperimentTag())
	cmd.AddCommand(newSetLoggedModelTags())
	cmd.AddCommand(newSetPermissions())
	cmd.AddCommand(newSetTag())
	cmd.AddCommand(newUpdateExperiment())
	cmd.AddCommand(newUpdatePermissions())
	cmd.AddCommand(newUpdateRun())

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
	*experiments.CreateExperimentRequest,
)

func newCreateExperiment() *cobra.Command {
	cmd := &cobra.Command{}
	var createExperimentReq experiments.CreateExperimentRequest

	var createExperimentJson flags.JsonFlag

	cmd.Flags().Var(&createExperimentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createExperimentReq.ArtifactLocation, "artifact-location", `Location where all artifacts for the experiment are stored.`)
	// TODO: array: tags
	// TODO: complex arg: trace_location

	cmd.Use = "create-experiment NAME"
	cmd.Short = `Create experiment.`
	cmd.Long = `Create experiment.

  Creates an experiment with a name. Returns the ID of the newly created
  experiment. Validates that another experiment with the same name does not
  already exist and fails if another experiment with the same name already
  exists.

  Throws RESOURCE_ALREADY_EXISTS if an experiment with the given name exists.
  Note: In some contexts, this error may be remapped to ALREADY_EXISTS. To be
  safe, clients should check for both error codes.

  Arguments:
    NAME: Experiment name.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'name' in your JSON input")
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
			createExperimentReq.Name = new(args[0])
		}

		sdkReq, err := experiments.CreateExperimentRequestToSDK(&createExperimentReq)
		if err != nil {
			return err
		}
		response, err := w.Experiments.CreateExperiment(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := experiments.CreateExperimentResponseFromSDK(response)
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

// start create-logged-model command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createLoggedModelOverrides []func(
	*cobra.Command,
	*experiments.CreateLoggedModelRequest,
)

func newCreateLoggedModel() *cobra.Command {
	cmd := &cobra.Command{}
	var createLoggedModelReq experiments.CreateLoggedModelRequest

	var createLoggedModelJson flags.JsonFlag

	cmd.Flags().Var(&createLoggedModelJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createLoggedModelReq.ModelType, "model-type", `The type of the model, such as "Agent", "Classifier", "LLM".`)
	flags.StringPtrVar(cmd.Flags(), &createLoggedModelReq.Name, "name", `The name of the model (optional).`)
	// TODO: array: params
	flags.StringPtrVar(cmd.Flags(), &createLoggedModelReq.SourceRunId, "source-run-id", `The ID of the run that created the model.`)
	// TODO: array: tags

	cmd.Use = "create-logged-model EXPERIMENT_ID"
	cmd.Short = `Create a logged model.`
	cmd.Long = `Create a logged model.

  Arguments:
    EXPERIMENT_ID: The ID of the experiment that owns the model.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'experiment_id' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createLoggedModelJson.Unmarshal(&createLoggedModelReq)
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
			createLoggedModelReq.ExperimentId = new(args[0])
		}

		sdkReq, err := experiments.CreateLoggedModelRequestToSDK(&createLoggedModelReq)
		if err != nil {
			return err
		}
		response, err := w.Experiments.CreateLoggedModel(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := experiments.CreateLoggedModelResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createLoggedModelOverrides {
		fn(cmd, &createLoggedModelReq)
	}

	return cmd
}

// start create-run command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createRunOverrides []func(
	*cobra.Command,
	*experiments.CreateRunRequest,
)

func newCreateRun() *cobra.Command {
	cmd := &cobra.Command{}
	var createRunReq experiments.CreateRunRequest

	var createRunJson flags.JsonFlag

	cmd.Flags().Var(&createRunJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &createRunReq.ExperimentId, "experiment-id", `ID of the associated experiment.`)
	flags.StringPtrVar(cmd.Flags(), &createRunReq.RunName, "run-name", `The name of the run.`)
	flags.Int64PtrVar(cmd.Flags(), &createRunReq.StartTime, "start-time", `Unix timestamp in milliseconds of when the run started.`)
	// TODO: array: tags
	flags.StringPtrVar(cmd.Flags(), &createRunReq.UserId, "user-id", `ID of the user executing the run.`)

	cmd.Use = "create-run"
	cmd.Short = `Create a run.`
	cmd.Long = `Create a run.

  Creates a new run within an experiment. A run is usually a single execution of
  a machine learning or data ETL pipeline. MLflow uses runs to track the
  mlflowParam, mlflowMetric, and mlflowRunTag associated with a single
  execution.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createRunJson.Unmarshal(&createRunReq)
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

		sdkReq, err := experiments.CreateRunRequestToSDK(&createRunReq)
		if err != nil {
			return err
		}
		response, err := w.Experiments.CreateRun(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := experiments.CreateRunResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createRunOverrides {
		fn(cmd, &createRunReq)
	}

	return cmd
}

// start delete-experiment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteExperimentOverrides []func(
	*cobra.Command,
	*experiments.DeleteExperimentRequest,
)

func newDeleteExperiment() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteExperimentReq experiments.DeleteExperimentRequest

	var deleteExperimentJson flags.JsonFlag

	cmd.Flags().Var(&deleteExperimentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "delete-experiment EXPERIMENT_ID"
	cmd.Short = `Delete an experiment.`
	cmd.Long = `Delete an experiment.

  Marks an experiment and associated metadata, runs, metrics, params, and tags
  for deletion. If the experiment uses FileStore, artifacts associated with the
  experiment are also deleted.

  Arguments:
    EXPERIMENT_ID: ID of the associated experiment.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'experiment_id' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := deleteExperimentJson.Unmarshal(&deleteExperimentReq)
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
			deleteExperimentReq.ExperimentId = new(args[0])
		}

		sdkReq, err := experiments.DeleteExperimentRequestToSDK(&deleteExperimentReq)
		if err != nil {
			return err
		}
		response, err := w.Experiments.DeleteExperiment(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteExperimentOverrides {
		fn(cmd, &deleteExperimentReq)
	}

	return cmd
}

// start delete-logged-model command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteLoggedModelOverrides []func(
	*cobra.Command,
	*sdkexperiments.DeleteLoggedModelRequest,
)

func newDeleteLoggedModel() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteLoggedModelReq sdkexperiments.DeleteLoggedModelRequest

	cmd.Use = "delete-logged-model MODEL_ID"
	cmd.Short = `Delete a logged model.`
	cmd.Long = `Delete a logged model.

  Arguments:
    MODEL_ID: The ID of the logged model to delete.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteLoggedModelReq.ModelId = new(args[0])

		sdkReq := &deleteLoggedModelReq
		response, err := w.Experiments.DeleteLoggedModel(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteLoggedModelOverrides {
		fn(cmd, &deleteLoggedModelReq)
	}

	return cmd
}

// start delete-logged-model-tag command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteLoggedModelTagOverrides []func(
	*cobra.Command,
	*sdkexperiments.DeleteLoggedModelTagRequest,
)

func newDeleteLoggedModelTag() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteLoggedModelTagReq sdkexperiments.DeleteLoggedModelTagRequest

	cmd.Use = "delete-logged-model-tag MODEL_ID TAG_KEY"
	cmd.Short = `Delete a tag on a logged model.`
	cmd.Long = `Delete a tag on a logged model.

  Arguments:
    MODEL_ID: The ID of the logged model to delete the tag from.
    TAG_KEY: The tag key.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteLoggedModelTagReq.ModelId = new(args[0])
		deleteLoggedModelTagReq.TagKey = new(args[1])

		sdkReq := &deleteLoggedModelTagReq
		response, err := w.Experiments.DeleteLoggedModelTag(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteLoggedModelTagOverrides {
		fn(cmd, &deleteLoggedModelTagReq)
	}

	return cmd
}

// start delete-run command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteRunOverrides []func(
	*cobra.Command,
	*experiments.DeleteRunRequest,
)

func newDeleteRun() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteRunReq experiments.DeleteRunRequest

	var deleteRunJson flags.JsonFlag

	cmd.Flags().Var(&deleteRunJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "delete-run RUN_ID"
	cmd.Short = `Delete a run.`
	cmd.Long = `Delete a run.

  Marks a run for deletion.

  Arguments:
    RUN_ID: ID of the run to delete.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'run_id' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := deleteRunJson.Unmarshal(&deleteRunReq)
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
			deleteRunReq.RunId = new(args[0])
		}

		sdkReq, err := experiments.DeleteRunRequestToSDK(&deleteRunReq)
		if err != nil {
			return err
		}
		response, err := w.Experiments.DeleteRun(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteRunOverrides {
		fn(cmd, &deleteRunReq)
	}

	return cmd
}

// start delete-runs command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteRunsOverrides []func(
	*cobra.Command,
	*experiments.DeleteRunsRequest,
)

func newDeleteRuns() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteRunsReq experiments.DeleteRunsRequest

	var deleteRunsJson flags.JsonFlag

	cmd.Flags().Var(&deleteRunsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.IntPtrVar(cmd.Flags(), &deleteRunsReq.MaxRuns, "max-runs", `An optional positive integer indicating the maximum number of runs to delete.`)

	cmd.Use = "delete-runs EXPERIMENT_ID MAX_TIMESTAMP_MILLIS"
	cmd.Short = `Delete runs by creation time.`
	cmd.Long = `Delete runs by creation time.

  Bulk delete runs in an experiment that were created prior to or at the
  specified timestamp. Deletes at most max_runs per request. See the
  cloud-specific MLflow runs documentation for a Python client example.

  Arguments:
    EXPERIMENT_ID: The ID of the experiment containing the runs to delete.
    MAX_TIMESTAMP_MILLIS: The maximum creation timestamp in milliseconds since the UNIX epoch for
      deleting runs. Only runs created prior to or at this timestamp are
      deleted.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'experiment_id', 'max_timestamp_millis' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := deleteRunsJson.Unmarshal(&deleteRunsReq)
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
			deleteRunsReq.ExperimentId = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			var deleteRunsMaxTimestampMillisArg int64
			_, err = fmt.Sscan(args[1], &deleteRunsMaxTimestampMillisArg)
			if err != nil {
				return fmt.Errorf("invalid MAX_TIMESTAMP_MILLIS: %s", args[1])
			}
			deleteRunsReq.MaxTimestampMillis = &deleteRunsMaxTimestampMillisArg
		}

		sdkReq, err := experiments.DeleteRunsRequestToSDK(&deleteRunsReq)
		if err != nil {
			return err
		}
		response, err := w.Experiments.DeleteRuns(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := experiments.DeleteRunsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteRunsOverrides {
		fn(cmd, &deleteRunsReq)
	}

	return cmd
}

// start delete-tag command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteTagOverrides []func(
	*cobra.Command,
	*experiments.DeleteTagRequest,
)

func newDeleteTag() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteTagReq experiments.DeleteTagRequest

	var deleteTagJson flags.JsonFlag

	cmd.Flags().Var(&deleteTagJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "delete-tag RUN_ID KEY"
	cmd.Short = `Delete a tag on a run.`
	cmd.Long = `Delete a tag on a run.

  Deletes a tag on a run. Tags are run metadata that can be updated during a run
  and after a run completes.

  Arguments:
    RUN_ID: ID of the run that the tag was logged under. Must be provided.
    KEY: Name of the tag. Maximum size is 255 bytes. Must be provided.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'run_id', 'key' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := deleteTagJson.Unmarshal(&deleteTagReq)
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
			deleteTagReq.RunId = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			deleteTagReq.Key = new(args[1])
		}

		sdkReq, err := experiments.DeleteTagRequestToSDK(&deleteTagReq)
		if err != nil {
			return err
		}
		response, err := w.Experiments.DeleteTag(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteTagOverrides {
		fn(cmd, &deleteTagReq)
	}

	return cmd
}

// start finalize-logged-model command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var finalizeLoggedModelOverrides []func(
	*cobra.Command,
	*experiments.FinalizeLoggedModelRequest,
)

func newFinalizeLoggedModel() *cobra.Command {
	cmd := &cobra.Command{}
	var finalizeLoggedModelReq experiments.FinalizeLoggedModelRequest

	var finalizeLoggedModelJson flags.JsonFlag

	cmd.Flags().Var(&finalizeLoggedModelJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "finalize-logged-model MODEL_ID STATUS"
	cmd.Short = `Finalize a logged model.`
	cmd.Long = `Finalize a logged model.

  Arguments:
    MODEL_ID: The ID of the logged model to finalize.
    STATUS: Whether or not the model is ready for use.
      "LOGGED_MODEL_UPLOAD_FAILED" indicates that something went wrong when
      logging the model weights / agent code.
      Supported values: [LOGGED_MODEL_PENDING, LOGGED_MODEL_READY, LOGGED_MODEL_UPLOAD_FAILED]`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(1)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only MODEL_ID as positional arguments. Provide 'status' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := finalizeLoggedModelJson.Unmarshal(&finalizeLoggedModelReq)
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
		finalizeLoggedModelReq.ModelId = new(args[0])
		if !cmd.Flags().Changed("json") {
			finalizeLoggedModelReq.Status = experiments.LoggedModelStatus(args[1])
		}

		sdkReq, err := experiments.FinalizeLoggedModelRequestToSDK(&finalizeLoggedModelReq)
		if err != nil {
			return err
		}
		response, err := w.Experiments.FinalizeLoggedModel(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := experiments.FinalizeLoggedModelResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range finalizeLoggedModelOverrides {
		fn(cmd, &finalizeLoggedModelReq)
	}

	return cmd
}

// start get-by-name command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getByNameOverrides []func(
	*cobra.Command,
	*experiments.GetExperimentByNameRequest,
)

func newGetByName() *cobra.Command {
	cmd := &cobra.Command{}
	var getByNameReq experiments.GetExperimentByNameRequest

	cmd.Use = "get-by-name EXPERIMENT_NAME"
	cmd.Short = `Get an experiment by name.`
	cmd.Long = `Get an experiment by name.

  Gets metadata for an experiment.

  This endpoint will return deleted experiments, but prefers the active
  experiment if an active and deleted experiment share the same name. If
  multiple deleted experiments share the same name, the API will return one of
  them.

  Throws RESOURCE_DOES_NOT_EXIST if no experiment with the specified name
  exists.

  Arguments:
    EXPERIMENT_NAME: Name of the associated experiment.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getByNameReq.ExperimentName = new(args[0])

		sdkReq, err := experiments.GetExperimentByNameRequestToSDK(&getByNameReq)
		if err != nil {
			return err
		}
		response, err := w.Experiments.GetExperimentByName(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := experiments.GetExperimentByNameResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getByNameOverrides {
		fn(cmd, &getByNameReq)
	}

	return cmd
}

// start get-experiment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getExperimentOverrides []func(
	*cobra.Command,
	*experiments.GetExperimentRequest,
)

func newGetExperiment() *cobra.Command {
	cmd := &cobra.Command{}
	var getExperimentReq experiments.GetExperimentRequest

	cmd.Use = "get-experiment EXPERIMENT_ID"
	cmd.Short = `Get an experiment.`
	cmd.Long = `Get an experiment.

  Gets metadata for an experiment. This method works on deleted experiments.

  Arguments:
    EXPERIMENT_ID: ID of the associated experiment.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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

		sdkReq, err := experiments.GetExperimentRequestToSDK(&getExperimentReq)
		if err != nil {
			return err
		}
		response, err := w.Experiments.GetExperiment(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := experiments.GetExperimentResponseFromSDK(response)
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

// start get-history command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getHistoryOverrides []func(
	*cobra.Command,
	*experiments.ListMetricHistoryRequest,
)

func newGetHistory() *cobra.Command {
	cmd := &cobra.Command{}
	var getHistoryReq experiments.ListMetricHistoryRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var getHistoryLimit int

	flags.IntPtrVar(cmd.Flags(), &getHistoryReq.MaxResults, "max-results", `Maximum number of Metric records to return per paginated request.`)
	flags.StringPtrVar(cmd.Flags(), &getHistoryReq.RunId, "run-id", `ID of the run from which to fetch metric values.`)
	flags.StringPtrVar(cmd.Flags(), &getHistoryReq.RunUuid, "run-uuid", `[Deprecated, use run_id instead] ID of the run from which to fetch metric values.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&getHistoryLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &getHistoryReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "get-history METRIC_KEY"
	cmd.Short = `Get metric history for a run.`
	cmd.Long = `Get metric history for a run.

  Gets a list of all values for the specified metric for a given run.

  Arguments:
    METRIC_KEY: Name of the metric.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getHistoryReq.MetricKey = new(args[0])

		sdkReq, err := experiments.ListMetricHistoryRequestToSDK(&getHistoryReq)
		if err != nil {
			return err
		}
		if getHistoryLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", getHistoryLimit)
		}
		if getHistoryLimit > 0 {
			ctx = cmdio.WithLimit(ctx, getHistoryLimit)
		}
		response := w.Experiments.ListMetricHistoryIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, experiments.MetricFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getHistoryOverrides {
		fn(cmd, &getHistoryReq)
	}

	return cmd
}

// start get-logged-model command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getLoggedModelOverrides []func(
	*cobra.Command,
	*sdkexperiments.GetLoggedModelRequest,
)

func newGetLoggedModel() *cobra.Command {
	cmd := &cobra.Command{}
	var getLoggedModelReq sdkexperiments.GetLoggedModelRequest

	cmd.Use = "get-logged-model MODEL_ID"
	cmd.Short = `Get a logged model.`
	cmd.Long = `Get a logged model.

  Arguments:
    MODEL_ID: The ID of the logged model to retrieve.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getLoggedModelReq.ModelId = new(args[0])

		sdkReq := &getLoggedModelReq
		response, err := w.Experiments.GetLoggedModel(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := experiments.GetLoggedModelResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getLoggedModelOverrides {
		fn(cmd, &getLoggedModelReq)
	}

	return cmd
}

// start get-permission-levels command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPermissionLevelsOverrides []func(
	*cobra.Command,
	*sdkaccessmanagement.ListPermissionLevelsRequest,
)

func newGetPermissionLevels() *cobra.Command {
	cmd := &cobra.Command{}
	var getPermissionLevelsReq sdkaccessmanagement.ListPermissionLevelsRequest

	cmd.Use = "get-permission-levels EXPERIMENT_ID"
	cmd.Short = `Get experiment permission levels.`
	cmd.Long = `Get experiment permission levels.

  Gets the permission levels that a user can have on an object.

  Arguments:
    EXPERIMENT_ID: The experiment for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getPermissionLevelsReq.RequestObjectId = new(args[0])

		getPermissionLevelsReq.RequestObjectType = new("experiments")

		sdkReq := &getPermissionLevelsReq
		response, err := w.AccessManagement.ListPermissionLevels(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.ListPermissionLevelsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkaccessmanagement.GetObjectPermissionsRequest,
)

func newGetPermissions() *cobra.Command {
	cmd := &cobra.Command{}
	var getPermissionsReq sdkaccessmanagement.GetObjectPermissionsRequest

	cmd.Use = "get-permissions EXPERIMENT_ID"
	cmd.Short = `Get experiment permissions.`
	cmd.Long = `Get experiment permissions.

  Gets the permissions of an experiment. Experiments can inherit permissions
  from their root object.

  Arguments:
    EXPERIMENT_ID: The experiment for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getPermissionsReq.RequestObjectId = new(args[0])

		getPermissionsReq.RequestObjectType = new("experiments")

		sdkReq := &getPermissionsReq
		response, err := w.AccessManagement.GetObjectPermissions(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.PermissionsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start get-run command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getRunOverrides []func(
	*cobra.Command,
	*experiments.GetRunRequest,
)

func newGetRun() *cobra.Command {
	cmd := &cobra.Command{}
	var getRunReq experiments.GetRunRequest

	flags.StringPtrVar(cmd.Flags(), &getRunReq.RunUuid, "run-uuid", `[Deprecated, use run_id instead] ID of the run to fetch.`)

	cmd.Use = "get-run RUN_ID"
	cmd.Short = `Get a run.`
	cmd.Long = `Get a run.

  Gets the metadata, metrics, params, and tags for a run. In the case where
  multiple metrics with the same key are logged for a run, return only the value
  with the latest timestamp.

  If there are multiple values with the latest timestamp, return the maximum of
  these values.

  Arguments:
    RUN_ID: ID of the run to fetch. Must be provided.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getRunReq.RunId = new(args[0])

		sdkReq, err := experiments.GetRunRequestToSDK(&getRunReq)
		if err != nil {
			return err
		}
		response, err := w.Experiments.GetRun(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := experiments.GetRunResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getRunOverrides {
		fn(cmd, &getRunReq)
	}

	return cmd
}

// start list-artifacts command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listArtifactsOverrides []func(
	*cobra.Command,
	*experiments.ListArtifactsRequest,
)

func newListArtifacts() *cobra.Command {
	cmd := &cobra.Command{}
	var listArtifactsReq experiments.ListArtifactsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listArtifactsLimit int

	flags.StringPtrVar(cmd.Flags(), &listArtifactsReq.Path, "path", `Filter artifacts matching this path (a relative path from the root artifact directory).`)
	flags.StringPtrVar(cmd.Flags(), &listArtifactsReq.RunId, "run-id", `ID of the run whose artifacts to list.`)
	flags.StringPtrVar(cmd.Flags(), &listArtifactsReq.RunUuid, "run-uuid", `[Deprecated, use run_id instead] ID of the run whose artifacts to list.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listArtifactsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listArtifactsReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-artifacts"
	cmd.Short = `List artifacts.`
	cmd.Long = `List artifacts.

  List artifacts for a run. Takes an optional artifact_path prefix which if
  specified, the response contains only artifacts with the specified prefix. A
  maximum of 1000 artifacts will be retrieved for UC Volumes. Please call
  /api/2.0/fs/directories{directory_path} for listing artifacts in UC Volumes,
  which supports pagination. See [List directory contents | Files
  API](/api/workspace/files/listdirectorycontents).`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := experiments.ListArtifactsRequestToSDK(&listArtifactsReq)
		if err != nil {
			return err
		}
		if listArtifactsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listArtifactsLimit)
		}
		if listArtifactsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listArtifactsLimit)
		}
		response := w.Experiments.ListArtifactsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, experiments.FileInfoFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listArtifactsOverrides {
		fn(cmd, &listArtifactsReq)
	}

	return cmd
}

// start list-experiments command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listExperimentsOverrides []func(
	*cobra.Command,
	*experiments.ListExperimentsRequest,
)

func newListExperiments() *cobra.Command {
	cmd := &cobra.Command{}
	var listExperimentsReq experiments.ListExperimentsRequest

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listExperimentsLimit int

	flags.Int64PtrVar(cmd.Flags(), &listExperimentsReq.MaxResults, "max-results", `Maximum number of experiments desired.`)
	flags.EnumVar(cmd.Flags(), &listExperimentsReq.ViewType, "view-type", `Qualifier for type of experiments to be returned. Supported values: [ACTIVE_ONLY, ALL, DELETED_ONLY]`, "ViewType", "ACTIVE_ONLY", "ALL", "DELETED_ONLY")

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listExperimentsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listExperimentsReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-experiments"
	cmd.Short = `List experiments.`
	cmd.Long = `List experiments.

  Gets a list of all experiments.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		sdkReq, err := experiments.ListExperimentsRequestToSDK(&listExperimentsReq)
		if err != nil {
			return err
		}
		if listExperimentsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listExperimentsLimit)
		}
		if listExperimentsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listExperimentsLimit)
		}
		response := w.Experiments.ListExperimentsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, experiments.ExperimentFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listExperimentsOverrides {
		fn(cmd, &listExperimentsReq)
	}

	return cmd
}

// start log-batch command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var logBatchOverrides []func(
	*cobra.Command,
	*experiments.LogBatchRequest,
)

func newLogBatch() *cobra.Command {
	cmd := &cobra.Command{}
	var logBatchReq experiments.LogBatchRequest

	var logBatchJson flags.JsonFlag

	cmd.Flags().Var(&logBatchJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: metrics
	// TODO: array: params
	flags.StringPtrVar(cmd.Flags(), &logBatchReq.RunId, "run-id", `ID of the run to log under.`)
	// TODO: array: tags

	cmd.Use = "log-batch"
	cmd.Short = `Log a batch of metrics/params/tags for a run.`
	cmd.Long = `Log a batch of metrics/params/tags for a run.

  Logs a batch of metrics, params, and tags for a run. If any data failed to be
  persisted, the server will respond with an error (non-200 status code).

  In case of error (due to internal server error or an invalid request), partial
  data may be written.

  You can write metrics, params, and tags in interleaving fashion, but within a
  given entity type are guaranteed to follow the order specified in the request
  body.

  The overwrite behavior for metrics, params, and tags is as follows:

  * Metrics: metric values are never overwritten. Logging a metric (key, value,
  timestamp) appends to the set of values for the metric with the provided key.

  * Tags: tag values can be overwritten by successive writes to the same tag
  key. That is, if multiple tag values with the same key are provided in the
  same API request, the last-provided tag value is written. Logging the same tag
  (key, value) is permitted. Specifically, logging a tag is idempotent.

  * Parameters: once written, param values cannot be changed (attempting to
  overwrite a param value will result in an error). However, logging the same
  param (key, value) is permitted. Specifically, logging a param is idempotent.

  Request Limits ------------------------------- A single JSON-serialized API
  request may be up to 1 MB in size and contain:

  * No more than 1000 metrics, params, and tags in total

  * Up to 1000 metrics

  * Up to 100 params

  * Up to 100 tags

  For example, a valid request might contain 900 metrics, 50 params, and 50
  tags, but logging 900 metrics, 50 params, and 51 tags is invalid.

  The following limits also apply to metric, param, and tag keys and values:

  * Metric keys, param keys, and tag keys can be up to 250 characters in length

  * Parameter and tag values can be up to 250 characters in length`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := logBatchJson.Unmarshal(&logBatchReq)
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

		sdkReq, err := experiments.LogBatchRequestToSDK(&logBatchReq)
		if err != nil {
			return err
		}
		response, err := w.Experiments.LogBatch(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range logBatchOverrides {
		fn(cmd, &logBatchReq)
	}

	return cmd
}

// start log-inputs command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var logInputsOverrides []func(
	*cobra.Command,
	*experiments.LogInputsRequest,
)

func newLogInputs() *cobra.Command {
	cmd := &cobra.Command{}
	var logInputsReq experiments.LogInputsRequest

	var logInputsJson flags.JsonFlag

	cmd.Flags().Var(&logInputsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: datasets
	// TODO: array: models

	cmd.Use = "log-inputs RUN_ID"
	cmd.Short = `Log inputs to a run.`
	cmd.Long = `Log inputs to a run.

  Logs inputs, such as datasets and models, to an MLflow Run.

  Arguments:
    RUN_ID: ID of the run to log under`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'run_id' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := logInputsJson.Unmarshal(&logInputsReq)
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
			logInputsReq.RunId = new(args[0])
		}

		sdkReq, err := experiments.LogInputsRequestToSDK(&logInputsReq)
		if err != nil {
			return err
		}
		response, err := w.Experiments.LogInputs(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range logInputsOverrides {
		fn(cmd, &logInputsReq)
	}

	return cmd
}

// start log-logged-model-params command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var logLoggedModelParamsOverrides []func(
	*cobra.Command,
	*experiments.LogLoggedModelParamsRequest,
)

func newLogLoggedModelParams() *cobra.Command {
	cmd := &cobra.Command{}
	var logLoggedModelParamsReq experiments.LogLoggedModelParamsRequest

	var logLoggedModelParamsJson flags.JsonFlag

	cmd.Flags().Var(&logLoggedModelParamsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: params

	cmd.Use = "log-logged-model-params MODEL_ID"
	cmd.Short = `Log params for a logged model.`
	cmd.Long = `Log params for a logged model.

  Logs params for a logged model. A param is a key-value pair (string key,
  string value). Examples include hyperparameters used for ML model training. A
  param can be logged only once for a logged model, and attempting to overwrite
  an existing param with a different value will result in an error

  Arguments:
    MODEL_ID: The ID of the logged model to log params for.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := logLoggedModelParamsJson.Unmarshal(&logLoggedModelParamsReq)
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
		logLoggedModelParamsReq.ModelId = new(args[0])

		sdkReq, err := experiments.LogLoggedModelParamsRequestToSDK(&logLoggedModelParamsReq)
		if err != nil {
			return err
		}
		response, err := w.Experiments.LogLoggedModelParams(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range logLoggedModelParamsOverrides {
		fn(cmd, &logLoggedModelParamsReq)
	}

	return cmd
}

// start log-metric command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var logMetricOverrides []func(
	*cobra.Command,
	*experiments.LogMetricRequest,
)

func newLogMetric() *cobra.Command {
	cmd := &cobra.Command{}
	var logMetricReq experiments.LogMetricRequest

	var logMetricJson flags.JsonFlag

	cmd.Flags().Var(&logMetricJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &logMetricReq.DatasetDigest, "dataset-digest", `Dataset digest of the dataset associated with the metric, e.g.`)
	flags.StringPtrVar(cmd.Flags(), &logMetricReq.DatasetName, "dataset-name", `The name of the dataset associated with the metric.`)
	flags.StringPtrVar(cmd.Flags(), &logMetricReq.ModelId, "model-id", `ID of the logged model associated with the metric, if applicable.`)
	flags.StringPtrVar(cmd.Flags(), &logMetricReq.RunId, "run-id", `ID of the run under which to log the metric.`)
	flags.StringPtrVar(cmd.Flags(), &logMetricReq.RunUuid, "run-uuid", `[Deprecated, use run_id instead] ID of the run under which to log the metric.`)
	flags.Int64PtrVar(cmd.Flags(), &logMetricReq.Step, "step", `Step at which to log the metric.`)

	cmd.Use = "log-metric KEY VALUE TIMESTAMP"
	cmd.Short = `Log a metric for a run.`
	cmd.Long = `Log a metric for a run.

  Log a metric for a run. A metric is a key-value pair (string key, float value)
  with an associated timestamp. Examples include the various metrics that
  represent ML model accuracy. A metric can be logged multiple times.

  Arguments:
    KEY: Name of the metric.
    VALUE: Double value of the metric being logged.
    TIMESTAMP: Unix timestamp in milliseconds at the time metric was logged.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'key', 'value', 'timestamp' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := logMetricJson.Unmarshal(&logMetricReq)
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
			logMetricReq.Key = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			var logMetricValueArg float64
			_, err = fmt.Sscan(args[1], &logMetricValueArg)
			if err != nil {
				return fmt.Errorf("invalid VALUE: %s", args[1])
			}
			logMetricReq.Value = &logMetricValueArg
		}
		if !cmd.Flags().Changed("json") {
			var logMetricTimestampArg int64
			_, err = fmt.Sscan(args[2], &logMetricTimestampArg)
			if err != nil {
				return fmt.Errorf("invalid TIMESTAMP: %s", args[2])
			}
			logMetricReq.Timestamp = &logMetricTimestampArg
		}

		sdkReq, err := experiments.LogMetricRequestToSDK(&logMetricReq)
		if err != nil {
			return err
		}
		response, err := w.Experiments.LogMetric(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range logMetricOverrides {
		fn(cmd, &logMetricReq)
	}

	return cmd
}

// start log-model command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var logModelOverrides []func(
	*cobra.Command,
	*experiments.LogModelRequest,
)

func newLogModel() *cobra.Command {
	cmd := &cobra.Command{}
	var logModelReq experiments.LogModelRequest

	var logModelJson flags.JsonFlag

	cmd.Flags().Var(&logModelJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &logModelReq.ModelJson, "model-json", `MLmodel file in json format.`)
	flags.StringPtrVar(cmd.Flags(), &logModelReq.RunId, "run-id", `ID of the run to log under.`)

	cmd.Use = "log-model"
	cmd.Short = `Log a model.`
	cmd.Long = `Log a model.

  **Note:** the [Create a logged
  model](/api/workspace/experiments/createloggedmodel) API replaces this
  endpoint.

  Log a model to an MLflow Run.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := logModelJson.Unmarshal(&logModelReq)
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

		sdkReq, err := experiments.LogModelRequestToSDK(&logModelReq)
		if err != nil {
			return err
		}
		response, err := w.Experiments.LogModel(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range logModelOverrides {
		fn(cmd, &logModelReq)
	}

	return cmd
}

// start log-outputs command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var logOutputsOverrides []func(
	*cobra.Command,
	*experiments.LogOutputsRequest,
)

func newLogOutputs() *cobra.Command {
	cmd := &cobra.Command{}
	var logOutputsReq experiments.LogOutputsRequest

	var logOutputsJson flags.JsonFlag

	cmd.Flags().Var(&logOutputsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: models

	cmd.Use = "log-outputs RUN_ID"
	cmd.Short = `Log outputs from a run.`
	cmd.Long = `Log outputs from a run.

  Logs outputs, such as models, from an MLflow Run.

  Arguments:
    RUN_ID: The ID of the Run from which to log outputs.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'run_id' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := logOutputsJson.Unmarshal(&logOutputsReq)
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
			logOutputsReq.RunId = new(args[0])
		}

		sdkReq, err := experiments.LogOutputsRequestToSDK(&logOutputsReq)
		if err != nil {
			return err
		}
		response, err := w.Experiments.LogOutputs(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range logOutputsOverrides {
		fn(cmd, &logOutputsReq)
	}

	return cmd
}

// start log-param command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var logParamOverrides []func(
	*cobra.Command,
	*experiments.LogParamRequest,
)

func newLogParam() *cobra.Command {
	cmd := &cobra.Command{}
	var logParamReq experiments.LogParamRequest

	var logParamJson flags.JsonFlag

	cmd.Flags().Var(&logParamJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &logParamReq.RunId, "run-id", `ID of the run under which to log the param.`)
	flags.StringPtrVar(cmd.Flags(), &logParamReq.RunUuid, "run-uuid", `[Deprecated, use run_id instead] ID of the run under which to log the param.`)

	cmd.Use = "log-param KEY VALUE"
	cmd.Short = `Log a param for a run.`
	cmd.Long = `Log a param for a run.

  Logs a param used for a run. A param is a key-value pair (string key, string
  value). Examples include hyperparameters used for ML model training and
  constant dates and values used in an ETL pipeline. A param can be logged only
  once for a run.

  Arguments:
    KEY: Name of the param. Maximum size is 255 bytes.
    VALUE: String value of the param being logged. Maximum size is 500 bytes.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'key', 'value' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := logParamJson.Unmarshal(&logParamReq)
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
			logParamReq.Key = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			logParamReq.Value = new(args[1])
		}

		sdkReq, err := experiments.LogParamRequestToSDK(&logParamReq)
		if err != nil {
			return err
		}
		response, err := w.Experiments.LogParam(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range logParamOverrides {
		fn(cmd, &logParamReq)
	}

	return cmd
}

// start restore-experiment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var restoreExperimentOverrides []func(
	*cobra.Command,
	*experiments.RestoreExperimentRequest,
)

func newRestoreExperiment() *cobra.Command {
	cmd := &cobra.Command{}
	var restoreExperimentReq experiments.RestoreExperimentRequest

	var restoreExperimentJson flags.JsonFlag

	cmd.Flags().Var(&restoreExperimentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "restore-experiment EXPERIMENT_ID"
	cmd.Short = `Restore an experiment.`
	cmd.Long = `Restore an experiment.

  Restore an experiment marked for deletion. This also restores associated
  metadata, runs, metrics, params, and tags. If experiment uses FileStore,
  underlying artifacts associated with experiment are also restored.

  Throws RESOURCE_DOES_NOT_EXIST if experiment was never created or was
  permanently deleted.

  Arguments:
    EXPERIMENT_ID: ID of the associated experiment.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'experiment_id' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := restoreExperimentJson.Unmarshal(&restoreExperimentReq)
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
			restoreExperimentReq.ExperimentId = new(args[0])
		}

		sdkReq, err := experiments.RestoreExperimentRequestToSDK(&restoreExperimentReq)
		if err != nil {
			return err
		}
		response, err := w.Experiments.RestoreExperiment(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range restoreExperimentOverrides {
		fn(cmd, &restoreExperimentReq)
	}

	return cmd
}

// start restore-run command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var restoreRunOverrides []func(
	*cobra.Command,
	*experiments.RestoreRunRequest,
)

func newRestoreRun() *cobra.Command {
	cmd := &cobra.Command{}
	var restoreRunReq experiments.RestoreRunRequest

	var restoreRunJson flags.JsonFlag

	cmd.Flags().Var(&restoreRunJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "restore-run RUN_ID"
	cmd.Short = `Restore a run.`
	cmd.Long = `Restore a run.

  Restores a deleted run. This also restores associated metadata, runs, metrics,
  params, and tags.

  Throws RESOURCE_DOES_NOT_EXIST if the run was never created or was
  permanently deleted.

  Arguments:
    RUN_ID: ID of the run to restore.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'run_id' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := restoreRunJson.Unmarshal(&restoreRunReq)
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
			restoreRunReq.RunId = new(args[0])
		}

		sdkReq, err := experiments.RestoreRunRequestToSDK(&restoreRunReq)
		if err != nil {
			return err
		}
		response, err := w.Experiments.RestoreRun(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range restoreRunOverrides {
		fn(cmd, &restoreRunReq)
	}

	return cmd
}

// start restore-runs command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var restoreRunsOverrides []func(
	*cobra.Command,
	*experiments.RestoreRunsRequest,
)

func newRestoreRuns() *cobra.Command {
	cmd := &cobra.Command{}
	var restoreRunsReq experiments.RestoreRunsRequest

	var restoreRunsJson flags.JsonFlag

	cmd.Flags().Var(&restoreRunsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.IntPtrVar(cmd.Flags(), &restoreRunsReq.MaxRuns, "max-runs", `An optional positive integer indicating the maximum number of runs to restore.`)

	cmd.Use = "restore-runs EXPERIMENT_ID MIN_TIMESTAMP_MILLIS"
	cmd.Short = `Restore runs by deletion time.`
	cmd.Long = `Restore runs by deletion time.

  Bulk restore runs in an experiment that were deleted no earlier than the
  specified timestamp. Restores at most max_runs per request. See the
  cloud-specific MLflow runs documentation for a Python client example.

  Arguments:
    EXPERIMENT_ID: The ID of the experiment containing the runs to restore.
    MIN_TIMESTAMP_MILLIS: The minimum deletion timestamp in milliseconds since the UNIX epoch for
      restoring runs. Only runs deleted no earlier than this timestamp are
      restored.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'experiment_id', 'min_timestamp_millis' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := restoreRunsJson.Unmarshal(&restoreRunsReq)
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
			restoreRunsReq.ExperimentId = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			var restoreRunsMinTimestampMillisArg int64
			_, err = fmt.Sscan(args[1], &restoreRunsMinTimestampMillisArg)
			if err != nil {
				return fmt.Errorf("invalid MIN_TIMESTAMP_MILLIS: %s", args[1])
			}
			restoreRunsReq.MinTimestampMillis = &restoreRunsMinTimestampMillisArg
		}

		sdkReq, err := experiments.RestoreRunsRequestToSDK(&restoreRunsReq)
		if err != nil {
			return err
		}
		response, err := w.Experiments.RestoreRuns(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := experiments.RestoreRunsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range restoreRunsOverrides {
		fn(cmd, &restoreRunsReq)
	}

	return cmd
}

// start search-experiments command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var searchExperimentsOverrides []func(
	*cobra.Command,
	*experiments.SearchExperimentsRequest,
)

func newSearchExperiments() *cobra.Command {
	cmd := &cobra.Command{}
	var searchExperimentsReq experiments.SearchExperimentsRequest

	var searchExperimentsJson flags.JsonFlag
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var searchExperimentsLimit int

	cmd.Flags().Var(&searchExperimentsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &searchExperimentsReq.Filter, "filter", `String representing a SQL filter condition (e.g.`)
	flags.Int64PtrVar(cmd.Flags(), &searchExperimentsReq.MaxResults, "max-results", `Maximum number of experiments desired.`)
	// TODO: array: order_by
	flags.EnumVar(cmd.Flags(), &searchExperimentsReq.ViewType, "view-type", `Qualifier for type of experiments to be returned. Supported values: [ACTIVE_ONLY, ALL, DELETED_ONLY]`, "ViewType", "ACTIVE_ONLY", "ALL", "DELETED_ONLY")

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&searchExperimentsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &searchExperimentsReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "search-experiments"
	cmd.Short = `Search experiments.`
	cmd.Long = `Search experiments.

  Searches for experiments that satisfy specified search criteria.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := searchExperimentsJson.Unmarshal(&searchExperimentsReq)
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

		sdkReq, err := experiments.SearchExperimentsRequestToSDK(&searchExperimentsReq)
		if err != nil {
			return err
		}
		if searchExperimentsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", searchExperimentsLimit)
		}
		if searchExperimentsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, searchExperimentsLimit)
		}
		response := w.Experiments.SearchExperimentsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, experiments.ExperimentFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range searchExperimentsOverrides {
		fn(cmd, &searchExperimentsReq)
	}

	return cmd
}

// start search-logged-models command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var searchLoggedModelsOverrides []func(
	*cobra.Command,
	*experiments.SearchLoggedModelsRequest,
)

func newSearchLoggedModels() *cobra.Command {
	cmd := &cobra.Command{}
	var searchLoggedModelsReq experiments.SearchLoggedModelsRequest

	var searchLoggedModelsJson flags.JsonFlag

	cmd.Flags().Var(&searchLoggedModelsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: datasets
	// TODO: array: experiment_ids
	flags.StringPtrVar(cmd.Flags(), &searchLoggedModelsReq.Filter, "filter", `A filter expression over logged model info and data that allows returning a subset of logged models.`)
	flags.IntPtrVar(cmd.Flags(), &searchLoggedModelsReq.MaxResults, "max-results", `The maximum number of Logged Models to return.`)
	// TODO: array: order_by
	flags.StringPtrVar(cmd.Flags(), &searchLoggedModelsReq.PageToken, "page-token", `The token indicating the page of logged models to fetch.`)

	cmd.Use = "search-logged-models"
	cmd.Short = `Search logged models.`
	cmd.Long = `Search logged models.

  Search for Logged Models that satisfy specified search criteria.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := searchLoggedModelsJson.Unmarshal(&searchLoggedModelsReq)
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

		sdkReq, err := experiments.SearchLoggedModelsRequestToSDK(&searchLoggedModelsReq)
		if err != nil {
			return err
		}
		response, err := w.Experiments.SearchLoggedModels(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := experiments.SearchLoggedModelsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range searchLoggedModelsOverrides {
		fn(cmd, &searchLoggedModelsReq)
	}

	return cmd
}

// start search-runs command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var searchRunsOverrides []func(
	*cobra.Command,
	*experiments.SearchRunsRequest,
)

func newSearchRuns() *cobra.Command {
	cmd := &cobra.Command{}
	var searchRunsReq experiments.SearchRunsRequest

	var searchRunsJson flags.JsonFlag
	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var searchRunsLimit int

	cmd.Flags().Var(&searchRunsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: experiment_ids
	flags.StringPtrVar(cmd.Flags(), &searchRunsReq.Filter, "filter", `A filter expression over params, metrics, and tags, that allows returning a subset of runs.`)
	flags.IntPtrVar(cmd.Flags(), &searchRunsReq.MaxResults, "max-results", `Maximum number of runs desired.`)
	// TODO: array: order_by
	flags.EnumVar(cmd.Flags(), &searchRunsReq.RunViewType, "run-view-type", `Whether to display only active, only deleted, or all runs. Supported values: [ACTIVE_ONLY, ALL, DELETED_ONLY]`, "ViewType", "ACTIVE_ONLY", "ALL", "DELETED_ONLY")

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&searchRunsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &searchRunsReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "search-runs"
	cmd.Short = `Search for runs.`
	cmd.Long = `Search for runs.

  Searches for runs that satisfy expressions.

  Search expressions can use mlflowMetric and mlflowParam keys.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := searchRunsJson.Unmarshal(&searchRunsReq)
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

		sdkReq, err := experiments.SearchRunsRequestToSDK(&searchRunsReq)
		if err != nil {
			return err
		}
		if searchRunsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", searchRunsLimit)
		}
		if searchRunsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, searchRunsLimit)
		}
		response := w.Experiments.SearchRunsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, experiments.RunFromSDK))
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range searchRunsOverrides {
		fn(cmd, &searchRunsReq)
	}

	return cmd
}

// start set-experiment-tag command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var setExperimentTagOverrides []func(
	*cobra.Command,
	*experiments.SetExperimentTagRequest,
)

func newSetExperimentTag() *cobra.Command {
	cmd := &cobra.Command{}
	var setExperimentTagReq experiments.SetExperimentTagRequest

	var setExperimentTagJson flags.JsonFlag

	cmd.Flags().Var(&setExperimentTagJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "set-experiment-tag EXPERIMENT_ID KEY VALUE"
	cmd.Short = `Set a tag for an experiment.`
	cmd.Long = `Set a tag for an experiment.

  Sets a tag on an experiment. Experiment tags are metadata that can be updated.

  Arguments:
    EXPERIMENT_ID: ID of the experiment under which to log the tag. Must be provided.
    KEY: Name of the tag. Keys up to 250 bytes in size are supported.
    VALUE: String value of the tag being logged. Values up to 64KB in size are
      supported.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'experiment_id', 'key', 'value' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := setExperimentTagJson.Unmarshal(&setExperimentTagReq)
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
			setExperimentTagReq.ExperimentId = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			setExperimentTagReq.Key = new(args[1])
		}
		if !cmd.Flags().Changed("json") {
			setExperimentTagReq.Value = new(args[2])
		}

		sdkReq, err := experiments.SetExperimentTagRequestToSDK(&setExperimentTagReq)
		if err != nil {
			return err
		}
		response, err := w.Experiments.SetExperimentTag(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range setExperimentTagOverrides {
		fn(cmd, &setExperimentTagReq)
	}

	return cmd
}

// start set-logged-model-tags command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var setLoggedModelTagsOverrides []func(
	*cobra.Command,
	*experiments.SetLoggedModelTagsRequest,
)

func newSetLoggedModelTags() *cobra.Command {
	cmd := &cobra.Command{}
	var setLoggedModelTagsReq experiments.SetLoggedModelTagsRequest

	var setLoggedModelTagsJson flags.JsonFlag

	cmd.Flags().Var(&setLoggedModelTagsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: tags

	cmd.Use = "set-logged-model-tags MODEL_ID"
	cmd.Short = `Set tags for a logged model.`
	cmd.Long = `Set tags for a logged model.

  Arguments:
    MODEL_ID: The ID of the logged model to set the tags on.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := setLoggedModelTagsJson.Unmarshal(&setLoggedModelTagsReq)
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
		setLoggedModelTagsReq.ModelId = new(args[0])

		sdkReq, err := experiments.SetLoggedModelTagsRequestToSDK(&setLoggedModelTagsReq)
		if err != nil {
			return err
		}
		response, err := w.Experiments.SetLoggedModelTags(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range setLoggedModelTagsOverrides {
		fn(cmd, &setLoggedModelTagsReq)
	}

	return cmd
}

// start set-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var setPermissionsOverrides []func(
	*cobra.Command,
	*accessmanagement.SetObjectPermissionsRequest,
)

func newSetPermissions() *cobra.Command {
	cmd := &cobra.Command{}
	var setPermissionsReq accessmanagement.SetObjectPermissionsRequest

	var setPermissionsJson flags.JsonFlag

	cmd.Flags().Var(&setPermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "set-permissions EXPERIMENT_ID"
	cmd.Short = `Set experiment permissions.`
	cmd.Long = `Set experiment permissions.

  Sets permissions on an object, replacing existing permissions if they exist.
  Deletes all direct permissions if none are specified. Objects can inherit
  permissions from their root object.

  Arguments:
    EXPERIMENT_ID: The experiment for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := setPermissionsJson.Unmarshal(&setPermissionsReq)
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
		setPermissionsReq.RequestObjectId = new(args[0])

		setPermissionsReq.RequestObjectType = new("experiments")

		sdkReq, err := accessmanagement.SetObjectPermissionsRequestToSDK(&setPermissionsReq)
		if err != nil {
			return err
		}
		response, err := w.AccessManagement.SetObjectPermissions(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.PermissionsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start set-tag command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var setTagOverrides []func(
	*cobra.Command,
	*experiments.SetTagRequest,
)

func newSetTag() *cobra.Command {
	cmd := &cobra.Command{}
	var setTagReq experiments.SetTagRequest

	var setTagJson flags.JsonFlag

	cmd.Flags().Var(&setTagJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &setTagReq.RunId, "run-id", `ID of the run under which to log the tag.`)
	flags.StringPtrVar(cmd.Flags(), &setTagReq.RunUuid, "run-uuid", `[Deprecated, use run_id instead] ID of the run under which to log the tag.`)

	cmd.Use = "set-tag KEY VALUE"
	cmd.Short = `Set a tag for a run.`
	cmd.Long = `Set a tag for a run.

  Sets a tag on a run. Tags are run metadata that can be updated during a run
  and after a run completes.

  Arguments:
    KEY: Name of the tag. Keys up to 250 bytes in size are supported.
    VALUE: String value of the tag being logged. Values up to 64KB in size are
      supported.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'key', 'value' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := setTagJson.Unmarshal(&setTagReq)
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
			setTagReq.Key = new(args[0])
		}
		if !cmd.Flags().Changed("json") {
			setTagReq.Value = new(args[1])
		}

		sdkReq, err := experiments.SetTagRequestToSDK(&setTagReq)
		if err != nil {
			return err
		}
		response, err := w.Experiments.SetTag(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range setTagOverrides {
		fn(cmd, &setTagReq)
	}

	return cmd
}

// start update-experiment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateExperimentOverrides []func(
	*cobra.Command,
	*experiments.UpdateExperimentRequest,
)

func newUpdateExperiment() *cobra.Command {
	cmd := &cobra.Command{}
	var updateExperimentReq experiments.UpdateExperimentRequest

	var updateExperimentJson flags.JsonFlag

	cmd.Flags().Var(&updateExperimentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.StringPtrVar(cmd.Flags(), &updateExperimentReq.NewName, "new-name", `If provided, the experiment's name is changed to the new name.`)

	cmd.Use = "update-experiment EXPERIMENT_ID"
	cmd.Short = `Update an experiment.`
	cmd.Long = `Update an experiment.

  Updates experiment metadata.

  Arguments:
    EXPERIMENT_ID: ID of the associated experiment.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'experiment_id' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateExperimentJson.Unmarshal(&updateExperimentReq)
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
			updateExperimentReq.ExperimentId = new(args[0])
		}

		sdkReq, err := experiments.UpdateExperimentRequestToSDK(&updateExperimentReq)
		if err != nil {
			return err
		}
		response, err := w.Experiments.UpdateExperiment(ctx, *sdkReq)
		if err != nil {
			return err
		}

		_ = response
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateExperimentOverrides {
		fn(cmd, &updateExperimentReq)
	}

	return cmd
}

// start update-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updatePermissionsOverrides []func(
	*cobra.Command,
	*accessmanagement.UpdateObjectPermissionsRequest,
)

func newUpdatePermissions() *cobra.Command {
	cmd := &cobra.Command{}
	var updatePermissionsReq accessmanagement.UpdateObjectPermissionsRequest

	var updatePermissionsJson flags.JsonFlag

	cmd.Flags().Var(&updatePermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "update-permissions EXPERIMENT_ID"
	cmd.Short = `Update experiment permissions.`
	cmd.Long = `Update experiment permissions.

  Updates the permissions on an experiment. Experiments can inherit permissions
  from their root object.

  Arguments:
    EXPERIMENT_ID: The experiment for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updatePermissionsJson.Unmarshal(&updatePermissionsReq)
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
		updatePermissionsReq.RequestObjectId = new(args[0])

		updatePermissionsReq.RequestObjectType = new("experiments")

		sdkReq, err := accessmanagement.UpdateObjectPermissionsRequestToSDK(&updatePermissionsReq)
		if err != nil {
			return err
		}
		response, err := w.AccessManagement.UpdateObjectPermissions(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := accessmanagement.PermissionsResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

// start update-run command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateRunOverrides []func(
	*cobra.Command,
	*experiments.UpdateRunRequest,
)

func newUpdateRun() *cobra.Command {
	cmd := &cobra.Command{}
	var updateRunReq experiments.UpdateRunRequest

	var updateRunJson flags.JsonFlag

	cmd.Flags().Var(&updateRunJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.Int64PtrVar(cmd.Flags(), &updateRunReq.EndTime, "end-time", `Unix timestamp in milliseconds of when the run ended.`)
	flags.StringPtrVar(cmd.Flags(), &updateRunReq.RunId, "run-id", `ID of the run to update.`)
	flags.StringPtrVar(cmd.Flags(), &updateRunReq.RunName, "run-name", `Updated name of the run.`)
	flags.StringPtrVar(cmd.Flags(), &updateRunReq.RunUuid, "run-uuid", `[Deprecated, use run_id instead] ID of the run to update.`)
	flags.EnumVar(cmd.Flags(), &updateRunReq.Status, "status", `Updated status of the run. Supported values: [FAILED, FINISHED, KILLED, RUNNING, SCHEDULED]`, "UpdateRunStatus", "FAILED", "FINISHED", "KILLED", "RUNNING", "SCHEDULED")

	cmd.Use = "update-run"
	cmd.Short = `Update a run.`
	cmd.Long = `Update a run.

  Updates run metadata.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateRunJson.Unmarshal(&updateRunReq)
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

		sdkReq, err := experiments.UpdateRunRequestToSDK(&updateRunReq)
		if err != nil {
			return err
		}
		response, err := w.Experiments.UpdateRun(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := experiments.UpdateRunResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateRunOverrides {
		fn(cmd, &updateRunReq)
	}

	return cmd
}

// end service Experiments
