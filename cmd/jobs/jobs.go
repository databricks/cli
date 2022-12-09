package jobs

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "jobs",
	Short: `The Jobs API allows you to create, edit, and delete jobs.`,
	Long: `The Jobs API allows you to create, edit, and delete jobs.
  
  You can use a Databricks job to run a data processing or data analysis task in
  a Databricks cluster with scalable resources. Your job can consist of a single
  task or can be a large, multi-task workflow with complex dependencies.
  Databricks manages the task orchestration, cluster management, monitoring, and
  error reporting for all of your jobs. You can run your jobs immediately or
  periodically through an easy-to-use scheduling system. You can implement job
  tasks using notebooks, JARS, Delta Live Tables pipelines, or Python, Scala,
  Spark submit, and Java applications.
  
  You should never hard code secrets or store them in plain text. Use the
  :service:secrets to manage secrets in the [Databricks CLI]. Use the [Secrets
  utility] to reference secrets in notebooks and jobs.
  
  [Databricks CLI]: https://docs.databricks.com/dev-tools/cli/index.html
  [Secrets utility]: https://docs.databricks.com/dev-tools/databricks-utils.html#dbutils-secrets`,
}

var cancelAllRunsReq jobs.CancelAllRuns

func init() {
	Cmd.AddCommand(cancelAllRunsCmd)
	// TODO: short flags

	cancelAllRunsCmd.Flags().Int64Var(&cancelAllRunsReq.JobId, "job-id", 0, `The canonical identifier of the job to cancel all runs of.`)

}

var cancelAllRunsCmd = &cobra.Command{
	Use:   "cancel-all-runs",
	Short: `Cancel all runs of a job.`,
	Long: `Cancel all runs of a job.
  
  Cancels all active runs of a job. The runs are canceled asynchronously, so it
  doesn't prevent new runs from being started.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Jobs.CancelAllRuns(ctx, cancelAllRunsReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var cancelRunReq jobs.CancelRun

func init() {
	Cmd.AddCommand(cancelRunCmd)
	// TODO: short flags

	cancelRunCmd.Flags().Int64Var(&cancelRunReq.RunId, "run-id", 0, `This field is required.`)

}

var cancelRunCmd = &cobra.Command{
	Use:   "cancel-run",
	Short: `Cancel a job run.`,
	Long: `Cancel a job run.
  
  Cancels a job run. The run is canceled asynchronously, so it may still be
  running when this request completes.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Jobs.CancelRun(ctx, cancelRunReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var createReq jobs.CreateJob

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	// TODO: array: access_control_list
	// TODO: complex arg: email_notifications
	createCmd.Flags().Var(&createReq.Format, "format", `Used to tell what is the format of the job.`)
	// TODO: complex arg: git_source
	// TODO: array: job_clusters
	createCmd.Flags().IntVar(&createReq.MaxConcurrentRuns, "max-concurrent-runs", 0, `An optional maximum allowed number of concurrent runs of the job.`)
	createCmd.Flags().StringVar(&createReq.Name, "name", "", `An optional name for the job.`)
	// TODO: complex arg: schedule
	// TODO: map via StringToStringVar: tags
	// TODO: array: tasks
	createCmd.Flags().IntVar(&createReq.TimeoutSeconds, "timeout-seconds", 0, `An optional timeout applied to each run of this job.`)
	// TODO: complex arg: webhook_notifications

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a new job.`,
	Long: `Create a new job.
  
  Create a new job.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Jobs.Create(ctx, createReq)
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

var deleteReq jobs.DeleteJob

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().Int64Var(&deleteReq.JobId, "job-id", 0, `The canonical identifier of the job to delete.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a job.`,
	Long: `Delete a job.
  
  Deletes a job.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Jobs.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var deleteRunReq jobs.DeleteRun

func init() {
	Cmd.AddCommand(deleteRunCmd)
	// TODO: short flags

	deleteRunCmd.Flags().Int64Var(&deleteRunReq.RunId, "run-id", 0, `The canonical identifier of the run for which to retrieve the metadata.`)

}

var deleteRunCmd = &cobra.Command{
	Use:   "delete-run",
	Short: `Delete a job run.`,
	Long: `Delete a job run.
  
  Deletes a non-active run. Returns an error if the run is active.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Jobs.DeleteRun(ctx, deleteRunReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var exportRunReq jobs.ExportRun

func init() {
	Cmd.AddCommand(exportRunCmd)
	// TODO: short flags

	exportRunCmd.Flags().Int64Var(&exportRunReq.RunId, "run-id", 0, `The canonical identifier for the run.`)
	exportRunCmd.Flags().Var(&exportRunReq.ViewsToExport, "views-to-export", `Which views to export (CODE, DASHBOARDS, or ALL).`)

}

var exportRunCmd = &cobra.Command{
	Use:   "export-run",
	Short: `Export and retrieve a job run.`,
	Long: `Export and retrieve a job run.
  
  Export and retrieve the job run task.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Jobs.ExportRun(ctx, exportRunReq)
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

var getReq jobs.Get

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().Int64Var(&getReq.JobId, "job-id", 0, `The canonical identifier of the job to retrieve information about.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get a single job.`,
	Long: `Get a single job.
  
  Retrieves the details for a single job.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Jobs.Get(ctx, getReq)
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

var getRunReq jobs.GetRun

func init() {
	Cmd.AddCommand(getRunCmd)
	// TODO: short flags

	getRunCmd.Flags().BoolVar(&getRunReq.IncludeHistory, "include-history", false, `Whether to include the repair history in the response.`)
	getRunCmd.Flags().Int64Var(&getRunReq.RunId, "run-id", 0, `The canonical identifier of the run for which to retrieve the metadata.`)

}

var getRunCmd = &cobra.Command{
	Use:   "get-run",
	Short: `Get a single job run.`,
	Long: `Get a single job run.
  
  Retrieve the metadata of a run.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Jobs.GetRun(ctx, getRunReq)
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

var getRunOutputReq jobs.GetRunOutput

func init() {
	Cmd.AddCommand(getRunOutputCmd)
	// TODO: short flags

	getRunOutputCmd.Flags().Int64Var(&getRunOutputReq.RunId, "run-id", 0, `The canonical identifier for the run.`)

}

var getRunOutputCmd = &cobra.Command{
	Use:   "get-run-output",
	Short: `Get the output for a single run.`,
	Long: `Get the output for a single run.
  
  Retrieve the output and metadata of a single task run. When a notebook task
  returns a value through the dbutils.notebook.exit() call, you can use this
  endpoint to retrieve that value. Databricks restricts this API to returning
  the first 5 MB of the output. To return a larger result, you can store job
  results in a cloud storage service.
  
  This endpoint validates that the __run_id__ parameter is valid and returns an
  HTTP status code 400 if the __run_id__ parameter is invalid. Runs are
  automatically removed after 60 days. If you to want to reference them beyond
  60 days, you must save old run results before they expire.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Jobs.GetRunOutput(ctx, getRunOutputReq)
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

var listReq jobs.List

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	listCmd.Flags().BoolVar(&listReq.ExpandTasks, "expand-tasks", false, `Whether to include task and cluster details in the response.`)
	listCmd.Flags().IntVar(&listReq.Limit, "limit", 0, `The number of jobs to return.`)
	listCmd.Flags().StringVar(&listReq.Name, "name", "", `A filter on the list based on the exact (case insensitive) job name.`)
	listCmd.Flags().IntVar(&listReq.Offset, "offset", 0, `The offset of the first job to return, relative to the most recently created job.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List all jobs.`,
	Long: `List all jobs.
  
  Retrieves a list of jobs.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Jobs.ListAll(ctx, listReq)
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

var listRunsReq jobs.ListRuns

func init() {
	Cmd.AddCommand(listRunsCmd)
	// TODO: short flags

	listRunsCmd.Flags().BoolVar(&listRunsReq.ActiveOnly, "active-only", false, `If active_only is true, only active runs are included in the results; otherwise, lists both active and completed runs.`)
	listRunsCmd.Flags().BoolVar(&listRunsReq.CompletedOnly, "completed-only", false, `If completed_only is true, only completed runs are included in the results; otherwise, lists both active and completed runs.`)
	listRunsCmd.Flags().BoolVar(&listRunsReq.ExpandTasks, "expand-tasks", false, `Whether to include task and cluster details in the response.`)
	listRunsCmd.Flags().Int64Var(&listRunsReq.JobId, "job-id", 0, `The job for which to list runs.`)
	listRunsCmd.Flags().IntVar(&listRunsReq.Limit, "limit", 0, `The number of runs to return.`)
	listRunsCmd.Flags().IntVar(&listRunsReq.Offset, "offset", 0, `The offset of the first run to return, relative to the most recent run.`)
	listRunsCmd.Flags().Var(&listRunsReq.RunType, "run-type", `The type of runs to return.`)
	listRunsCmd.Flags().IntVar(&listRunsReq.StartTimeFrom, "start-time-from", 0, `Show runs that started _at or after_ this value.`)
	listRunsCmd.Flags().IntVar(&listRunsReq.StartTimeTo, "start-time-to", 0, `Show runs that started _at or before_ this value.`)

}

var listRunsCmd = &cobra.Command{
	Use:   "list-runs",
	Short: `List runs for a job.`,
	Long: `List runs for a job.
  
  List runs in descending order by start time.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Jobs.ListRunsAll(ctx, listRunsReq)
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

var repairRunReq jobs.RepairRun

func init() {
	Cmd.AddCommand(repairRunCmd)
	// TODO: short flags

	// TODO: array: dbt_commands
	// TODO: array: jar_params
	repairRunCmd.Flags().Int64Var(&repairRunReq.LatestRepairId, "latest-repair-id", 0, `The ID of the latest repair.`)
	// TODO: map via StringToStringVar: notebook_params
	// TODO: complex arg: pipeline_params
	// TODO: map via StringToStringVar: python_named_params
	// TODO: array: python_params
	repairRunCmd.Flags().BoolVar(&repairRunReq.RerunAllFailedTasks, "rerun-all-failed-tasks", false, `If true, repair all failed tasks.`)
	// TODO: array: rerun_tasks
	repairRunCmd.Flags().Int64Var(&repairRunReq.RunId, "run-id", 0, `The job run ID of the run to repair.`)
	// TODO: array: spark_submit_params
	// TODO: map via StringToStringVar: sql_params

}

var repairRunCmd = &cobra.Command{
	Use:   "repair-run",
	Short: `Repair a job run.`,
	Long: `Repair a job run.
  
  Re-run one or more tasks. Tasks are re-run as part of the original job run.
  They use the current job and task settings, and can be viewed in the history
  for the original job run.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Jobs.RepairRun(ctx, repairRunReq)
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

var resetReq jobs.ResetJob

func init() {
	Cmd.AddCommand(resetCmd)
	// TODO: short flags

	resetCmd.Flags().Int64Var(&resetReq.JobId, "job-id", 0, `The canonical identifier of the job to reset.`)
	// TODO: complex arg: new_settings

}

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: `Overwrites all settings for a job.`,
	Long: `Overwrites all settings for a job.
  
  Overwrites all the settings for a specific job. Use the Update endpoint to
  update job settings partially.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Jobs.Reset(ctx, resetReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var runNowReq jobs.RunNow

func init() {
	Cmd.AddCommand(runNowCmd)
	// TODO: short flags

	// TODO: array: dbt_commands
	runNowCmd.Flags().StringVar(&runNowReq.IdempotencyToken, "idempotency-token", "", `An optional token to guarantee the idempotency of job run requests.`)
	// TODO: array: jar_params
	runNowCmd.Flags().Int64Var(&runNowReq.JobId, "job-id", 0, `The ID of the job to be executed.`)
	// TODO: map via StringToStringVar: notebook_params
	// TODO: complex arg: pipeline_params
	// TODO: map via StringToStringVar: python_named_params
	// TODO: array: python_params
	// TODO: array: spark_submit_params
	// TODO: map via StringToStringVar: sql_params

}

var runNowCmd = &cobra.Command{
	Use:   "run-now",
	Short: `Trigger a new job run.`,
	Long: `Trigger a new job run.
  
  Run a job and return the run_id of the triggered run.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Jobs.RunNow(ctx, runNowReq)
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

var submitReq jobs.SubmitRun

func init() {
	Cmd.AddCommand(submitCmd)
	// TODO: short flags

	// TODO: array: access_control_list
	// TODO: complex arg: git_source
	submitCmd.Flags().StringVar(&submitReq.IdempotencyToken, "idempotency-token", "", `An optional token that can be used to guarantee the idempotency of job run requests.`)
	submitCmd.Flags().StringVar(&submitReq.RunName, "run-name", "", `An optional name for the run.`)
	// TODO: array: tasks
	submitCmd.Flags().IntVar(&submitReq.TimeoutSeconds, "timeout-seconds", 0, `An optional timeout applied to each run of this job.`)
	// TODO: complex arg: webhook_notifications

}

var submitCmd = &cobra.Command{
	Use:   "submit",
	Short: `Create and trigger a one-time run.`,
	Long: `Create and trigger a one-time run.
  
  Submit a one-time run. This endpoint allows you to submit a workload directly
  without creating a job. Runs submitted using this endpoint don’t display in
  the UI. Use the jobs/runs/get API to check the run state after the job is
  submitted.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Jobs.Submit(ctx, submitReq)
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

var updateReq jobs.UpdateJob

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	// TODO: array: fields_to_remove
	updateCmd.Flags().Int64Var(&updateReq.JobId, "job-id", 0, `The canonical identifier of the job to update.`)
	// TODO: complex arg: new_settings

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Partially updates a job.`,
	Long: `Partially updates a job.
  
  Add, update, or remove specific settings of an existing job. Use the ResetJob
  to overwrite all job settings.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Jobs.Update(ctx, updateReq)
		if err != nil {
			return err
		}

		return nil
	},
}

// end service Jobs

func init() {
	Cmd.PersistentFlags().String("profile", "", "~/.databrickscfg profile")

}
