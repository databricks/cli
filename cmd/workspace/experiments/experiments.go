// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package experiments

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
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
		Annotations: map[string]string{
			"package": "ml",
		},
	}

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
	*ml.CreateExperiment,
)

func newCreateExperiment() *cobra.Command {
	cmd := &cobra.Command{}

	var createExperimentReq ml.CreateExperiment
	var createExperimentJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&createExperimentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createExperimentReq.ArtifactLocation, "artifact-location", createExperimentReq.ArtifactLocation, `Location where all artifacts for the experiment are stored.`)
	// TODO: array: tags

	cmd.Use = "create-experiment NAME"
	cmd.Short = `Create experiment.`
	cmd.Long = `Create experiment.
  
  Creates an experiment with a name. Returns the ID of the newly created
  experiment. Validates that another experiment with the same name does not
  already exist and fails if another experiment with the same name already
  exists.
  
  Throws RESOURCE_ALREADY_EXISTS if a experiment with the given name exists.

  Arguments:
    NAME: Experiment name.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := cobra.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'name' in your JSON input")
			}
			return nil
		}
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = createExperimentJson.Unmarshal(&createExperimentReq)
			if err != nil {
				return err
			}
		}
		if !cmd.Flags().Changed("json") {
			createExperimentReq.Name = args[0]
		}

		response, err := w.Experiments.CreateExperiment(ctx, createExperimentReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newCreateExperiment())
	})
}

// start create-run command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createRunOverrides []func(
	*cobra.Command,
	*ml.CreateRun,
)

func newCreateRun() *cobra.Command {
	cmd := &cobra.Command{}

	var createRunReq ml.CreateRun
	var createRunJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&createRunJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createRunReq.ExperimentId, "experiment-id", createRunReq.ExperimentId, `ID of the associated experiment.`)
	cmd.Flags().Int64Var(&createRunReq.StartTime, "start-time", createRunReq.StartTime, `Unix timestamp in milliseconds of when the run started.`)
	// TODO: array: tags
	cmd.Flags().StringVar(&createRunReq.UserId, "user-id", createRunReq.UserId, `ID of the user executing the run.`)

	cmd.Use = "create-run"
	cmd.Short = `Create a run.`
	cmd.Long = `Create a run.
  
  Creates a new run within an experiment. A run is usually a single execution of
  a machine learning or data ETL pipeline. MLflow uses runs to track the
  mlflowParam, mlflowMetric and mlflowRunTag associated with a single
  execution.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = createRunJson.Unmarshal(&createRunReq)
			if err != nil {
				return err
			}
		}

		response, err := w.Experiments.CreateRun(ctx, createRunReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newCreateRun())
	})
}

// start delete-experiment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteExperimentOverrides []func(
	*cobra.Command,
	*ml.DeleteExperiment,
)

func newDeleteExperiment() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteExperimentReq ml.DeleteExperiment
	var deleteExperimentJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&deleteExperimentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "delete-experiment EXPERIMENT_ID"
	cmd.Short = `Delete an experiment.`
	cmd.Long = `Delete an experiment.
  
  Marks an experiment and associated metadata, runs, metrics, params, and tags
  for deletion. If the experiment uses FileStore, artifacts associated with
  experiment are also deleted.

  Arguments:
    EXPERIMENT_ID: ID of the associated experiment.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := cobra.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'experiment_id' in your JSON input")
			}
			return nil
		}
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = deleteExperimentJson.Unmarshal(&deleteExperimentReq)
			if err != nil {
				return err
			}
		}
		if !cmd.Flags().Changed("json") {
			deleteExperimentReq.ExperimentId = args[0]
		}

		err = w.Experiments.DeleteExperiment(ctx, deleteExperimentReq)
		if err != nil {
			return err
		}
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newDeleteExperiment())
	})
}

// start delete-run command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteRunOverrides []func(
	*cobra.Command,
	*ml.DeleteRun,
)

func newDeleteRun() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteRunReq ml.DeleteRun
	var deleteRunJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&deleteRunJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "delete-run RUN_ID"
	cmd.Short = `Delete a run.`
	cmd.Long = `Delete a run.
  
  Marks a run for deletion.

  Arguments:
    RUN_ID: ID of the run to delete.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := cobra.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'run_id' in your JSON input")
			}
			return nil
		}
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = deleteRunJson.Unmarshal(&deleteRunReq)
			if err != nil {
				return err
			}
		}
		if !cmd.Flags().Changed("json") {
			deleteRunReq.RunId = args[0]
		}

		err = w.Experiments.DeleteRun(ctx, deleteRunReq)
		if err != nil {
			return err
		}
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newDeleteRun())
	})
}

// start delete-runs command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteRunsOverrides []func(
	*cobra.Command,
	*ml.DeleteRuns,
)

func newDeleteRuns() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteRunsReq ml.DeleteRuns
	var deleteRunsJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&deleteRunsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().IntVar(&deleteRunsReq.MaxRuns, "max-runs", deleteRunsReq.MaxRuns, `An optional positive integer indicating the maximum number of runs to delete.`)

	cmd.Use = "delete-runs EXPERIMENT_ID MAX_TIMESTAMP_MILLIS"
	cmd.Short = `Delete runs by creation time.`
	cmd.Long = `Delete runs by creation time.
  
  Bulk delete runs in an experiment that were created prior to or at the
  specified timestamp. Deletes at most max_runs per request.

  Arguments:
    EXPERIMENT_ID: The ID of the experiment containing the runs to delete.
    MAX_TIMESTAMP_MILLIS: The maximum creation timestamp in milliseconds since the UNIX epoch for
      deleting runs. Only runs created prior to or at this timestamp are
      deleted.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := cobra.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'experiment_id', 'max_timestamp_millis' in your JSON input")
			}
			return nil
		}
		check := cobra.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = deleteRunsJson.Unmarshal(&deleteRunsReq)
			if err != nil {
				return err
			}
		}
		if !cmd.Flags().Changed("json") {
			deleteRunsReq.ExperimentId = args[0]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &deleteRunsReq.MaxTimestampMillis)
			if err != nil {
				return fmt.Errorf("invalid MAX_TIMESTAMP_MILLIS: %s", args[1])
			}
		}

		response, err := w.Experiments.DeleteRuns(ctx, deleteRunsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newDeleteRuns())
	})
}

// start delete-tag command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteTagOverrides []func(
	*cobra.Command,
	*ml.DeleteTag,
)

func newDeleteTag() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteTagReq ml.DeleteTag
	var deleteTagJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&deleteTagJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "delete-tag RUN_ID KEY"
	cmd.Short = `Delete a tag.`
	cmd.Long = `Delete a tag.
  
  Deletes a tag on a run. Tags are run metadata that can be updated during a run
  and after a run completes.

  Arguments:
    RUN_ID: ID of the run that the tag was logged under. Must be provided.
    KEY: Name of the tag. Maximum size is 255 bytes. Must be provided.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := cobra.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'run_id', 'key' in your JSON input")
			}
			return nil
		}
		check := cobra.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = deleteTagJson.Unmarshal(&deleteTagReq)
			if err != nil {
				return err
			}
		}
		if !cmd.Flags().Changed("json") {
			deleteTagReq.RunId = args[0]
		}
		if !cmd.Flags().Changed("json") {
			deleteTagReq.Key = args[1]
		}

		err = w.Experiments.DeleteTag(ctx, deleteTagReq)
		if err != nil {
			return err
		}
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newDeleteTag())
	})
}

// start get-by-name command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getByNameOverrides []func(
	*cobra.Command,
	*ml.GetByNameRequest,
)

func newGetByName() *cobra.Command {
	cmd := &cobra.Command{}

	var getByNameReq ml.GetByNameRequest

	// TODO: short flags

	cmd.Use = "get-by-name EXPERIMENT_NAME"
	cmd.Short = `Get metadata.`
	cmd.Long = `Get metadata.
  
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

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		getByNameReq.ExperimentName = args[0]

		response, err := w.Experiments.GetByName(ctx, getByNameReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newGetByName())
	})
}

// start get-experiment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getExperimentOverrides []func(
	*cobra.Command,
	*ml.GetExperimentRequest,
)

func newGetExperiment() *cobra.Command {
	cmd := &cobra.Command{}

	var getExperimentReq ml.GetExperimentRequest

	// TODO: short flags

	cmd.Use = "get-experiment EXPERIMENT_ID"
	cmd.Short = `Get an experiment.`
	cmd.Long = `Get an experiment.
  
  Gets metadata for an experiment. This method works on deleted experiments.

  Arguments:
    EXPERIMENT_ID: ID of the associated experiment.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		getExperimentReq.ExperimentId = args[0]

		response, err := w.Experiments.GetExperiment(ctx, getExperimentReq)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newGetExperiment())
	})
}

// start get-history command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getHistoryOverrides []func(
	*cobra.Command,
	*ml.GetHistoryRequest,
)

func newGetHistory() *cobra.Command {
	cmd := &cobra.Command{}

	var getHistoryReq ml.GetHistoryRequest

	// TODO: short flags

	cmd.Flags().IntVar(&getHistoryReq.MaxResults, "max-results", getHistoryReq.MaxResults, `Maximum number of Metric records to return per paginated request.`)
	cmd.Flags().StringVar(&getHistoryReq.PageToken, "page-token", getHistoryReq.PageToken, `Token indicating the page of metric histories to fetch.`)
	cmd.Flags().StringVar(&getHistoryReq.RunId, "run-id", getHistoryReq.RunId, `ID of the run from which to fetch metric values.`)
	cmd.Flags().StringVar(&getHistoryReq.RunUuid, "run-uuid", getHistoryReq.RunUuid, `[Deprecated, use run_id instead] ID of the run from which to fetch metric values.`)

	cmd.Use = "get-history METRIC_KEY"
	cmd.Short = `Get history of a given metric within a run.`
	cmd.Long = `Get history of a given metric within a run.
  
  Gets a list of all values for the specified metric for a given run.

  Arguments:
    METRIC_KEY: Name of the metric.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		getHistoryReq.MetricKey = args[0]

		response := w.Experiments.GetHistory(ctx, getHistoryReq)
		return cmdio.Render(ctx, response)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newGetHistory())
	})
}

// start get-permission-levels command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPermissionLevelsOverrides []func(
	*cobra.Command,
	*ml.GetExperimentPermissionLevelsRequest,
)

func newGetPermissionLevels() *cobra.Command {
	cmd := &cobra.Command{}

	var getPermissionLevelsReq ml.GetExperimentPermissionLevelsRequest

	// TODO: short flags

	cmd.Use = "get-permission-levels EXPERIMENT_ID"
	cmd.Short = `Get experiment permission levels.`
	cmd.Long = `Get experiment permission levels.
  
  Gets the permission levels that a user can have on an object.

  Arguments:
    EXPERIMENT_ID: The experiment for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		getPermissionLevelsReq.ExperimentId = args[0]

		response, err := w.Experiments.GetPermissionLevels(ctx, getPermissionLevelsReq)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newGetPermissionLevels())
	})
}

// start get-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPermissionsOverrides []func(
	*cobra.Command,
	*ml.GetExperimentPermissionsRequest,
)

func newGetPermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var getPermissionsReq ml.GetExperimentPermissionsRequest

	// TODO: short flags

	cmd.Use = "get-permissions EXPERIMENT_ID"
	cmd.Short = `Get experiment permissions.`
	cmd.Long = `Get experiment permissions.
  
  Gets the permissions of an experiment. Experiments can inherit permissions
  from their root object.

  Arguments:
    EXPERIMENT_ID: The experiment for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		getPermissionsReq.ExperimentId = args[0]

		response, err := w.Experiments.GetPermissions(ctx, getPermissionsReq)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newGetPermissions())
	})
}

// start get-run command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getRunOverrides []func(
	*cobra.Command,
	*ml.GetRunRequest,
)

func newGetRun() *cobra.Command {
	cmd := &cobra.Command{}

	var getRunReq ml.GetRunRequest

	// TODO: short flags

	cmd.Flags().StringVar(&getRunReq.RunUuid, "run-uuid", getRunReq.RunUuid, `[Deprecated, use run_id instead] ID of the run to fetch.`)

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

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		getRunReq.RunId = args[0]

		response, err := w.Experiments.GetRun(ctx, getRunReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newGetRun())
	})
}

// start list-artifacts command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listArtifactsOverrides []func(
	*cobra.Command,
	*ml.ListArtifactsRequest,
)

func newListArtifacts() *cobra.Command {
	cmd := &cobra.Command{}

	var listArtifactsReq ml.ListArtifactsRequest

	// TODO: short flags

	cmd.Flags().StringVar(&listArtifactsReq.PageToken, "page-token", listArtifactsReq.PageToken, `Token indicating the page of artifact results to fetch.`)
	cmd.Flags().StringVar(&listArtifactsReq.Path, "path", listArtifactsReq.Path, `Filter artifacts matching this path (a relative path from the root artifact directory).`)
	cmd.Flags().StringVar(&listArtifactsReq.RunId, "run-id", listArtifactsReq.RunId, `ID of the run whose artifacts to list.`)
	cmd.Flags().StringVar(&listArtifactsReq.RunUuid, "run-uuid", listArtifactsReq.RunUuid, `[Deprecated, use run_id instead] ID of the run whose artifacts to list.`)

	cmd.Use = "list-artifacts"
	cmd.Short = `Get all artifacts.`
	cmd.Long = `Get all artifacts.
  
  List artifacts for a run. Takes an optional artifact_path prefix. If it is
  specified, the response contains only artifacts with the specified prefix.",`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		response := w.Experiments.ListArtifacts(ctx, listArtifactsReq)
		return cmdio.Render(ctx, response)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newListArtifacts())
	})
}

// start list-experiments command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listExperimentsOverrides []func(
	*cobra.Command,
	*ml.ListExperimentsRequest,
)

func newListExperiments() *cobra.Command {
	cmd := &cobra.Command{}

	var listExperimentsReq ml.ListExperimentsRequest

	// TODO: short flags

	cmd.Flags().IntVar(&listExperimentsReq.MaxResults, "max-results", listExperimentsReq.MaxResults, `Maximum number of experiments desired.`)
	cmd.Flags().StringVar(&listExperimentsReq.PageToken, "page-token", listExperimentsReq.PageToken, `Token indicating the page of experiments to fetch.`)
	cmd.Flags().StringVar(&listExperimentsReq.ViewType, "view-type", listExperimentsReq.ViewType, `Qualifier for type of experiments to be returned.`)

	cmd.Use = "list-experiments"
	cmd.Short = `List experiments.`
	cmd.Long = `List experiments.
  
  Gets a list of all experiments.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		response := w.Experiments.ListExperiments(ctx, listExperimentsReq)
		return cmdio.Render(ctx, response)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newListExperiments())
	})
}

// start log-batch command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var logBatchOverrides []func(
	*cobra.Command,
	*ml.LogBatch,
)

func newLogBatch() *cobra.Command {
	cmd := &cobra.Command{}

	var logBatchReq ml.LogBatch
	var logBatchJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&logBatchJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: metrics
	// TODO: array: params
	cmd.Flags().StringVar(&logBatchReq.RunId, "run-id", logBatchReq.RunId, `ID of the run to log under.`)
	// TODO: array: tags

	cmd.Use = "log-batch"
	cmd.Short = `Log a batch.`
	cmd.Long = `Log a batch.
  
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
  
  * No more than 1000 metrics, params, and tags in total * Up to 1000 metrics *
  Up to 100 params * Up to 100 tags
  
  For example, a valid request might contain 900 metrics, 50 params, and 50
  tags, but logging 900 metrics, 50 params, and 51 tags is invalid.
  
  The following limits also apply to metric, param, and tag keys and values:
  
  * Metric keys, param keys, and tag keys can be up to 250 characters in length
  * Parameter and tag values can be up to 250 characters in length`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = logBatchJson.Unmarshal(&logBatchReq)
			if err != nil {
				return err
			}
		}

		err = w.Experiments.LogBatch(ctx, logBatchReq)
		if err != nil {
			return err
		}
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newLogBatch())
	})
}

// start log-inputs command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var logInputsOverrides []func(
	*cobra.Command,
	*ml.LogInputs,
)

func newLogInputs() *cobra.Command {
	cmd := &cobra.Command{}

	var logInputsReq ml.LogInputs
	var logInputsJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&logInputsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: datasets
	cmd.Flags().StringVar(&logInputsReq.RunId, "run-id", logInputsReq.RunId, `ID of the run to log under.`)

	cmd.Use = "log-inputs"
	cmd.Short = `Log inputs to a run.`
	cmd.Long = `Log inputs to a run.
  
  **NOTE:** Experimental: This API may change or be removed in a future release
  without warning.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = logInputsJson.Unmarshal(&logInputsReq)
			if err != nil {
				return err
			}
		}

		err = w.Experiments.LogInputs(ctx, logInputsReq)
		if err != nil {
			return err
		}
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newLogInputs())
	})
}

// start log-metric command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var logMetricOverrides []func(
	*cobra.Command,
	*ml.LogMetric,
)

func newLogMetric() *cobra.Command {
	cmd := &cobra.Command{}

	var logMetricReq ml.LogMetric
	var logMetricJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&logMetricJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&logMetricReq.RunId, "run-id", logMetricReq.RunId, `ID of the run under which to log the metric.`)
	cmd.Flags().StringVar(&logMetricReq.RunUuid, "run-uuid", logMetricReq.RunUuid, `[Deprecated, use run_id instead] ID of the run under which to log the metric.`)
	cmd.Flags().Int64Var(&logMetricReq.Step, "step", logMetricReq.Step, `Step at which to log the metric.`)

	cmd.Use = "log-metric KEY VALUE TIMESTAMP"
	cmd.Short = `Log a metric.`
	cmd.Long = `Log a metric.
  
  Logs a metric for a run. A metric is a key-value pair (string key, float
  value) with an associated timestamp. Examples include the various metrics that
  represent ML model accuracy. A metric can be logged multiple times.

  Arguments:
    KEY: Name of the metric.
    VALUE: Double value of the metric being logged.
    TIMESTAMP: Unix timestamp in milliseconds at the time metric was logged.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := cobra.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'key', 'value', 'timestamp' in your JSON input")
			}
			return nil
		}
		check := cobra.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = logMetricJson.Unmarshal(&logMetricReq)
			if err != nil {
				return err
			}
		}
		if !cmd.Flags().Changed("json") {
			logMetricReq.Key = args[0]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &logMetricReq.Value)
			if err != nil {
				return fmt.Errorf("invalid VALUE: %s", args[1])
			}
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[2], &logMetricReq.Timestamp)
			if err != nil {
				return fmt.Errorf("invalid TIMESTAMP: %s", args[2])
			}
		}

		err = w.Experiments.LogMetric(ctx, logMetricReq)
		if err != nil {
			return err
		}
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newLogMetric())
	})
}

// start log-model command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var logModelOverrides []func(
	*cobra.Command,
	*ml.LogModel,
)

func newLogModel() *cobra.Command {
	cmd := &cobra.Command{}

	var logModelReq ml.LogModel
	var logModelJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&logModelJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&logModelReq.ModelJson, "model-json", logModelReq.ModelJson, `MLmodel file in json format.`)
	cmd.Flags().StringVar(&logModelReq.RunId, "run-id", logModelReq.RunId, `ID of the run to log under.`)

	cmd.Use = "log-model"
	cmd.Short = `Log a model.`
	cmd.Long = `Log a model.
  
  **NOTE:** Experimental: This API may change or be removed in a future release
  without warning.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = logModelJson.Unmarshal(&logModelReq)
			if err != nil {
				return err
			}
		}

		err = w.Experiments.LogModel(ctx, logModelReq)
		if err != nil {
			return err
		}
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newLogModel())
	})
}

// start log-param command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var logParamOverrides []func(
	*cobra.Command,
	*ml.LogParam,
)

func newLogParam() *cobra.Command {
	cmd := &cobra.Command{}

	var logParamReq ml.LogParam
	var logParamJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&logParamJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&logParamReq.RunId, "run-id", logParamReq.RunId, `ID of the run under which to log the param.`)
	cmd.Flags().StringVar(&logParamReq.RunUuid, "run-uuid", logParamReq.RunUuid, `[Deprecated, use run_id instead] ID of the run under which to log the param.`)

	cmd.Use = "log-param KEY VALUE"
	cmd.Short = `Log a param.`
	cmd.Long = `Log a param.
  
  Logs a param used for a run. A param is a key-value pair (string key, string
  value). Examples include hyperparameters used for ML model training and
  constant dates and values used in an ETL pipeline. A param can be logged only
  once for a run.

  Arguments:
    KEY: Name of the param. Maximum size is 255 bytes.
    VALUE: String value of the param being logged. Maximum size is 500 bytes.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := cobra.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'key', 'value' in your JSON input")
			}
			return nil
		}
		check := cobra.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = logParamJson.Unmarshal(&logParamReq)
			if err != nil {
				return err
			}
		}
		if !cmd.Flags().Changed("json") {
			logParamReq.Key = args[0]
		}
		if !cmd.Flags().Changed("json") {
			logParamReq.Value = args[1]
		}

		err = w.Experiments.LogParam(ctx, logParamReq)
		if err != nil {
			return err
		}
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newLogParam())
	})
}

// start restore-experiment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var restoreExperimentOverrides []func(
	*cobra.Command,
	*ml.RestoreExperiment,
)

func newRestoreExperiment() *cobra.Command {
	cmd := &cobra.Command{}

	var restoreExperimentReq ml.RestoreExperiment
	var restoreExperimentJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&restoreExperimentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "restore-experiment EXPERIMENT_ID"
	cmd.Short = `Restores an experiment.`
	cmd.Long = `Restores an experiment.
  
  Restore an experiment marked for deletion. This also restores associated
  metadata, runs, metrics, params, and tags. If experiment uses FileStore,
  underlying artifacts associated with experiment are also restored.
  
  Throws RESOURCE_DOES_NOT_EXIST if experiment was never created or was
  permanently deleted.

  Arguments:
    EXPERIMENT_ID: ID of the associated experiment.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := cobra.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'experiment_id' in your JSON input")
			}
			return nil
		}
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = restoreExperimentJson.Unmarshal(&restoreExperimentReq)
			if err != nil {
				return err
			}
		}
		if !cmd.Flags().Changed("json") {
			restoreExperimentReq.ExperimentId = args[0]
		}

		err = w.Experiments.RestoreExperiment(ctx, restoreExperimentReq)
		if err != nil {
			return err
		}
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newRestoreExperiment())
	})
}

// start restore-run command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var restoreRunOverrides []func(
	*cobra.Command,
	*ml.RestoreRun,
)

func newRestoreRun() *cobra.Command {
	cmd := &cobra.Command{}

	var restoreRunReq ml.RestoreRun
	var restoreRunJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&restoreRunJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "restore-run RUN_ID"
	cmd.Short = `Restore a run.`
	cmd.Long = `Restore a run.
  
  Restores a deleted run.

  Arguments:
    RUN_ID: ID of the run to restore.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := cobra.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'run_id' in your JSON input")
			}
			return nil
		}
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = restoreRunJson.Unmarshal(&restoreRunReq)
			if err != nil {
				return err
			}
		}
		if !cmd.Flags().Changed("json") {
			restoreRunReq.RunId = args[0]
		}

		err = w.Experiments.RestoreRun(ctx, restoreRunReq)
		if err != nil {
			return err
		}
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newRestoreRun())
	})
}

// start restore-runs command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var restoreRunsOverrides []func(
	*cobra.Command,
	*ml.RestoreRuns,
)

func newRestoreRuns() *cobra.Command {
	cmd := &cobra.Command{}

	var restoreRunsReq ml.RestoreRuns
	var restoreRunsJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&restoreRunsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().IntVar(&restoreRunsReq.MaxRuns, "max-runs", restoreRunsReq.MaxRuns, `An optional positive integer indicating the maximum number of runs to restore.`)

	cmd.Use = "restore-runs EXPERIMENT_ID MIN_TIMESTAMP_MILLIS"
	cmd.Short = `Restore runs by deletion time.`
	cmd.Long = `Restore runs by deletion time.
  
  Bulk restore runs in an experiment that were deleted no earlier than the
  specified timestamp. Restores at most max_runs per request.

  Arguments:
    EXPERIMENT_ID: The ID of the experiment containing the runs to restore.
    MIN_TIMESTAMP_MILLIS: The minimum deletion timestamp in milliseconds since the UNIX epoch for
      restoring runs. Only runs deleted no earlier than this timestamp are
      restored.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := cobra.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'experiment_id', 'min_timestamp_millis' in your JSON input")
			}
			return nil
		}
		check := cobra.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = restoreRunsJson.Unmarshal(&restoreRunsReq)
			if err != nil {
				return err
			}
		}
		if !cmd.Flags().Changed("json") {
			restoreRunsReq.ExperimentId = args[0]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &restoreRunsReq.MinTimestampMillis)
			if err != nil {
				return fmt.Errorf("invalid MIN_TIMESTAMP_MILLIS: %s", args[1])
			}
		}

		response, err := w.Experiments.RestoreRuns(ctx, restoreRunsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newRestoreRuns())
	})
}

// start search-experiments command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var searchExperimentsOverrides []func(
	*cobra.Command,
	*ml.SearchExperiments,
)

func newSearchExperiments() *cobra.Command {
	cmd := &cobra.Command{}

	var searchExperimentsReq ml.SearchExperiments
	var searchExperimentsJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&searchExperimentsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&searchExperimentsReq.Filter, "filter", searchExperimentsReq.Filter, `String representing a SQL filter condition (e.g.`)
	cmd.Flags().Int64Var(&searchExperimentsReq.MaxResults, "max-results", searchExperimentsReq.MaxResults, `Maximum number of experiments desired.`)
	// TODO: array: order_by
	cmd.Flags().StringVar(&searchExperimentsReq.PageToken, "page-token", searchExperimentsReq.PageToken, `Token indicating the page of experiments to fetch.`)
	cmd.Flags().Var(&searchExperimentsReq.ViewType, "view-type", `Qualifier for type of experiments to be returned. Supported values: [ACTIVE_ONLY, ALL, DELETED_ONLY]`)

	cmd.Use = "search-experiments"
	cmd.Short = `Search experiments.`
	cmd.Long = `Search experiments.
  
  Searches for experiments that satisfy specified search criteria.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = searchExperimentsJson.Unmarshal(&searchExperimentsReq)
			if err != nil {
				return err
			}
		}

		response := w.Experiments.SearchExperiments(ctx, searchExperimentsReq)
		return cmdio.Render(ctx, response)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newSearchExperiments())
	})
}

// start search-runs command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var searchRunsOverrides []func(
	*cobra.Command,
	*ml.SearchRuns,
)

func newSearchRuns() *cobra.Command {
	cmd := &cobra.Command{}

	var searchRunsReq ml.SearchRuns
	var searchRunsJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&searchRunsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: experiment_ids
	cmd.Flags().StringVar(&searchRunsReq.Filter, "filter", searchRunsReq.Filter, `A filter expression over params, metrics, and tags, that allows returning a subset of runs.`)
	cmd.Flags().IntVar(&searchRunsReq.MaxResults, "max-results", searchRunsReq.MaxResults, `Maximum number of runs desired.`)
	// TODO: array: order_by
	cmd.Flags().StringVar(&searchRunsReq.PageToken, "page-token", searchRunsReq.PageToken, `Token for the current page of runs.`)
	cmd.Flags().Var(&searchRunsReq.RunViewType, "run-view-type", `Whether to display only active, only deleted, or all runs. Supported values: [ACTIVE_ONLY, ALL, DELETED_ONLY]`)

	cmd.Use = "search-runs"
	cmd.Short = `Search for runs.`
	cmd.Long = `Search for runs.
  
  Searches for runs that satisfy expressions.
  
  Search expressions can use mlflowMetric and mlflowParam keys.",`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = searchRunsJson.Unmarshal(&searchRunsReq)
			if err != nil {
				return err
			}
		}

		response := w.Experiments.SearchRuns(ctx, searchRunsReq)
		return cmdio.Render(ctx, response)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newSearchRuns())
	})
}

// start set-experiment-tag command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var setExperimentTagOverrides []func(
	*cobra.Command,
	*ml.SetExperimentTag,
)

func newSetExperimentTag() *cobra.Command {
	cmd := &cobra.Command{}

	var setExperimentTagReq ml.SetExperimentTag
	var setExperimentTagJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&setExperimentTagJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "set-experiment-tag EXPERIMENT_ID KEY VALUE"
	cmd.Short = `Set a tag.`
	cmd.Long = `Set a tag.
  
  Sets a tag on an experiment. Experiment tags are metadata that can be updated.

  Arguments:
    EXPERIMENT_ID: ID of the experiment under which to log the tag. Must be provided.
    KEY: Name of the tag. Maximum size depends on storage backend. All storage
      backends are guaranteed to support key values up to 250 bytes in size.
    VALUE: String value of the tag being logged. Maximum size depends on storage
      backend. All storage backends are guaranteed to support key values up to
      5000 bytes in size.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := cobra.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'experiment_id', 'key', 'value' in your JSON input")
			}
			return nil
		}
		check := cobra.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = setExperimentTagJson.Unmarshal(&setExperimentTagReq)
			if err != nil {
				return err
			}
		}
		if !cmd.Flags().Changed("json") {
			setExperimentTagReq.ExperimentId = args[0]
		}
		if !cmd.Flags().Changed("json") {
			setExperimentTagReq.Key = args[1]
		}
		if !cmd.Flags().Changed("json") {
			setExperimentTagReq.Value = args[2]
		}

		err = w.Experiments.SetExperimentTag(ctx, setExperimentTagReq)
		if err != nil {
			return err
		}
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newSetExperimentTag())
	})
}

// start set-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var setPermissionsOverrides []func(
	*cobra.Command,
	*ml.ExperimentPermissionsRequest,
)

func newSetPermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var setPermissionsReq ml.ExperimentPermissionsRequest
	var setPermissionsJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&setPermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "set-permissions EXPERIMENT_ID"
	cmd.Short = `Set experiment permissions.`
	cmd.Long = `Set experiment permissions.
  
  Sets permissions on an experiment. Experiments can inherit permissions from
  their root object.

  Arguments:
    EXPERIMENT_ID: The experiment for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = setPermissionsJson.Unmarshal(&setPermissionsReq)
			if err != nil {
				return err
			}
		}
		setPermissionsReq.ExperimentId = args[0]

		response, err := w.Experiments.SetPermissions(ctx, setPermissionsReq)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newSetPermissions())
	})
}

// start set-tag command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var setTagOverrides []func(
	*cobra.Command,
	*ml.SetTag,
)

func newSetTag() *cobra.Command {
	cmd := &cobra.Command{}

	var setTagReq ml.SetTag
	var setTagJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&setTagJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&setTagReq.RunId, "run-id", setTagReq.RunId, `ID of the run under which to log the tag.`)
	cmd.Flags().StringVar(&setTagReq.RunUuid, "run-uuid", setTagReq.RunUuid, `[Deprecated, use run_id instead] ID of the run under which to log the tag.`)

	cmd.Use = "set-tag KEY VALUE"
	cmd.Short = `Set a tag.`
	cmd.Long = `Set a tag.
  
  Sets a tag on a run. Tags are run metadata that can be updated during a run
  and after a run completes.

  Arguments:
    KEY: Name of the tag. Maximum size depends on storage backend. All storage
      backends are guaranteed to support key values up to 250 bytes in size.
    VALUE: String value of the tag being logged. Maximum size depends on storage
      backend. All storage backends are guaranteed to support key values up to
      5000 bytes in size.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := cobra.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'key', 'value' in your JSON input")
			}
			return nil
		}
		check := cobra.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = setTagJson.Unmarshal(&setTagReq)
			if err != nil {
				return err
			}
		}
		if !cmd.Flags().Changed("json") {
			setTagReq.Key = args[0]
		}
		if !cmd.Flags().Changed("json") {
			setTagReq.Value = args[1]
		}

		err = w.Experiments.SetTag(ctx, setTagReq)
		if err != nil {
			return err
		}
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newSetTag())
	})
}

// start update-experiment command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateExperimentOverrides []func(
	*cobra.Command,
	*ml.UpdateExperiment,
)

func newUpdateExperiment() *cobra.Command {
	cmd := &cobra.Command{}

	var updateExperimentReq ml.UpdateExperiment
	var updateExperimentJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&updateExperimentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateExperimentReq.NewName, "new-name", updateExperimentReq.NewName, `If provided, the experiment's name is changed to the new name.`)

	cmd.Use = "update-experiment EXPERIMENT_ID"
	cmd.Short = `Update an experiment.`
	cmd.Long = `Update an experiment.
  
  Updates experiment metadata.

  Arguments:
    EXPERIMENT_ID: ID of the associated experiment.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := cobra.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'experiment_id' in your JSON input")
			}
			return nil
		}
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = updateExperimentJson.Unmarshal(&updateExperimentReq)
			if err != nil {
				return err
			}
		}
		if !cmd.Flags().Changed("json") {
			updateExperimentReq.ExperimentId = args[0]
		}

		err = w.Experiments.UpdateExperiment(ctx, updateExperimentReq)
		if err != nil {
			return err
		}
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newUpdateExperiment())
	})
}

// start update-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updatePermissionsOverrides []func(
	*cobra.Command,
	*ml.ExperimentPermissionsRequest,
)

func newUpdatePermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var updatePermissionsReq ml.ExperimentPermissionsRequest
	var updatePermissionsJson flags.JsonFlag

	// TODO: short flags
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

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = updatePermissionsJson.Unmarshal(&updatePermissionsReq)
			if err != nil {
				return err
			}
		}
		updatePermissionsReq.ExperimentId = args[0]

		response, err := w.Experiments.UpdatePermissions(ctx, updatePermissionsReq)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newUpdatePermissions())
	})
}

// start update-run command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateRunOverrides []func(
	*cobra.Command,
	*ml.UpdateRun,
)

func newUpdateRun() *cobra.Command {
	cmd := &cobra.Command{}

	var updateRunReq ml.UpdateRun
	var updateRunJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&updateRunJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().Int64Var(&updateRunReq.EndTime, "end-time", updateRunReq.EndTime, `Unix timestamp in milliseconds of when the run ended.`)
	cmd.Flags().StringVar(&updateRunReq.RunId, "run-id", updateRunReq.RunId, `ID of the run to update.`)
	cmd.Flags().StringVar(&updateRunReq.RunUuid, "run-uuid", updateRunReq.RunUuid, `[Deprecated, use run_id instead] ID of the run to update.`)
	cmd.Flags().Var(&updateRunReq.Status, "status", `Updated status of the run. Supported values: [FAILED, FINISHED, KILLED, RUNNING, SCHEDULED]`)

	cmd.Use = "update-run"
	cmd.Short = `Update a run.`
	cmd.Long = `Update a run.
  
  Updates run metadata.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = updateRunJson.Unmarshal(&updateRunReq)
			if err != nil {
				return err
			}
		}

		response, err := w.Experiments.UpdateRun(ctx, updateRunReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newUpdateRun())
	})
}

// end service Experiments
