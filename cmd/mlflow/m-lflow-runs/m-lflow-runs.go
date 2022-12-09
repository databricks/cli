package m_lflow_runs

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/mlflow"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "m-lflow-runs",
}

var createReq mlflow.CreateRun

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().StringVar(&createReq.ExperimentId, "experiment-id", createReq.ExperimentId, `ID of the associated experiment.`)
	createCmd.Flags().Int64Var(&createReq.StartTime, "start-time", createReq.StartTime, `Unix timestamp in milliseconds of when the run started.`)
	// TODO: array: tags
	createCmd.Flags().StringVar(&createReq.UserId, "user-id", createReq.UserId, `ID of the user executing the run.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a run.`,
	Long: `Create a run.
  
  Creates a new run within an experiment. A run is usually a single execution of
  a machine learning or data ETL pipeline. MLflow uses runs to track the
  mlflowParam, mlflowMetric and mlflowRunTag associated with a single
  execution.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.MLflowRuns.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var deleteReq mlflow.DeleteRun

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.RunId, "run-id", deleteReq.RunId, `ID of the run to delete.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a run.`,
	Long: `Delete a run.
  
  Marks a run for deletion.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.MLflowRuns.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

var deleteTagReq mlflow.DeleteTag

func init() {
	Cmd.AddCommand(deleteTagCmd)
	// TODO: short flags

	deleteTagCmd.Flags().StringVar(&deleteTagReq.Key, "key", deleteTagReq.Key, `Name of the tag.`)
	deleteTagCmd.Flags().StringVar(&deleteTagReq.RunId, "run-id", deleteTagReq.RunId, `ID of the run that the tag was logged under.`)

}

var deleteTagCmd = &cobra.Command{
	Use:   "delete-tag",
	Short: `Delete a tag.`,
	Long: `Delete a tag.
  
  Deletes a tag on a run. Tags are run metadata that can be updated during a run
  and after a run completes.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.MLflowRuns.DeleteTag(ctx, deleteTagReq)
		if err != nil {
			return err
		}
		return nil
	},
}

var getReq mlflow.GetRunRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.RunId, "run-id", getReq.RunId, `ID of the run to fetch.`)
	getCmd.Flags().StringVar(&getReq.RunUuid, "run-uuid", getReq.RunUuid, `[Deprecated, use run_id instead] ID of the run to fetch.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get a run.`,
	Long: `Get a run.
  
  "Gets the metadata, metrics, params, and tags for a run. In the case where
  multiple metrics with the same key are logged for a run, return only the value
  with the latest timestamp.
  
  If there are multiple values with the latest timestamp, return the maximum of
  these values.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.MLflowRuns.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var logBatchReq mlflow.LogBatch

func init() {
	Cmd.AddCommand(logBatchCmd)
	// TODO: short flags

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
  
  * No more than 1000 metrics, params, and tags in total * Up to 1000 metrics\n-
  Up to 100 params * Up to 100 tags
  
  For example, a valid request might contain 900 metrics, 50 params, and 50
  tags, but logging 900 metrics, 50 params, and 51 tags is invalid.
  
  The following limits also apply to metric, param, and tag keys and values:
  
  * Metric keyes, param keys, and tag keys can be up to 250 characters in length
  * Parameter and tag values can be up to 250 characters in length`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.MLflowRuns.LogBatch(ctx, logBatchReq)
		if err != nil {
			return err
		}
		return nil
	},
}

var logMetricReq mlflow.LogMetric

func init() {
	Cmd.AddCommand(logMetricCmd)
	// TODO: short flags

	logMetricCmd.Flags().StringVar(&logMetricReq.Key, "key", logMetricReq.Key, `Name of the metric.`)
	logMetricCmd.Flags().StringVar(&logMetricReq.RunId, "run-id", logMetricReq.RunId, `ID of the run under which to log the metric.`)
	logMetricCmd.Flags().StringVar(&logMetricReq.RunUuid, "run-uuid", logMetricReq.RunUuid, `[Deprecated, use run_id instead] ID of the run under which to log the metric.`)
	logMetricCmd.Flags().Int64Var(&logMetricReq.Step, "step", logMetricReq.Step, `Step at which to log the metric.`)
	logMetricCmd.Flags().Int64Var(&logMetricReq.Timestamp, "timestamp", logMetricReq.Timestamp, `Unix timestamp in milliseconds at the time metric was logged.`)
	logMetricCmd.Flags().Float64Var(&logMetricReq.Value, "value", logMetricReq.Value, `Double value of the metric being logged.`)

}

var logMetricCmd = &cobra.Command{
	Use:   "log-metric",
	Short: `Log a metric.`,
	Long: `Log a metric.
  
  Logs a metric for a run. A metric is a key-value pair (string key, float
  value) with an associated timestamp. Examples include the various metrics that
  represent ML model accuracy. A metric can be logged multiple times.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.MLflowRuns.LogMetric(ctx, logMetricReq)
		if err != nil {
			return err
		}
		return nil
	},
}

var logModelReq mlflow.LogModel

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

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.MLflowRuns.LogModel(ctx, logModelReq)
		if err != nil {
			return err
		}
		return nil
	},
}

var logParameterReq mlflow.LogParam

func init() {
	Cmd.AddCommand(logParameterCmd)
	// TODO: short flags

	logParameterCmd.Flags().StringVar(&logParameterReq.Key, "key", logParameterReq.Key, `Name of the param.`)
	logParameterCmd.Flags().StringVar(&logParameterReq.RunId, "run-id", logParameterReq.RunId, `ID of the run under which to log the param.`)
	logParameterCmd.Flags().StringVar(&logParameterReq.RunUuid, "run-uuid", logParameterReq.RunUuid, `[Deprecated, use run_id instead] ID of the run under which to log the param.`)
	logParameterCmd.Flags().StringVar(&logParameterReq.Value, "value", logParameterReq.Value, `String value of the param being logged.`)

}

var logParameterCmd = &cobra.Command{
	Use:   "log-parameter",
	Short: `Log a param.`,
	Long: `Log a param.
  
  Logs a param used for a run. A param is a key-value pair (string key, string
  value). Examples include hyperparameters used for ML model training and
  constant dates and values used in an ETL pipeline. A param can be logged only
  once for a run.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.MLflowRuns.LogParameter(ctx, logParameterReq)
		if err != nil {
			return err
		}
		return nil
	},
}

var restoreReq mlflow.RestoreRun

func init() {
	Cmd.AddCommand(restoreCmd)
	// TODO: short flags

	restoreCmd.Flags().StringVar(&restoreReq.RunId, "run-id", restoreReq.RunId, `ID of the run to restore.`)

}

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: `Restore a run.`,
	Long: `Restore a run.
  
  Restores a deleted run.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.MLflowRuns.Restore(ctx, restoreReq)
		if err != nil {
			return err
		}
		return nil
	},
}

var searchReq mlflow.SearchRuns

func init() {
	Cmd.AddCommand(searchCmd)
	// TODO: short flags

	// TODO: array: experiment_ids
	searchCmd.Flags().StringVar(&searchReq.Filter, "filter", searchReq.Filter, `A filter expression over params, metrics, and tags, that allows returning a subset of runs.`)
	searchCmd.Flags().IntVar(&searchReq.MaxResults, "max-results", searchReq.MaxResults, `Maximum number of runs desired.`)
	// TODO: array: order_by
	searchCmd.Flags().StringVar(&searchReq.PageToken, "page-token", searchReq.PageToken, `Token for the current page of runs.`)
	searchCmd.Flags().Var(&searchReq.RunViewType, "run-view-type", `Whether to display only active, only deleted, or all runs.`)

}

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: `Search for runs.`,
	Long: `Search for runs.
  
  Searches for runs that satisfy expressions.
  
  Search expressions can use mlflowMetric and mlflowParam keys.",`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.MLflowRuns.SearchAll(ctx, searchReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

var setTagReq mlflow.SetTag

func init() {
	Cmd.AddCommand(setTagCmd)
	// TODO: short flags

	setTagCmd.Flags().StringVar(&setTagReq.Key, "key", setTagReq.Key, `Name of the tag.`)
	setTagCmd.Flags().StringVar(&setTagReq.RunId, "run-id", setTagReq.RunId, `ID of the run under which to log the tag.`)
	setTagCmd.Flags().StringVar(&setTagReq.RunUuid, "run-uuid", setTagReq.RunUuid, `[Deprecated, use run_id instead] ID of the run under which to log the tag.`)
	setTagCmd.Flags().StringVar(&setTagReq.Value, "value", setTagReq.Value, `String value of the tag being logged.`)

}

var setTagCmd = &cobra.Command{
	Use:   "set-tag",
	Short: `Set a tag.`,
	Long: `Set a tag.
  
  Sets a tag on a run. Tags are run metadata that can be updated during a run
  and after a run completes.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.MLflowRuns.SetTag(ctx, setTagReq)
		if err != nil {
			return err
		}
		return nil
	},
}

var updateReq mlflow.UpdateRun

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().Int64Var(&updateReq.EndTime, "end-time", updateReq.EndTime, `Unix timestamp in milliseconds of when the run ended.`)
	updateCmd.Flags().StringVar(&updateReq.RunId, "run-id", updateReq.RunId, `ID of the run to update.`)
	updateCmd.Flags().StringVar(&updateReq.RunUuid, "run-uuid", updateReq.RunUuid, `[Deprecated, use run_id instead] ID of the run to update.`)
	updateCmd.Flags().Var(&updateReq.Status, "status", `Updated status of the run.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update a run.`,
	Long: `Update a run.
  
  Updates run metadata.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.MLflowRuns.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// end service MLflowRuns
