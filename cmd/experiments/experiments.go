// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package experiments

import (
	"fmt"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "experiments",
}

// start create-experiment command

var createExperimentReq ml.CreateExperiment
var createExperimentJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(createExperimentCmd)
	// TODO: short flags
	createExperimentCmd.Flags().Var(&createExperimentJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createExperimentCmd.Flags().StringVar(&createExperimentReq.ArtifactLocation, "artifact-location", createExperimentReq.ArtifactLocation, `Location where all artifacts for the experiment are stored.`)
	// TODO: array: tags

}

var createExperimentCmd = &cobra.Command{
	Use:   "create-experiment",
	Short: `Create experiment.`,
	Long: `Create experiment.
  
  Creates an experiment with a name. Returns the ID of the newly created
  experiment. Validates that another experiment with the same name does not
  already exist and fails if another experiment with the same name already
  exists.
  
  Throws RESOURCE_ALREADY_EXISTS if a experiment with the given name exists.`,

	Annotations: map[string]string{},
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = createExperimentJson.Unmarshall(&createExperimentReq)
		if err != nil {
			return err
		}
		createExperimentReq.Name = args[0]

		response, err := w.Experiments.CreateExperiment(ctx, createExperimentReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start create-run command

var createRunReq ml.CreateRun
var createRunJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(createRunCmd)
	// TODO: short flags
	createRunCmd.Flags().Var(&createRunJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createRunCmd.Flags().StringVar(&createRunReq.ExperimentId, "experiment-id", createRunReq.ExperimentId, `ID of the associated experiment.`)
	createRunCmd.Flags().Int64Var(&createRunReq.StartTime, "start-time", createRunReq.StartTime, `Unix timestamp in milliseconds of when the run started.`)
	// TODO: array: tags
	createRunCmd.Flags().StringVar(&createRunReq.UserId, "user-id", createRunReq.UserId, `ID of the user executing the run.`)

}

var createRunCmd = &cobra.Command{
	Use:   "create-run",
	Short: `Create a run.`,
	Long: `Create a run.
  
  Creates a new run within an experiment. A run is usually a single execution of
  a machine learning or data ETL pipeline. MLflow uses runs to track the
  mlflowParam, mlflowMetric and mlflowRunTag associated with a single
  execution.`,

	Annotations: map[string]string{},
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = createRunJson.Unmarshall(&createRunReq)
		if err != nil {
			return err
		}

		response, err := w.Experiments.CreateRun(ctx, createRunReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete-experiment command

var deleteExperimentReq ml.DeleteExperiment

func init() {
	Cmd.AddCommand(deleteExperimentCmd)
	// TODO: short flags

}

var deleteExperimentCmd = &cobra.Command{
	Use:   "delete-experiment EXPERIMENT_ID",
	Short: `Delete an experiment.`,
	Long: `Delete an experiment.
  
  Marks an experiment and associated metadata, runs, metrics, params, and tags
  for deletion. If the experiment uses FileStore, artifacts associated with
  experiment are also deleted.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		deleteExperimentReq.ExperimentId = args[0]

		err = w.Experiments.DeleteExperiment(ctx, deleteExperimentReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start delete-run command

var deleteRunReq ml.DeleteRun

func init() {
	Cmd.AddCommand(deleteRunCmd)
	// TODO: short flags

}

var deleteRunCmd = &cobra.Command{
	Use:   "delete-run RUN_ID",
	Short: `Delete a run.`,
	Long: `Delete a run.
  
  Marks a run for deletion.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		deleteRunReq.RunId = args[0]

		err = w.Experiments.DeleteRun(ctx, deleteRunReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start delete-tag command

var deleteTagReq ml.DeleteTag

func init() {
	Cmd.AddCommand(deleteTagCmd)
	// TODO: short flags

}

var deleteTagCmd = &cobra.Command{
	Use:   "delete-tag RUN_ID KEY",
	Short: `Delete a tag.`,
	Long: `Delete a tag.
  
  Deletes a tag on a run. Tags are run metadata that can be updated during a run
  and after a run completes.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		deleteTagReq.RunId = args[0]
		deleteTagReq.Key = args[1]

		err = w.Experiments.DeleteTag(ctx, deleteTagReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get-by-name command

var getByNameReq ml.GetByNameRequest

func init() {
	Cmd.AddCommand(getByNameCmd)
	// TODO: short flags

}

var getByNameCmd = &cobra.Command{
	Use:   "get-by-name EXPERIMENT_NAME",
	Short: `Get metadata.`,
	Long: `Get metadata.
  
  "Gets metadata for an experiment.
  
  This endpoint will return deleted experiments, but prefers the active
  experiment if an active and deleted experiment share the same name. If
  multiple deleted experiments share the same name, the API will return one of
  them.
  
  Throws RESOURCE_DOES_NOT_EXIST if no experiment with the specified name
  exists.S`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		getByNameReq.ExperimentName = args[0]

		response, err := w.Experiments.GetByName(ctx, getByNameReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start get-experiment command

var getExperimentReq ml.GetExperimentRequest

func init() {
	Cmd.AddCommand(getExperimentCmd)
	// TODO: short flags

}

var getExperimentCmd = &cobra.Command{
	Use:   "get-experiment EXPERIMENT_ID",
	Short: `Get an experiment.`,
	Long: `Get an experiment.
  
  Gets metadata for an experiment. This method works on deleted experiments.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		getExperimentReq.ExperimentId = args[0]

		response, err := w.Experiments.GetExperiment(ctx, getExperimentReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start get-history command

var getHistoryReq ml.GetHistoryRequest

func init() {
	Cmd.AddCommand(getHistoryCmd)
	// TODO: short flags

	getHistoryCmd.Flags().IntVar(&getHistoryReq.MaxResults, "max-results", getHistoryReq.MaxResults, `Maximum number of Metric records to return per paginated request.`)
	getHistoryCmd.Flags().StringVar(&getHistoryReq.PageToken, "page-token", getHistoryReq.PageToken, `Token indicating the page of metric histories to fetch.`)
	getHistoryCmd.Flags().StringVar(&getHistoryReq.RunId, "run-id", getHistoryReq.RunId, `ID of the run from which to fetch metric values.`)
	getHistoryCmd.Flags().StringVar(&getHistoryReq.RunUuid, "run-uuid", getHistoryReq.RunUuid, `[Deprecated, use run_id instead] ID of the run from which to fetch metric values.`)

}

var getHistoryCmd = &cobra.Command{
	Use:   "get-history METRIC_KEY",
	Short: `Get history of a given metric within a run.`,
	Long: `Get history of a given metric within a run.
  
  Gets a list of all values for the specified metric for a given run.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		getHistoryReq.MetricKey = args[0]

		response, err := w.Experiments.GetHistory(ctx, getHistoryReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start get-run command

var getRunReq ml.GetRunRequest

func init() {
	Cmd.AddCommand(getRunCmd)
	// TODO: short flags

	getRunCmd.Flags().StringVar(&getRunReq.RunUuid, "run-uuid", getRunReq.RunUuid, `[Deprecated, use run_id instead] ID of the run to fetch.`)

}

var getRunCmd = &cobra.Command{
	Use:   "get-run RUN_ID",
	Short: `Get a run.`,
	Long: `Get a run.
  
  "Gets the metadata, metrics, params, and tags for a run. In the case where
  multiple metrics with the same key are logged for a run, return only the value
  with the latest timestamp.
  
  If there are multiple values with the latest timestamp, return the maximum of
  these values.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		getRunReq.RunId = args[0]

		response, err := w.Experiments.GetRun(ctx, getRunReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list-artifacts command

var listArtifactsReq ml.ListArtifactsRequest

func init() {
	Cmd.AddCommand(listArtifactsCmd)
	// TODO: short flags

	listArtifactsCmd.Flags().StringVar(&listArtifactsReq.PageToken, "page-token", listArtifactsReq.PageToken, `Token indicating the page of artifact results to fetch.`)
	listArtifactsCmd.Flags().StringVar(&listArtifactsReq.Path, "path", listArtifactsReq.Path, `Filter artifacts matching this path (a relative path from the root artifact directory).`)
	listArtifactsCmd.Flags().StringVar(&listArtifactsReq.RunId, "run-id", listArtifactsReq.RunId, `ID of the run whose artifacts to list.`)
	listArtifactsCmd.Flags().StringVar(&listArtifactsReq.RunUuid, "run-uuid", listArtifactsReq.RunUuid, `[Deprecated, use run_id instead] ID of the run whose artifacts to list.`)

}

var listArtifactsCmd = &cobra.Command{
	Use:   "list-artifacts",
	Short: `Get all artifacts.`,
	Long: `Get all artifacts.
  
  List artifacts for a run. Takes an optional artifact_path prefix. If it is
  specified, the response contains only artifacts with the specified prefix.",`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(0),
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)

		response, err := w.Experiments.ListArtifactsAll(ctx, listArtifactsReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list-experiments command

var listExperimentsReq ml.ListExperimentsRequest

func init() {
	Cmd.AddCommand(listExperimentsCmd)
	// TODO: short flags

	listExperimentsCmd.Flags().IntVar(&listExperimentsReq.MaxResults, "max-results", listExperimentsReq.MaxResults, `Maximum number of experiments desired.`)
	listExperimentsCmd.Flags().StringVar(&listExperimentsReq.PageToken, "page-token", listExperimentsReq.PageToken, `Token indicating the page of experiments to fetch.`)
	listExperimentsCmd.Flags().StringVar(&listExperimentsReq.ViewType, "view-type", listExperimentsReq.ViewType, `Qualifier for type of experiments to be returned.`)

}

var listExperimentsCmd = &cobra.Command{
	Use:   "list-experiments",
	Short: `List experiments.`,
	Long: `List experiments.
  
  Gets a list of all experiments.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(0),
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)

		response, err := w.Experiments.ListExperimentsAll(ctx, listExperimentsReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start log-batch command

var logBatchReq ml.LogBatch
var logBatchJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(logBatchCmd)
	// TODO: short flags
	logBatchCmd.Flags().Var(&logBatchJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: metrics
	// TODO: array: params
	logBatchCmd.Flags().StringVar(&logBatchReq.RunId, "run-id", logBatchReq.RunId, `ID of the run to log under.`)
	// TODO: array: tags

}

var logBatchCmd = &cobra.Command{
	Use:   "log-batch",
	Short: `Log a batch.`,
	Long: `Log a batch.
  
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
  
  * No more than 1000 metrics, params, and tags in total * Up to 1000 metrics -
  Up to 100 params * Up to 100 tags
  
  For example, a valid request might contain 900 metrics, 50 params, and 50
  tags, but logging 900 metrics, 50 params, and 51 tags is invalid.
  
  The following limits also apply to metric, param, and tag keys and values:
  
  * Metric keyes, param keys, and tag keys can be up to 250 characters in length
  * Parameter and tag values can be up to 250 characters in length`,

	Annotations: map[string]string{},
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = logBatchJson.Unmarshall(&logBatchReq)
		if err != nil {
			return err
		}

		err = w.Experiments.LogBatch(ctx, logBatchReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start log-metric command

var logMetricReq ml.LogMetric

func init() {
	Cmd.AddCommand(logMetricCmd)
	// TODO: short flags

	logMetricCmd.Flags().StringVar(&logMetricReq.RunId, "run-id", logMetricReq.RunId, `ID of the run under which to log the metric.`)
	logMetricCmd.Flags().StringVar(&logMetricReq.RunUuid, "run-uuid", logMetricReq.RunUuid, `[Deprecated, use run_id instead] ID of the run under which to log the metric.`)
	logMetricCmd.Flags().Int64Var(&logMetricReq.Step, "step", logMetricReq.Step, `Step at which to log the metric.`)

}

var logMetricCmd = &cobra.Command{
	Use:   "log-metric KEY VALUE TIMESTAMP",
	Short: `Log a metric.`,
	Long: `Log a metric.
  
  Logs a metric for a run. A metric is a key-value pair (string key, float
  value) with an associated timestamp. Examples include the various metrics that
  represent ML model accuracy. A metric can be logged multiple times.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(3),
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		logMetricReq.Key = args[0]
		_, err = fmt.Sscan(args[1], &logMetricReq.Value)
		if err != nil {
			return fmt.Errorf("invalid VALUE: %s", args[1])
		}
		_, err = fmt.Sscan(args[2], &logMetricReq.Timestamp)
		if err != nil {
			return fmt.Errorf("invalid TIMESTAMP: %s", args[2])
		}

		err = w.Experiments.LogMetric(ctx, logMetricReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start log-model command

var logModelReq ml.LogModel

func init() {
	Cmd.AddCommand(logModelCmd)
	// TODO: short flags

	logModelCmd.Flags().StringVar(&logModelReq.ModelJson, "model-json", logModelReq.ModelJson, `MLmodel file in json format.`)
	logModelCmd.Flags().StringVar(&logModelReq.RunId, "run-id", logModelReq.RunId, `ID of the run to log under.`)

}

var logModelCmd = &cobra.Command{
	Use:   "log-model",
	Short: `Log a model.`,
	Long: `Log a model.
  
  **NOTE:** Experimental: This API may change or be removed in a future release
  without warning.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(0),
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)

		err = w.Experiments.LogModel(ctx, logModelReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start log-param command

var logParamReq ml.LogParam

func init() {
	Cmd.AddCommand(logParamCmd)
	// TODO: short flags

	logParamCmd.Flags().StringVar(&logParamReq.RunId, "run-id", logParamReq.RunId, `ID of the run under which to log the param.`)
	logParamCmd.Flags().StringVar(&logParamReq.RunUuid, "run-uuid", logParamReq.RunUuid, `[Deprecated, use run_id instead] ID of the run under which to log the param.`)

}

var logParamCmd = &cobra.Command{
	Use:   "log-param KEY VALUE",
	Short: `Log a param.`,
	Long: `Log a param.
  
  Logs a param used for a run. A param is a key-value pair (string key, string
  value). Examples include hyperparameters used for ML model training and
  constant dates and values used in an ETL pipeline. A param can be logged only
  once for a run.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		logParamReq.Key = args[0]
		logParamReq.Value = args[1]

		err = w.Experiments.LogParam(ctx, logParamReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start restore-experiment command

var restoreExperimentReq ml.RestoreExperiment

func init() {
	Cmd.AddCommand(restoreExperimentCmd)
	// TODO: short flags

}

var restoreExperimentCmd = &cobra.Command{
	Use:   "restore-experiment EXPERIMENT_ID",
	Short: `Restores an experiment.`,
	Long: `Restores an experiment.
  
  "Restore an experiment marked for deletion. This also restores associated
  metadata, runs, metrics, params, and tags. If experiment uses FileStore,
  underlying artifacts associated with experiment are also restored.
  
  Throws RESOURCE_DOES_NOT_EXIST if experiment was never created or was
  permanently deleted.",`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		restoreExperimentReq.ExperimentId = args[0]

		err = w.Experiments.RestoreExperiment(ctx, restoreExperimentReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start restore-run command

var restoreRunReq ml.RestoreRun

func init() {
	Cmd.AddCommand(restoreRunCmd)
	// TODO: short flags

}

var restoreRunCmd = &cobra.Command{
	Use:   "restore-run RUN_ID",
	Short: `Restore a run.`,
	Long: `Restore a run.
  
  Restores a deleted run.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		restoreRunReq.RunId = args[0]

		err = w.Experiments.RestoreRun(ctx, restoreRunReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start search-experiments command

var searchExperimentsReq ml.SearchExperiments
var searchExperimentsJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(searchExperimentsCmd)
	// TODO: short flags
	searchExperimentsCmd.Flags().Var(&searchExperimentsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	searchExperimentsCmd.Flags().StringVar(&searchExperimentsReq.Filter, "filter", searchExperimentsReq.Filter, `String representing a SQL filter condition (e.g.`)
	searchExperimentsCmd.Flags().Int64Var(&searchExperimentsReq.MaxResults, "max-results", searchExperimentsReq.MaxResults, `Maximum number of experiments desired.`)
	// TODO: array: order_by
	searchExperimentsCmd.Flags().StringVar(&searchExperimentsReq.PageToken, "page-token", searchExperimentsReq.PageToken, `Token indicating the page of experiments to fetch.`)
	searchExperimentsCmd.Flags().Var(&searchExperimentsReq.ViewType, "view-type", `Qualifier for type of experiments to be returned.`)

}

var searchExperimentsCmd = &cobra.Command{
	Use:   "search-experiments",
	Short: `Search experiments.`,
	Long: `Search experiments.
  
  Searches for experiments that satisfy specified search criteria.`,

	Annotations: map[string]string{},
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = searchExperimentsJson.Unmarshall(&searchExperimentsReq)
		if err != nil {
			return err
		}

		response, err := w.Experiments.SearchExperimentsAll(ctx, searchExperimentsReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start search-runs command

var searchRunsReq ml.SearchRuns
var searchRunsJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(searchRunsCmd)
	// TODO: short flags
	searchRunsCmd.Flags().Var(&searchRunsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: experiment_ids
	searchRunsCmd.Flags().StringVar(&searchRunsReq.Filter, "filter", searchRunsReq.Filter, `A filter expression over params, metrics, and tags, that allows returning a subset of runs.`)
	searchRunsCmd.Flags().IntVar(&searchRunsReq.MaxResults, "max-results", searchRunsReq.MaxResults, `Maximum number of runs desired.`)
	// TODO: array: order_by
	searchRunsCmd.Flags().StringVar(&searchRunsReq.PageToken, "page-token", searchRunsReq.PageToken, `Token for the current page of runs.`)
	searchRunsCmd.Flags().Var(&searchRunsReq.RunViewType, "run-view-type", `Whether to display only active, only deleted, or all runs.`)

}

var searchRunsCmd = &cobra.Command{
	Use:   "search-runs",
	Short: `Search for runs.`,
	Long: `Search for runs.
  
  Searches for runs that satisfy expressions.
  
  Search expressions can use mlflowMetric and mlflowParam keys.",`,

	Annotations: map[string]string{},
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = searchRunsJson.Unmarshall(&searchRunsReq)
		if err != nil {
			return err
		}

		response, err := w.Experiments.SearchRunsAll(ctx, searchRunsReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start set-experiment-tag command

var setExperimentTagReq ml.SetExperimentTag

func init() {
	Cmd.AddCommand(setExperimentTagCmd)
	// TODO: short flags

}

var setExperimentTagCmd = &cobra.Command{
	Use:   "set-experiment-tag EXPERIMENT_ID KEY VALUE",
	Short: `Set a tag.`,
	Long: `Set a tag.
  
  Sets a tag on an experiment. Experiment tags are metadata that can be updated.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(3),
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		setExperimentTagReq.ExperimentId = args[0]
		setExperimentTagReq.Key = args[1]
		setExperimentTagReq.Value = args[2]

		err = w.Experiments.SetExperimentTag(ctx, setExperimentTagReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start set-tag command

var setTagReq ml.SetTag

func init() {
	Cmd.AddCommand(setTagCmd)
	// TODO: short flags

	setTagCmd.Flags().StringVar(&setTagReq.RunId, "run-id", setTagReq.RunId, `ID of the run under which to log the tag.`)
	setTagCmd.Flags().StringVar(&setTagReq.RunUuid, "run-uuid", setTagReq.RunUuid, `[Deprecated, use run_id instead] ID of the run under which to log the tag.`)

}

var setTagCmd = &cobra.Command{
	Use:   "set-tag KEY VALUE",
	Short: `Set a tag.`,
	Long: `Set a tag.
  
  Sets a tag on a run. Tags are run metadata that can be updated during a run
  and after a run completes.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		setTagReq.Key = args[0]
		setTagReq.Value = args[1]

		err = w.Experiments.SetTag(ctx, setTagReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start update-experiment command

var updateExperimentReq ml.UpdateExperiment

func init() {
	Cmd.AddCommand(updateExperimentCmd)
	// TODO: short flags

	updateExperimentCmd.Flags().StringVar(&updateExperimentReq.NewName, "new-name", updateExperimentReq.NewName, `If provided, the experiment's name is changed to the new name.`)

}

var updateExperimentCmd = &cobra.Command{
	Use:   "update-experiment EXPERIMENT_ID",
	Short: `Update an experiment.`,
	Long: `Update an experiment.
  
  Updates experiment metadata.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		updateExperimentReq.ExperimentId = args[0]

		err = w.Experiments.UpdateExperiment(ctx, updateExperimentReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start update-run command

var updateRunReq ml.UpdateRun

func init() {
	Cmd.AddCommand(updateRunCmd)
	// TODO: short flags

	updateRunCmd.Flags().Int64Var(&updateRunReq.EndTime, "end-time", updateRunReq.EndTime, `Unix timestamp in milliseconds of when the run ended.`)
	updateRunCmd.Flags().StringVar(&updateRunReq.RunId, "run-id", updateRunReq.RunId, `ID of the run to update.`)
	updateRunCmd.Flags().StringVar(&updateRunReq.RunUuid, "run-uuid", updateRunReq.RunUuid, `[Deprecated, use run_id instead] ID of the run to update.`)
	updateRunCmd.Flags().Var(&updateRunReq.Status, "status", `Updated status of the run.`)

}

var updateRunCmd = &cobra.Command{
	Use:   "update-run",
	Short: `Update a run.`,
	Long: `Update a run.
  
  Updates run metadata.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(0),
	PreRunE:     root.TryWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)

		response, err := w.Experiments.UpdateRun(ctx, updateRunReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// end service Experiments
