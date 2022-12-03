package m_lflow_runs

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
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

	createCmd.Flags().StringVar(&createReq.ExperimentId, "experiment-id", "", `ID of the associated experiment.`)
	createCmd.Flags().Int64Var(&createReq.StartTime, "start-time", 0, `Unix timestamp in milliseconds of when the run started.`)
	// TODO: complex arg: tags
	createCmd.Flags().StringVar(&createReq.UserId, "user-id", "", `ID of the user executing the run.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a run Creates a new run within an experiment.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.MLflowRuns.Create(ctx, createReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var deleteReq mlflow.DeleteRun

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.RunId, "run-id", "", `ID of the run to delete.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a run Marks a run for deletion.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	deleteTagCmd.Flags().StringVar(&deleteTagReq.Key, "key", "", `Name of the tag.`)
	deleteTagCmd.Flags().StringVar(&deleteTagReq.RunId, "run-id", "", `ID of the run that the tag was logged under.`)

}

var deleteTagCmd = &cobra.Command{
	Use:   "delete-tag",
	Short: `Delete a tag Deletes a tag on a run.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	getCmd.Flags().StringVar(&getReq.RunId, "run-id", "", `ID of the run to fetch.`)
	getCmd.Flags().StringVar(&getReq.RunUuid, "run-uuid", "", `[Deprecated, use run_id instead] ID of the run to fetch.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get a run "Gets the metadata, metrics, params, and tags for a run.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.MLflowRuns.Get(ctx, getReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var logBatchReq mlflow.LogBatch

func init() {
	Cmd.AddCommand(logBatchCmd)
	// TODO: short flags

	// TODO: complex arg: metrics
	// TODO: complex arg: params
	logBatchCmd.Flags().StringVar(&logBatchReq.RunId, "run-id", "", `ID of the run to log under.`)
	// TODO: complex arg: tags

}

var logBatchCmd = &cobra.Command{
	Use:   "log-batch",
	Short: `Log a batch Logs a batch of metrics, params, and tags for a run.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	logMetricCmd.Flags().StringVar(&logMetricReq.Key, "key", "", `Name of the metric.`)
	logMetricCmd.Flags().StringVar(&logMetricReq.RunId, "run-id", "", `ID of the run under which to log the metric.`)
	logMetricCmd.Flags().StringVar(&logMetricReq.RunUuid, "run-uuid", "", `[Deprecated, use run_id instead] ID of the run under which to log the metric.`)
	logMetricCmd.Flags().Int64Var(&logMetricReq.Step, "step", 0, `Step at which to log the metric.`)
	logMetricCmd.Flags().Int64Var(&logMetricReq.Timestamp, "timestamp", 0, `Unix timestamp in milliseconds at the time metric was logged.`)
	logMetricCmd.Flags().Float64Var(&logMetricReq.Value, "value", 0, `Double value of the metric being logged.`)

}

var logMetricCmd = &cobra.Command{
	Use:   "log-metric",
	Short: `Log a metric Logs a metric for a run.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	logModelCmd.Flags().StringVar(&logModelReq.ModelJson, "model-json", "", `MLmodel file in json format.`)
	logModelCmd.Flags().StringVar(&logModelReq.RunId, "run-id", "", `ID of the run to log under.`)

}

var logModelCmd = &cobra.Command{
	Use:   "log-model",
	Short: `Log a model **NOTE:** Experimental: This API may change or be removed in a future release without warning.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	logParameterCmd.Flags().StringVar(&logParameterReq.Key, "key", "", `Name of the param.`)
	logParameterCmd.Flags().StringVar(&logParameterReq.RunId, "run-id", "", `ID of the run under which to log the param.`)
	logParameterCmd.Flags().StringVar(&logParameterReq.RunUuid, "run-uuid", "", `[Deprecated, use run_id instead] ID of the run under which to log the param.`)
	logParameterCmd.Flags().StringVar(&logParameterReq.Value, "value", "", `String value of the param being logged.`)

}

var logParameterCmd = &cobra.Command{
	Use:   "log-parameter",
	Short: `Log a param Logs a param used for a run.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	restoreCmd.Flags().StringVar(&restoreReq.RunId, "run-id", "", `ID of the run to restore.`)

}

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: `Restore a run Restores a deleted run.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	// TODO: complex arg: experiment_ids
	searchCmd.Flags().StringVar(&searchReq.Filter, "filter", "", `A filter expression over params, metrics, and tags, that allows returning a subset of runs.`)
	searchCmd.Flags().IntVar(&searchReq.MaxResults, "max-results", 0, `Maximum number of runs desired.`)
	// TODO: complex arg: order_by
	searchCmd.Flags().StringVar(&searchReq.PageToken, "page-token", "", `Token for the current page of runs.`)
	// TODO: complex arg: run_view_type

}

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: `Search for runs Searches for runs that satisfy expressions.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.MLflowRuns.SearchAll(ctx, searchReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}

var setTagReq mlflow.SetTag

func init() {
	Cmd.AddCommand(setTagCmd)
	// TODO: short flags

	setTagCmd.Flags().StringVar(&setTagReq.Key, "key", "", `Name of the tag.`)
	setTagCmd.Flags().StringVar(&setTagReq.RunId, "run-id", "", `ID of the run under which to log the tag.`)
	setTagCmd.Flags().StringVar(&setTagReq.RunUuid, "run-uuid", "", `[Deprecated, use run_id instead] ID of the run under which to log the tag.`)
	setTagCmd.Flags().StringVar(&setTagReq.Value, "value", "", `String value of the tag being logged.`)

}

var setTagCmd = &cobra.Command{
	Use:   "set-tag",
	Short: `Set a tag Sets a tag on a run.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	updateCmd.Flags().Int64Var(&updateReq.EndTime, "end-time", 0, `Unix timestamp in milliseconds of when the run ended.`)
	updateCmd.Flags().StringVar(&updateReq.RunId, "run-id", "", `ID of the run to update.`)
	updateCmd.Flags().StringVar(&updateReq.RunUuid, "run-uuid", "", `[Deprecated, use run_id instead] ID of the run to update.`)
	// TODO: complex arg: status

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update a run Updates run metadata.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		response, err := w.MLflowRuns.Update(ctx, updateReq)
		if err != nil {
			return err
		}

		pretty, err := ui.MarshalJSON(response)
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(pretty)

		return nil
	},
}
