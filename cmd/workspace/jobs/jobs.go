// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package jobs

import (
	"fmt"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
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
  [Secrets CLI] to manage secrets in the [Databricks CLI]. Use the [Secrets
  utility] to reference secrets in notebooks and jobs.
  
  [Databricks CLI]: https://docs.databricks.com/dev-tools/cli/index.html
  [Secrets CLI]: https://docs.databricks.com/dev-tools/cli/secrets-cli.html
  [Secrets utility]: https://docs.databricks.com/dev-tools/databricks-utils.html#dbutils-secrets`,
}

// start cancel-all-runs command

var cancelAllRunsReq jobs.CancelAllRuns
var cancelAllRunsJson flags.JsonFlag

func init() {
	Cmd.AddCommand(cancelAllRunsCmd)
	// TODO: short flags
	cancelAllRunsCmd.Flags().Var(&cancelAllRunsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var cancelAllRunsCmd = &cobra.Command{
	Use:   "cancel-all-runs JOB_ID",
	Short: `Cancel all runs of a job.`,
	Long: `Cancel all runs of a job.
  
  Cancels all active runs of a job. The runs are canceled asynchronously, so it
  doesn't prevent new runs from being started.`,

	Annotations: map[string]string{
		"package": "jobs",
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = cancelAllRunsJson.Unmarshal(&cancelAllRunsReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No JOB_ID argument specified. Loading names for Jobs drop-down."
				names, err := w.Jobs.BaseJobSettingsNameToJobIdMap(ctx, jobs.ListJobsRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Jobs drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The canonical identifier of the job to cancel all runs of")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have the canonical identifier of the job to cancel all runs of")
			}
			_, err = fmt.Sscan(args[0], &cancelAllRunsReq.JobId)
			if err != nil {
				return fmt.Errorf("invalid JOB_ID: %s", args[0])
			}
		}

		err = w.Jobs.CancelAllRuns(ctx, cancelAllRunsReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start cancel-run command

var cancelRunReq jobs.CancelRun
var cancelRunJson flags.JsonFlag
var cancelRunSkipWait bool
var cancelRunTimeout time.Duration

func init() {
	Cmd.AddCommand(cancelRunCmd)

	cancelRunCmd.Flags().BoolVar(&cancelRunSkipWait, "no-wait", cancelRunSkipWait, `do not wait to reach TERMINATED or SKIPPED state`)
	cancelRunCmd.Flags().DurationVar(&cancelRunTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach TERMINATED or SKIPPED state`)
	// TODO: short flags
	cancelRunCmd.Flags().Var(&cancelRunJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var cancelRunCmd = &cobra.Command{
	Use:   "cancel-run RUN_ID",
	Short: `Cancel a job run.`,
	Long: `Cancel a job run.
  
  Cancels a job run. The run is canceled asynchronously, so it may still be
  running when this request completes.`,

	Annotations: map[string]string{
		"package": "jobs",
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = cancelRunJson.Unmarshal(&cancelRunReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No RUN_ID argument specified. Loading names for Jobs drop-down."
				names, err := w.Jobs.BaseJobSettingsNameToJobIdMap(ctx, jobs.ListJobsRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Jobs drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "This field is required")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have this field is required")
			}
			_, err = fmt.Sscan(args[0], &cancelRunReq.RunId)
			if err != nil {
				return fmt.Errorf("invalid RUN_ID: %s", args[0])
			}
		}

		wait, err := w.Jobs.CancelRun(ctx, cancelRunReq)
		if err != nil {
			return err
		}
		if cancelRunSkipWait {
			return nil
		}
		spinner := cmdio.Spinner(ctx)
		info, err := wait.OnProgress(func(i *jobs.Run) {
			if i.State == nil {
				return
			}
			status := i.State.LifeCycleState
			statusMessage := fmt.Sprintf("current status: %s", status)
			if i.State != nil {
				statusMessage = i.State.StateMessage
			}
			spinner <- statusMessage
		}).GetWithTimeout(cancelRunTimeout)
		close(spinner)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, info)
	},
}

// start create command

var createReq jobs.CreateJob
var createJson flags.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list
	// TODO: complex arg: continuous
	// TODO: complex arg: email_notifications
	createCmd.Flags().Var(&createReq.Format, "format", `Used to tell what is the format of the job.`)
	// TODO: complex arg: git_source
	// TODO: array: job_clusters
	createCmd.Flags().IntVar(&createReq.MaxConcurrentRuns, "max-concurrent-runs", createReq.MaxConcurrentRuns, `An optional maximum allowed number of concurrent runs of the job.`)
	createCmd.Flags().StringVar(&createReq.Name, "name", createReq.Name, `An optional name for the job.`)
	// TODO: complex arg: notification_settings
	// TODO: complex arg: run_as
	// TODO: complex arg: schedule
	// TODO: map via StringToStringVar: tags
	// TODO: array: tasks
	createCmd.Flags().IntVar(&createReq.TimeoutSeconds, "timeout-seconds", createReq.TimeoutSeconds, `An optional timeout applied to each run of this job.`)
	// TODO: complex arg: trigger
	// TODO: complex arg: webhook_notifications

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a new job.`,
	Long: `Create a new job.
  
  Create a new job.`,

	Annotations: map[string]string{
		"package": "jobs",
	},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
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
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
		}

		response, err := w.Jobs.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start delete command

var deleteReq jobs.DeleteJob
var deleteJson flags.JsonFlag

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags
	deleteCmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete JOB_ID",
	Short: `Delete a job.`,
	Long: `Delete a job.
  
  Deletes a job.`,

	Annotations: map[string]string{
		"package": "jobs",
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
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No JOB_ID argument specified. Loading names for Jobs drop-down."
				names, err := w.Jobs.BaseJobSettingsNameToJobIdMap(ctx, jobs.ListJobsRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Jobs drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The canonical identifier of the job to delete")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have the canonical identifier of the job to delete")
			}
			_, err = fmt.Sscan(args[0], &deleteReq.JobId)
			if err != nil {
				return fmt.Errorf("invalid JOB_ID: %s", args[0])
			}
		}

		err = w.Jobs.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start delete-run command

var deleteRunReq jobs.DeleteRun
var deleteRunJson flags.JsonFlag

func init() {
	Cmd.AddCommand(deleteRunCmd)
	// TODO: short flags
	deleteRunCmd.Flags().Var(&deleteRunJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var deleteRunCmd = &cobra.Command{
	Use:   "delete-run RUN_ID",
	Short: `Delete a job run.`,
	Long: `Delete a job run.
  
  Deletes a non-active run. Returns an error if the run is active.`,

	Annotations: map[string]string{
		"package": "jobs",
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = deleteRunJson.Unmarshal(&deleteRunReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No RUN_ID argument specified. Loading names for Jobs drop-down."
				names, err := w.Jobs.BaseJobSettingsNameToJobIdMap(ctx, jobs.ListJobsRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Jobs drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The canonical identifier of the run for which to retrieve the metadata")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have the canonical identifier of the run for which to retrieve the metadata")
			}
			_, err = fmt.Sscan(args[0], &deleteRunReq.RunId)
			if err != nil {
				return fmt.Errorf("invalid RUN_ID: %s", args[0])
			}
		}

		err = w.Jobs.DeleteRun(ctx, deleteRunReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start export-run command

var exportRunReq jobs.ExportRunRequest
var exportRunJson flags.JsonFlag

func init() {
	Cmd.AddCommand(exportRunCmd)
	// TODO: short flags
	exportRunCmd.Flags().Var(&exportRunJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	exportRunCmd.Flags().Var(&exportRunReq.ViewsToExport, "views-to-export", `Which views to export (CODE, DASHBOARDS, or ALL).`)

}

var exportRunCmd = &cobra.Command{
	Use:   "export-run RUN_ID",
	Short: `Export and retrieve a job run.`,
	Long: `Export and retrieve a job run.
  
  Export and retrieve the job run task.`,

	Annotations: map[string]string{
		"package": "jobs",
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = exportRunJson.Unmarshal(&exportRunReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No RUN_ID argument specified. Loading names for Jobs drop-down."
				names, err := w.Jobs.BaseJobSettingsNameToJobIdMap(ctx, jobs.ListJobsRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Jobs drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The canonical identifier for the run")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have the canonical identifier for the run")
			}
			_, err = fmt.Sscan(args[0], &exportRunReq.RunId)
			if err != nil {
				return fmt.Errorf("invalid RUN_ID: %s", args[0])
			}
		}

		response, err := w.Jobs.ExportRun(ctx, exportRunReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start get command

var getReq jobs.GetJobRequest
var getJson flags.JsonFlag

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags
	getCmd.Flags().Var(&getJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var getCmd = &cobra.Command{
	Use:   "get JOB_ID",
	Short: `Get a single job.`,
	Long: `Get a single job.
  
  Retrieves the details for a single job.`,

	Annotations: map[string]string{
		"package": "jobs",
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
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No JOB_ID argument specified. Loading names for Jobs drop-down."
				names, err := w.Jobs.BaseJobSettingsNameToJobIdMap(ctx, jobs.ListJobsRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Jobs drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The canonical identifier of the job to retrieve information about")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have the canonical identifier of the job to retrieve information about")
			}
			_, err = fmt.Sscan(args[0], &getReq.JobId)
			if err != nil {
				return fmt.Errorf("invalid JOB_ID: %s", args[0])
			}
		}

		response, err := w.Jobs.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start get-run command

var getRunReq jobs.GetRunRequest
var getRunJson flags.JsonFlag
var getRunSkipWait bool
var getRunTimeout time.Duration

func init() {
	Cmd.AddCommand(getRunCmd)

	getRunCmd.Flags().BoolVar(&getRunSkipWait, "no-wait", getRunSkipWait, `do not wait to reach TERMINATED or SKIPPED state`)
	getRunCmd.Flags().DurationVar(&getRunTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach TERMINATED or SKIPPED state`)
	// TODO: short flags
	getRunCmd.Flags().Var(&getRunJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	getRunCmd.Flags().BoolVar(&getRunReq.IncludeHistory, "include-history", getRunReq.IncludeHistory, `Whether to include the repair history in the response.`)

}

var getRunCmd = &cobra.Command{
	Use:   "get-run RUN_ID",
	Short: `Get a single job run.`,
	Long: `Get a single job run.
  
  Retrieve the metadata of a run.`,

	Annotations: map[string]string{
		"package": "jobs",
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = getRunJson.Unmarshal(&getRunReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No RUN_ID argument specified. Loading names for Jobs drop-down."
				names, err := w.Jobs.BaseJobSettingsNameToJobIdMap(ctx, jobs.ListJobsRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Jobs drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The canonical identifier of the run for which to retrieve the metadata")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have the canonical identifier of the run for which to retrieve the metadata")
			}
			_, err = fmt.Sscan(args[0], &getRunReq.RunId)
			if err != nil {
				return fmt.Errorf("invalid RUN_ID: %s", args[0])
			}
		}

		response, err := w.Jobs.GetRun(ctx, getRunReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start get-run-output command

var getRunOutputReq jobs.GetRunOutputRequest
var getRunOutputJson flags.JsonFlag

func init() {
	Cmd.AddCommand(getRunOutputCmd)
	// TODO: short flags
	getRunOutputCmd.Flags().Var(&getRunOutputJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var getRunOutputCmd = &cobra.Command{
	Use:   "get-run-output RUN_ID",
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

	Annotations: map[string]string{
		"package": "jobs",
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = getRunOutputJson.Unmarshal(&getRunOutputReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No RUN_ID argument specified. Loading names for Jobs drop-down."
				names, err := w.Jobs.BaseJobSettingsNameToJobIdMap(ctx, jobs.ListJobsRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Jobs drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The canonical identifier for the run")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have the canonical identifier for the run")
			}
			_, err = fmt.Sscan(args[0], &getRunOutputReq.RunId)
			if err != nil {
				return fmt.Errorf("invalid RUN_ID: %s", args[0])
			}
		}

		response, err := w.Jobs.GetRunOutput(ctx, getRunOutputReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start list command

var listReq jobs.ListJobsRequest
var listJson flags.JsonFlag

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags
	listCmd.Flags().Var(&listJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	listCmd.Flags().BoolVar(&listReq.ExpandTasks, "expand-tasks", listReq.ExpandTasks, `Whether to include task and cluster details in the response.`)
	listCmd.Flags().IntVar(&listReq.Limit, "limit", listReq.Limit, `The number of jobs to return.`)
	listCmd.Flags().StringVar(&listReq.Name, "name", listReq.Name, `A filter on the list based on the exact (case insensitive) job name.`)
	listCmd.Flags().IntVar(&listReq.Offset, "offset", listReq.Offset, `The offset of the first job to return, relative to the most recently created job.`)
	listCmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Use next_page_token or prev_page_token returned from the previous request to list the next or previous page of jobs respectively.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List all jobs.`,
	Long: `List all jobs.
  
  Retrieves a list of jobs.`,

	Annotations: map[string]string{
		"package": "jobs",
	},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
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
			err = listJson.Unmarshal(&listReq)
			if err != nil {
				return err
			}
		} else {
		}

		response, err := w.Jobs.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start list-runs command

var listRunsReq jobs.ListRunsRequest
var listRunsJson flags.JsonFlag

func init() {
	Cmd.AddCommand(listRunsCmd)
	// TODO: short flags
	listRunsCmd.Flags().Var(&listRunsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	listRunsCmd.Flags().BoolVar(&listRunsReq.ActiveOnly, "active-only", listRunsReq.ActiveOnly, `If active_only is true, only active runs are included in the results; otherwise, lists both active and completed runs.`)
	listRunsCmd.Flags().BoolVar(&listRunsReq.CompletedOnly, "completed-only", listRunsReq.CompletedOnly, `If completed_only is true, only completed runs are included in the results; otherwise, lists both active and completed runs.`)
	listRunsCmd.Flags().BoolVar(&listRunsReq.ExpandTasks, "expand-tasks", listRunsReq.ExpandTasks, `Whether to include task and cluster details in the response.`)
	listRunsCmd.Flags().Int64Var(&listRunsReq.JobId, "job-id", listRunsReq.JobId, `The job for which to list runs.`)
	listRunsCmd.Flags().IntVar(&listRunsReq.Limit, "limit", listRunsReq.Limit, `The number of runs to return.`)
	listRunsCmd.Flags().IntVar(&listRunsReq.Offset, "offset", listRunsReq.Offset, `The offset of the first run to return, relative to the most recent run.`)
	listRunsCmd.Flags().StringVar(&listRunsReq.PageToken, "page-token", listRunsReq.PageToken, `Use next_page_token or prev_page_token returned from the previous request to list the next or previous page of runs respectively.`)
	listRunsCmd.Flags().Var(&listRunsReq.RunType, "run-type", `The type of runs to return.`)
	listRunsCmd.Flags().IntVar(&listRunsReq.StartTimeFrom, "start-time-from", listRunsReq.StartTimeFrom, `Show runs that started _at or after_ this value.`)
	listRunsCmd.Flags().IntVar(&listRunsReq.StartTimeTo, "start-time-to", listRunsReq.StartTimeTo, `Show runs that started _at or before_ this value.`)

}

var listRunsCmd = &cobra.Command{
	Use:   "list-runs",
	Short: `List runs for a job.`,
	Long: `List runs for a job.
  
  List runs in descending order by start time.`,

	Annotations: map[string]string{
		"package": "jobs",
	},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
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
			err = listRunsJson.Unmarshal(&listRunsReq)
			if err != nil {
				return err
			}
		} else {
		}

		response, err := w.Jobs.ListRunsAll(ctx, listRunsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start repair-run command

var repairRunReq jobs.RepairRun
var repairRunJson flags.JsonFlag
var repairRunSkipWait bool
var repairRunTimeout time.Duration

func init() {
	Cmd.AddCommand(repairRunCmd)

	repairRunCmd.Flags().BoolVar(&repairRunSkipWait, "no-wait", repairRunSkipWait, `do not wait to reach TERMINATED or SKIPPED state`)
	repairRunCmd.Flags().DurationVar(&repairRunTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach TERMINATED or SKIPPED state`)
	// TODO: short flags
	repairRunCmd.Flags().Var(&repairRunJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: dbt_commands
	// TODO: array: jar_params
	repairRunCmd.Flags().Int64Var(&repairRunReq.LatestRepairId, "latest-repair-id", repairRunReq.LatestRepairId, `The ID of the latest repair.`)
	// TODO: map via StringToStringVar: notebook_params
	// TODO: complex arg: pipeline_params
	// TODO: map via StringToStringVar: python_named_params
	// TODO: array: python_params
	repairRunCmd.Flags().BoolVar(&repairRunReq.RerunAllFailedTasks, "rerun-all-failed-tasks", repairRunReq.RerunAllFailedTasks, `If true, repair all failed tasks.`)
	// TODO: array: rerun_tasks
	// TODO: array: spark_submit_params
	// TODO: map via StringToStringVar: sql_params

}

var repairRunCmd = &cobra.Command{
	Use:   "repair-run RUN_ID",
	Short: `Repair a job run.`,
	Long: `Repair a job run.
  
  Re-run one or more tasks. Tasks are re-run as part of the original job run.
  They use the current job and task settings, and can be viewed in the history
  for the original job run.`,

	Annotations: map[string]string{
		"package": "jobs",
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = repairRunJson.Unmarshal(&repairRunReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No RUN_ID argument specified. Loading names for Jobs drop-down."
				names, err := w.Jobs.BaseJobSettingsNameToJobIdMap(ctx, jobs.ListJobsRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Jobs drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The job run ID of the run to repair")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have the job run id of the run to repair")
			}
			_, err = fmt.Sscan(args[0], &repairRunReq.RunId)
			if err != nil {
				return fmt.Errorf("invalid RUN_ID: %s", args[0])
			}
		}

		wait, err := w.Jobs.RepairRun(ctx, repairRunReq)
		if err != nil {
			return err
		}
		if repairRunSkipWait {
			return cmdio.Render(ctx, wait.Response)
		}
		spinner := cmdio.Spinner(ctx)
		info, err := wait.OnProgress(func(i *jobs.Run) {
			if i.State == nil {
				return
			}
			status := i.State.LifeCycleState
			statusMessage := fmt.Sprintf("current status: %s", status)
			if i.State != nil {
				statusMessage = i.State.StateMessage
			}
			spinner <- statusMessage
		}).GetWithTimeout(repairRunTimeout)
		close(spinner)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, info)
	},
}

// start reset command

var resetReq jobs.ResetJob
var resetJson flags.JsonFlag

func init() {
	Cmd.AddCommand(resetCmd)
	// TODO: short flags
	resetCmd.Flags().Var(&resetJson, "json", `either inline JSON string or @path/to/file.json with request body`)

}

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: `Overwrites all settings for a job.`,
	Long: `Overwrites all settings for a job.
  
  Overwrites all the settings for a specific job. Use the Update endpoint to
  update job settings partially.`,

	Annotations: map[string]string{
		"package": "jobs",
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = resetJson.Unmarshal(&resetReq)
			if err != nil {
				return err
			}
		} else {
			_, err = fmt.Sscan(args[0], &resetReq.JobId)
			if err != nil {
				return fmt.Errorf("invalid JOB_ID: %s", args[0])
			}
			_, err = fmt.Sscan(args[1], &resetReq.NewSettings)
			if err != nil {
				return fmt.Errorf("invalid NEW_SETTINGS: %s", args[1])
			}
		}

		err = w.Jobs.Reset(ctx, resetReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start run-now command

var runNowReq jobs.RunNow
var runNowJson flags.JsonFlag
var runNowSkipWait bool
var runNowTimeout time.Duration

func init() {
	Cmd.AddCommand(runNowCmd)

	runNowCmd.Flags().BoolVar(&runNowSkipWait, "no-wait", runNowSkipWait, `do not wait to reach TERMINATED or SKIPPED state`)
	runNowCmd.Flags().DurationVar(&runNowTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach TERMINATED or SKIPPED state`)
	// TODO: short flags
	runNowCmd.Flags().Var(&runNowJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: dbt_commands
	runNowCmd.Flags().StringVar(&runNowReq.IdempotencyToken, "idempotency-token", runNowReq.IdempotencyToken, `An optional token to guarantee the idempotency of job run requests.`)
	// TODO: array: jar_params
	// TODO: map via StringToStringVar: notebook_params
	// TODO: complex arg: pipeline_params
	// TODO: map via StringToStringVar: python_named_params
	// TODO: array: python_params
	// TODO: array: spark_submit_params
	// TODO: map via StringToStringVar: sql_params

}

var runNowCmd = &cobra.Command{
	Use:   "run-now JOB_ID",
	Short: `Trigger a new job run.`,
	Long: `Trigger a new job run.
  
  Run a job and return the run_id of the triggered run.`,

	Annotations: map[string]string{
		"package": "jobs",
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = runNowJson.Unmarshal(&runNowReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No JOB_ID argument specified. Loading names for Jobs drop-down."
				names, err := w.Jobs.BaseJobSettingsNameToJobIdMap(ctx, jobs.ListJobsRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Jobs drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The ID of the job to be executed")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have the id of the job to be executed")
			}
			_, err = fmt.Sscan(args[0], &runNowReq.JobId)
			if err != nil {
				return fmt.Errorf("invalid JOB_ID: %s", args[0])
			}
		}

		wait, err := w.Jobs.RunNow(ctx, runNowReq)
		if err != nil {
			return err
		}
		if runNowSkipWait {
			return cmdio.Render(ctx, wait.Response)
		}
		spinner := cmdio.Spinner(ctx)
		info, err := wait.OnProgress(func(i *jobs.Run) {
			if i.State == nil {
				return
			}
			status := i.State.LifeCycleState
			statusMessage := fmt.Sprintf("current status: %s", status)
			if i.State != nil {
				statusMessage = i.State.StateMessage
			}
			spinner <- statusMessage
		}).GetWithTimeout(runNowTimeout)
		close(spinner)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, info)
	},
}

// start submit command

var submitReq jobs.SubmitRun
var submitJson flags.JsonFlag
var submitSkipWait bool
var submitTimeout time.Duration

func init() {
	Cmd.AddCommand(submitCmd)

	submitCmd.Flags().BoolVar(&submitSkipWait, "no-wait", submitSkipWait, `do not wait to reach TERMINATED or SKIPPED state`)
	submitCmd.Flags().DurationVar(&submitTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach TERMINATED or SKIPPED state`)
	// TODO: short flags
	submitCmd.Flags().Var(&submitJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list
	// TODO: complex arg: git_source
	submitCmd.Flags().StringVar(&submitReq.IdempotencyToken, "idempotency-token", submitReq.IdempotencyToken, `An optional token that can be used to guarantee the idempotency of job run requests.`)
	// TODO: complex arg: notification_settings
	submitCmd.Flags().StringVar(&submitReq.RunName, "run-name", submitReq.RunName, `An optional name for the run.`)
	// TODO: array: tasks
	submitCmd.Flags().IntVar(&submitReq.TimeoutSeconds, "timeout-seconds", submitReq.TimeoutSeconds, `An optional timeout applied to each run of this job.`)
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

	Annotations: map[string]string{
		"package": "jobs",
	},
	Args: func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
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
			err = submitJson.Unmarshal(&submitReq)
			if err != nil {
				return err
			}
		} else {
		}

		wait, err := w.Jobs.Submit(ctx, submitReq)
		if err != nil {
			return err
		}
		if submitSkipWait {
			return cmdio.Render(ctx, wait.Response)
		}
		spinner := cmdio.Spinner(ctx)
		info, err := wait.OnProgress(func(i *jobs.Run) {
			if i.State == nil {
				return
			}
			status := i.State.LifeCycleState
			statusMessage := fmt.Sprintf("current status: %s", status)
			if i.State != nil {
				statusMessage = i.State.StateMessage
			}
			spinner <- statusMessage
		}).GetWithTimeout(submitTimeout)
		close(spinner)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, info)
	},
}

// start update command

var updateReq jobs.UpdateJob
var updateJson flags.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: fields_to_remove
	// TODO: complex arg: new_settings

}

var updateCmd = &cobra.Command{
	Use:   "update JOB_ID",
	Short: `Partially update a job.`,
	Long: `Partially update a job.
  
  Add, update, or remove specific settings of an existing job. Use the ResetJob
  to overwrite all job settings.`,

	Annotations: map[string]string{
		"package": "jobs",
	},
	PreRunE: root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if cmd.Flags().Changed("json") {
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
			}
		} else {
			if len(args) == 0 {
				promptSpinner := cmdio.Spinner(ctx)
				promptSpinner <- "No JOB_ID argument specified. Loading names for Jobs drop-down."
				names, err := w.Jobs.BaseJobSettingsNameToJobIdMap(ctx, jobs.ListJobsRequest{})
				close(promptSpinner)
				if err != nil {
					return fmt.Errorf("failed to load names for Jobs drop-down. Please manually specify required arguments. Original error: %w", err)
				}
				id, err := cmdio.Select(ctx, names, "The canonical identifier of the job to update")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have the canonical identifier of the job to update")
			}
			_, err = fmt.Sscan(args[0], &updateReq.JobId)
			if err != nil {
				return fmt.Errorf("invalid JOB_ID: %s", args[0])
			}
		}

		err = w.Jobs.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Jobs
