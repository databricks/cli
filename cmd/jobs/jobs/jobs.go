package jobs

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "jobs",
	Short: `The Jobs API allows you to create, edit, and delete jobs.`, // TODO: fix FirstSentence logic and append dot to summary
}

var cancelAllRunsReq jobs.CancelAllRuns

func init() {
	Cmd.AddCommand(cancelAllRunsCmd)
	// TODO: short flags

	cancelAllRunsCmd.Flags().Int64Var(&cancelAllRunsReq.JobId, "job-id", 0, `The canonical identifier of the job to cancel all runs of.`)

}

var cancelAllRunsCmd = &cobra.Command{
	Use:   "cancel-all-runs",
	Short: `Cancel all runs of a job Cancels all active runs of a job.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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
	Short: `Cancel a job run Cancels a job run.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	// TODO: complex arg: access_control_list
	// TODO: complex arg: email_notifications
	// TODO: complex arg: format
	// TODO: complex arg: git_source
	// TODO: complex arg: job_clusters
	createCmd.Flags().IntVar(&createReq.MaxConcurrentRuns, "max-concurrent-runs", 0, `An optional maximum allowed number of concurrent runs of the job.`)
	createCmd.Flags().StringVar(&createReq.Name, "name", "", `An optional name for the job.`)
	// TODO: complex arg: schedule
	// TODO: complex arg: tags
	// TODO: complex arg: tasks
	createCmd.Flags().IntVar(&createReq.TimeoutSeconds, "timeout-seconds", 0, `An optional timeout applied to each run of this job.`)

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a new job Create a new job.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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
	Short: `Delete a job Deletes a job.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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
	Short: `Delete a job run Deletes a non-active run.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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
	// TODO: complex arg: views_to_export

}

var exportRunCmd = &cobra.Command{
	Use:   "export-run",
	Short: `Export and retrieve a job run Export and retrieve the job run task.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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
	Short: `Get a single job Retrieves the details for a single job.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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
	Short: `Get a single job run Retrieve the metadata of a run.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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
	Short: `Get the output for a single run Retrieve the output and metadata of a single task run.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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
	Short: `List all jobs Retrieves a list of jobs.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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
	// TODO: complex arg: run_type
	listRunsCmd.Flags().IntVar(&listRunsReq.StartTimeFrom, "start-time-from", 0, `Show runs that started _at or after_ this value.`)
	listRunsCmd.Flags().IntVar(&listRunsReq.StartTimeTo, "start-time-to", 0, `Show runs that started _at or before_ this value.`)

}

var listRunsCmd = &cobra.Command{
	Use:   "list-runs",
	Short: `List runs for a job List runs in descending order by start time.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	// TODO: complex arg: dbt_commands
	// TODO: complex arg: jar_params
	repairRunCmd.Flags().Int64Var(&repairRunReq.LatestRepairId, "latest-repair-id", 0, `The ID of the latest repair.`)
	// TODO: complex arg: notebook_params
	// TODO: complex arg: pipeline_params
	// TODO: complex arg: python_named_params
	// TODO: complex arg: python_params
	repairRunCmd.Flags().BoolVar(&repairRunReq.RerunAllFailedTasks, "rerun-all-failed-tasks", false, `If true, repair all failed tasks.`)
	// TODO: complex arg: rerun_tasks
	repairRunCmd.Flags().Int64Var(&repairRunReq.RunId, "run-id", 0, `The job run ID of the run to repair.`)
	// TODO: complex arg: spark_submit_params
	// TODO: complex arg: sql_params

}

var repairRunCmd = &cobra.Command{
	Use:   "repair-run",
	Short: `Repair a job run Re-run one or more tasks.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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
	Short: `Overwrites all settings for a job Overwrites all the settings for a specific job.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	// TODO: complex arg: dbt_commands
	runNowCmd.Flags().StringVar(&runNowReq.IdempotencyToken, "idempotency-token", "", `An optional token to guarantee the idempotency of job run requests.`)
	// TODO: complex arg: jar_params
	runNowCmd.Flags().Int64Var(&runNowReq.JobId, "job-id", 0, `The ID of the job to be executed.`)
	// TODO: complex arg: notebook_params
	// TODO: complex arg: pipeline_params
	// TODO: complex arg: python_named_params
	// TODO: complex arg: python_params
	// TODO: complex arg: spark_submit_params
	// TODO: complex arg: sql_params

}

var runNowCmd = &cobra.Command{
	Use:   "run-now",
	Short: `Trigger a new job run Run a job and return the run_id of the triggered run.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	// TODO: complex arg: access_control_list
	// TODO: complex arg: git_source
	submitCmd.Flags().StringVar(&submitReq.IdempotencyToken, "idempotency-token", "", `An optional token that can be used to guarantee the idempotency of job run requests.`)
	submitCmd.Flags().StringVar(&submitReq.RunName, "run-name", "", `An optional name for the run.`)
	// TODO: complex arg: tasks
	submitCmd.Flags().IntVar(&submitReq.TimeoutSeconds, "timeout-seconds", 0, `An optional timeout applied to each run of this job.`)

}

var submitCmd = &cobra.Command{
	Use:   "submit",
	Short: `Create and trigger a one-time run Submit a one-time run.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	// TODO: complex arg: fields_to_remove
	updateCmd.Flags().Int64Var(&updateReq.JobId, "job-id", 0, `The canonical identifier of the job to update.`)
	// TODO: complex arg: new_settings

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Partially updates a job Add, update, or remove specific settings of an existing job.`, // TODO: fix logic

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Jobs.Update(ctx, updateReq)
		if err != nil {
			return err
		}

		return nil
	},
}
