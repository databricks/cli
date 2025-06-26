// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package jobs

import (
	"fmt"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
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
		GroupID: "jobs",
		Annotations: map[string]string{
			"package": "jobs",
		},
		RunE: root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCancelAllRuns())
	cmd.AddCommand(newCancelRun())
	cmd.AddCommand(newCreate())
	cmd.AddCommand(newDelete())
	cmd.AddCommand(newDeleteRun())
	cmd.AddCommand(newExportRun())
	cmd.AddCommand(newGet())
	cmd.AddCommand(newGetPermissionLevels())
	cmd.AddCommand(newGetPermissions())
	cmd.AddCommand(newGetRun())
	cmd.AddCommand(newGetRunOutput())
	cmd.AddCommand(newList())
	cmd.AddCommand(newListRuns())
	cmd.AddCommand(newRepairRun())
	cmd.AddCommand(newReset())
	cmd.AddCommand(newRunNow())
	cmd.AddCommand(newSetPermissions())
	cmd.AddCommand(newSubmit())
	cmd.AddCommand(newUpdate())
	cmd.AddCommand(newUpdatePermissions())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start cancel-all-runs command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cancelAllRunsOverrides []func(
	*cobra.Command,
	*jobs.CancelAllRuns,
)

func newCancelAllRuns() *cobra.Command {
	cmd := &cobra.Command{}

	var cancelAllRunsReq jobs.CancelAllRuns
	var cancelAllRunsJson flags.JsonFlag

	cmd.Flags().Var(&cancelAllRunsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&cancelAllRunsReq.AllQueuedRuns, "all-queued-runs", cancelAllRunsReq.AllQueuedRuns, `Optional boolean parameter to cancel all queued runs.`)
	cmd.Flags().Int64Var(&cancelAllRunsReq.JobId, "job-id", cancelAllRunsReq.JobId, `The canonical identifier of the job to cancel all runs of.`)

	cmd.Use = "cancel-all-runs"
	cmd.Short = `Cancel all runs of a job.`
	cmd.Long = `Cancel all runs of a job.
  
  Cancels all active runs of a job. The runs are canceled asynchronously, so it
  doesn't prevent new runs from being started.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := cancelAllRunsJson.Unmarshal(&cancelAllRunsReq)
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

		err = w.Jobs.CancelAllRuns(ctx, cancelAllRunsReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range cancelAllRunsOverrides {
		fn(cmd, &cancelAllRunsReq)
	}

	return cmd
}

// start cancel-run command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cancelRunOverrides []func(
	*cobra.Command,
	*jobs.CancelRun,
)

func newCancelRun() *cobra.Command {
	cmd := &cobra.Command{}

	var cancelRunReq jobs.CancelRun
	var cancelRunJson flags.JsonFlag

	var cancelRunSkipWait bool
	var cancelRunTimeout time.Duration

	cmd.Flags().BoolVar(&cancelRunSkipWait, "no-wait", cancelRunSkipWait, `do not wait to reach TERMINATED or SKIPPED state`)
	cmd.Flags().DurationVar(&cancelRunTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach TERMINATED or SKIPPED state`)

	cmd.Flags().Var(&cancelRunJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "cancel-run RUN_ID"
	cmd.Short = `Cancel a run.`
	cmd.Long = `Cancel a run.
  
  Cancels a job run or a task run. The run is canceled asynchronously, so it may
  still be running when this request completes.

  Arguments:
    RUN_ID: This field is required.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'run_id' in your JSON input")
			}
			return nil
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := cancelRunJson.Unmarshal(&cancelRunReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
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
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range cancelRunOverrides {
		fn(cmd, &cancelRunReq)
	}

	return cmd
}

// start create command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOverrides []func(
	*cobra.Command,
	*jobs.CreateJob,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq jobs.CreateJob
	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create"
	cmd.Short = `Create a new job.`
	cmd.Long = `Create a new job.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

		response, err := w.Jobs.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createOverrides {
		fn(cmd, &createReq)
	}

	return cmd
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*jobs.DeleteJob,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq jobs.DeleteJob
	var deleteJson flags.JsonFlag

	cmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "delete JOB_ID"
	cmd.Short = `Delete a job.`
	cmd.Long = `Delete a job.
  
  Deletes a job.

  Arguments:
    JOB_ID: The canonical identifier of the job to delete. This field is required.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'job_id' in your JSON input")
			}
			return nil
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := deleteJson.Unmarshal(&deleteReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
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
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteOverrides {
		fn(cmd, &deleteReq)
	}

	return cmd
}

// start delete-run command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteRunOverrides []func(
	*cobra.Command,
	*jobs.DeleteRun,
)

func newDeleteRun() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteRunReq jobs.DeleteRun
	var deleteRunJson flags.JsonFlag

	cmd.Flags().Var(&deleteRunJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "delete-run RUN_ID"
	cmd.Short = `Delete a job run.`
	cmd.Long = `Delete a job run.
  
  Deletes a non-active run. Returns an error if the run is active.

  Arguments:
    RUN_ID: ID of the run to delete.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'run_id' in your JSON input")
			}
			return nil
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := deleteRunJson.Unmarshal(&deleteRunReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
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
				id, err := cmdio.Select(ctx, names, "ID of the run to delete")
				if err != nil {
					return err
				}
				args = append(args, id)
			}
			if len(args) != 1 {
				return fmt.Errorf("expected to have id of the run to delete")
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

// start export-run command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var exportRunOverrides []func(
	*cobra.Command,
	*jobs.ExportRunRequest,
)

func newExportRun() *cobra.Command {
	cmd := &cobra.Command{}

	var exportRunReq jobs.ExportRunRequest

	cmd.Flags().Var(&exportRunReq.ViewsToExport, "views-to-export", `Which views to export (CODE, DASHBOARDS, or ALL). Supported values: [ALL, CODE, DASHBOARDS]`)

	cmd.Use = "export-run RUN_ID"
	cmd.Short = `Export and retrieve a job run.`
	cmd.Long = `Export and retrieve a job run.
  
  Export and retrieve the job run task.

  Arguments:
    RUN_ID: The canonical identifier for the run. This field is required.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

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

		response, err := w.Jobs.ExportRun(ctx, exportRunReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range exportRunOverrides {
		fn(cmd, &exportRunReq)
	}

	return cmd
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*jobs.GetJobRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq jobs.GetJobRequest

	cmd.Flags().StringVar(&getReq.PageToken, "page-token", getReq.PageToken, `Use next_page_token returned from the previous GetJob response to request the next page of the job's array properties.`)

	cmd.Use = "get JOB_ID"
	cmd.Short = `Get a single job.`
	cmd.Long = `Get a single job.
  
  Retrieves the details for a single job.
  
  Large arrays in the results will be paginated when they exceed 100 elements. A
  request for a single job will return all properties for that job, and the
  first 100 elements of array properties (tasks, job_clusters,
  environments and parameters). Use the next_page_token field to check for
  more results and pass its value as the page_token in subsequent requests. If
  any array properties have more than 100 elements, additional results will be
  returned on subsequent requests. Arrays without additional results will be
  empty on later pages.

  Arguments:
    JOB_ID: The canonical identifier of the job to retrieve information about. This
      field is required.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

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

		response, err := w.Jobs.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getOverrides {
		fn(cmd, &getReq)
	}

	return cmd
}

// start get-permission-levels command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPermissionLevelsOverrides []func(
	*cobra.Command,
	*jobs.GetJobPermissionLevelsRequest,
)

func newGetPermissionLevels() *cobra.Command {
	cmd := &cobra.Command{}

	var getPermissionLevelsReq jobs.GetJobPermissionLevelsRequest

	cmd.Use = "get-permission-levels JOB_ID"
	cmd.Short = `Get job permission levels.`
	cmd.Long = `Get job permission levels.
  
  Gets the permission levels that a user can have on an object.

  Arguments:
    JOB_ID: The job for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No JOB_ID argument specified. Loading names for Jobs drop-down."
			names, err := w.Jobs.BaseJobSettingsNameToJobIdMap(ctx, jobs.ListJobsRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Jobs drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The job for which to get or manage permissions")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the job for which to get or manage permissions")
		}
		getPermissionLevelsReq.JobId = args[0]

		response, err := w.Jobs.GetPermissionLevels(ctx, getPermissionLevelsReq)
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

// start get-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getPermissionsOverrides []func(
	*cobra.Command,
	*jobs.GetJobPermissionsRequest,
)

func newGetPermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var getPermissionsReq jobs.GetJobPermissionsRequest

	cmd.Use = "get-permissions JOB_ID"
	cmd.Short = `Get job permissions.`
	cmd.Long = `Get job permissions.
  
  Gets the permissions of a job. Jobs can inherit permissions from their root
  object.

  Arguments:
    JOB_ID: The job for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No JOB_ID argument specified. Loading names for Jobs drop-down."
			names, err := w.Jobs.BaseJobSettingsNameToJobIdMap(ctx, jobs.ListJobsRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Jobs drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The job for which to get or manage permissions")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the job for which to get or manage permissions")
		}
		getPermissionsReq.JobId = args[0]

		response, err := w.Jobs.GetPermissions(ctx, getPermissionsReq)
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

// start get-run command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getRunOverrides []func(
	*cobra.Command,
	*jobs.GetRunRequest,
)

func newGetRun() *cobra.Command {
	cmd := &cobra.Command{}

	var getRunReq jobs.GetRunRequest

	cmd.Flags().BoolVar(&getRunReq.IncludeHistory, "include-history", getRunReq.IncludeHistory, `Whether to include the repair history in the response.`)
	cmd.Flags().BoolVar(&getRunReq.IncludeResolvedValues, "include-resolved-values", getRunReq.IncludeResolvedValues, `Whether to include resolved parameter values in the response.`)
	cmd.Flags().StringVar(&getRunReq.PageToken, "page-token", getRunReq.PageToken, `Use next_page_token returned from the previous GetRun response to request the next page of the run's array properties.`)

	cmd.Use = "get-run RUN_ID"
	cmd.Short = `Get a single job run.`
	cmd.Long = `Get a single job run.
  
  Retrieves the metadata of a run.
  
  Large arrays in the results will be paginated when they exceed 100 elements. A
  request for a single run will return all properties for that run, and the
  first 100 elements of array properties (tasks, job_clusters,
  job_parameters and repair_history). Use the next_page_token field to check
  for more results and pass its value as the page_token in subsequent requests.
  If any array properties have more than 100 elements, additional results will
  be returned on subsequent requests. Arrays without additional results will be
  empty on later pages.

  Arguments:
    RUN_ID: The canonical identifier of the run for which to retrieve the metadata.
      This field is required.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

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

		response, err := w.Jobs.GetRun(ctx, getRunReq)
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

// start get-run-output command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getRunOutputOverrides []func(
	*cobra.Command,
	*jobs.GetRunOutputRequest,
)

func newGetRunOutput() *cobra.Command {
	cmd := &cobra.Command{}

	var getRunOutputReq jobs.GetRunOutputRequest

	cmd.Use = "get-run-output RUN_ID"
	cmd.Short = `Get the output for a single run.`
	cmd.Long = `Get the output for a single run.
  
  Retrieve the output and metadata of a single task run. When a notebook task
  returns a value through the dbutils.notebook.exit() call, you can use this
  endpoint to retrieve that value. Databricks restricts this API to returning
  the first 5 MB of the output. To return a larger result, you can store job
  results in a cloud storage service.
  
  This endpoint validates that the __run_id__ parameter is valid and returns an
  HTTP status code 400 if the __run_id__ parameter is invalid. Runs are
  automatically removed after 60 days. If you to want to reference them beyond
  60 days, you must save old run results before they expire.

  Arguments:
    RUN_ID: The canonical identifier for the run.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

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

		response, err := w.Jobs.GetRunOutput(ctx, getRunOutputReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getRunOutputOverrides {
		fn(cmd, &getRunOutputReq)
	}

	return cmd
}

// start list command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listOverrides []func(
	*cobra.Command,
	*jobs.ListJobsRequest,
)

func newList() *cobra.Command {
	cmd := &cobra.Command{}

	var listReq jobs.ListJobsRequest

	cmd.Flags().BoolVar(&listReq.ExpandTasks, "expand-tasks", listReq.ExpandTasks, `Whether to include task and cluster details in the response.`)
	cmd.Flags().IntVar(&listReq.Limit, "limit", listReq.Limit, `The number of jobs to return.`)
	cmd.Flags().StringVar(&listReq.Name, "name", listReq.Name, `A filter on the list based on the exact (case insensitive) job name.`)
	cmd.Flags().IntVar(&listReq.Offset, "offset", listReq.Offset, `The offset of the first job to return, relative to the most recently created job.`)
	cmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Use next_page_token or prev_page_token returned from the previous request to list the next or previous page of jobs respectively.`)

	cmd.Use = "list"
	cmd.Short = `List jobs.`
	cmd.Long = `List jobs.
  
  Retrieves a list of jobs.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.Jobs.List(ctx, listReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listOverrides {
		fn(cmd, &listReq)
	}

	return cmd
}

// start list-runs command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listRunsOverrides []func(
	*cobra.Command,
	*jobs.ListRunsRequest,
)

func newListRuns() *cobra.Command {
	cmd := &cobra.Command{}

	var listRunsReq jobs.ListRunsRequest

	cmd.Flags().BoolVar(&listRunsReq.ActiveOnly, "active-only", listRunsReq.ActiveOnly, `If active_only is true, only active runs are included in the results; otherwise, lists both active and completed runs.`)
	cmd.Flags().BoolVar(&listRunsReq.CompletedOnly, "completed-only", listRunsReq.CompletedOnly, `If completed_only is true, only completed runs are included in the results; otherwise, lists both active and completed runs.`)
	cmd.Flags().BoolVar(&listRunsReq.ExpandTasks, "expand-tasks", listRunsReq.ExpandTasks, `Whether to include task and cluster details in the response.`)
	cmd.Flags().Int64Var(&listRunsReq.JobId, "job-id", listRunsReq.JobId, `The job for which to list runs.`)
	cmd.Flags().IntVar(&listRunsReq.Limit, "limit", listRunsReq.Limit, `The number of runs to return.`)
	cmd.Flags().IntVar(&listRunsReq.Offset, "offset", listRunsReq.Offset, `The offset of the first run to return, relative to the most recent run.`)
	cmd.Flags().StringVar(&listRunsReq.PageToken, "page-token", listRunsReq.PageToken, `Use next_page_token or prev_page_token returned from the previous request to list the next or previous page of runs respectively.`)
	cmd.Flags().Var(&listRunsReq.RunType, "run-type", `The type of runs to return. Supported values: [JOB_RUN, SUBMIT_RUN, WORKFLOW_RUN]`)
	cmd.Flags().Int64Var(&listRunsReq.StartTimeFrom, "start-time-from", listRunsReq.StartTimeFrom, `Show runs that started _at or after_ this value.`)
	cmd.Flags().Int64Var(&listRunsReq.StartTimeTo, "start-time-to", listRunsReq.StartTimeTo, `Show runs that started _at or before_ this value.`)

	cmd.Use = "list-runs"
	cmd.Short = `List job runs.`
	cmd.Long = `List job runs.
  
  List runs in descending order by start time.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.Jobs.ListRuns(ctx, listRunsReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listRunsOverrides {
		fn(cmd, &listRunsReq)
	}

	return cmd
}

// start repair-run command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var repairRunOverrides []func(
	*cobra.Command,
	*jobs.RepairRun,
)

func newRepairRun() *cobra.Command {
	cmd := &cobra.Command{}

	var repairRunReq jobs.RepairRun
	var repairRunJson flags.JsonFlag

	var repairRunSkipWait bool
	var repairRunTimeout time.Duration

	cmd.Flags().BoolVar(&repairRunSkipWait, "no-wait", repairRunSkipWait, `do not wait to reach TERMINATED or SKIPPED state`)
	cmd.Flags().DurationVar(&repairRunTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach TERMINATED or SKIPPED state`)

	cmd.Flags().Var(&repairRunJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: dbt_commands
	// TODO: array: jar_params
	// TODO: map via StringToStringVar: job_parameters
	cmd.Flags().Int64Var(&repairRunReq.LatestRepairId, "latest-repair-id", repairRunReq.LatestRepairId, `The ID of the latest repair.`)
	// TODO: map via StringToStringVar: notebook_params
	cmd.Flags().Var(&repairRunReq.PerformanceTarget, "performance-target", `The performance mode on a serverless job. Supported values: [PERFORMANCE_OPTIMIZED, STANDARD]`)
	// TODO: complex arg: pipeline_params
	// TODO: map via StringToStringVar: python_named_params
	// TODO: array: python_params
	cmd.Flags().BoolVar(&repairRunReq.RerunAllFailedTasks, "rerun-all-failed-tasks", repairRunReq.RerunAllFailedTasks, `If true, repair all failed tasks.`)
	cmd.Flags().BoolVar(&repairRunReq.RerunDependentTasks, "rerun-dependent-tasks", repairRunReq.RerunDependentTasks, `If true, repair all tasks that depend on the tasks in rerun_tasks, even if they were previously successful.`)
	// TODO: array: rerun_tasks
	// TODO: array: spark_submit_params
	// TODO: map via StringToStringVar: sql_params

	cmd.Use = "repair-run RUN_ID"
	cmd.Short = `Repair a job run.`
	cmd.Long = `Repair a job run.
  
  Re-run one or more tasks. Tasks are re-run as part of the original job run.
  They use the current job and task settings, and can be viewed in the history
  for the original job run.

  Arguments:
    RUN_ID: The job run ID of the run to repair. The run must not be in progress.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'run_id' in your JSON input")
			}
			return nil
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := repairRunJson.Unmarshal(&repairRunReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
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
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range repairRunOverrides {
		fn(cmd, &repairRunReq)
	}

	return cmd
}

// start reset command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var resetOverrides []func(
	*cobra.Command,
	*jobs.ResetJob,
)

func newReset() *cobra.Command {
	cmd := &cobra.Command{}

	var resetReq jobs.ResetJob
	var resetJson flags.JsonFlag

	cmd.Flags().Var(&resetJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "reset"
	cmd.Short = `Update all job settings (reset).`
	cmd.Long = `Update all job settings (reset).
  
  Overwrite all settings for the given job. Use the [_Update_
  endpoint](:method:jobs/update) to update job settings partially.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := resetJson.Unmarshal(&resetReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

		err = w.Jobs.Reset(ctx, resetReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range resetOverrides {
		fn(cmd, &resetReq)
	}

	return cmd
}

// start run-now command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var runNowOverrides []func(
	*cobra.Command,
	*jobs.RunNow,
)

func newRunNow() *cobra.Command {
	cmd := &cobra.Command{}

	var runNowReq jobs.RunNow
	var runNowJson flags.JsonFlag

	var runNowSkipWait bool
	var runNowTimeout time.Duration

	cmd.Flags().BoolVar(&runNowSkipWait, "no-wait", runNowSkipWait, `do not wait to reach TERMINATED or SKIPPED state`)
	cmd.Flags().DurationVar(&runNowTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach TERMINATED or SKIPPED state`)

	cmd.Flags().Var(&runNowJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: dbt_commands
	cmd.Flags().StringVar(&runNowReq.IdempotencyToken, "idempotency-token", runNowReq.IdempotencyToken, `An optional token to guarantee the idempotency of job run requests.`)
	// TODO: array: jar_params
	// TODO: map via StringToStringVar: job_parameters
	// TODO: map via StringToStringVar: notebook_params
	// TODO: array: only
	cmd.Flags().Var(&runNowReq.PerformanceTarget, "performance-target", `The performance mode on a serverless job. Supported values: [PERFORMANCE_OPTIMIZED, STANDARD]`)
	// TODO: complex arg: pipeline_params
	// TODO: map via StringToStringVar: python_named_params
	// TODO: array: python_params
	// TODO: complex arg: queue
	// TODO: array: spark_submit_params
	// TODO: map via StringToStringVar: sql_params

	cmd.Use = "run-now JOB_ID"
	cmd.Short = `Trigger a new job run.`
	cmd.Long = `Trigger a new job run.
  
  Run a job and return the run_id of the triggered run.

  Arguments:
    JOB_ID: The ID of the job to be executed`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'job_id' in your JSON input")
			}
			return nil
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := runNowJson.Unmarshal(&runNowReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
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
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range runNowOverrides {
		fn(cmd, &runNowReq)
	}

	return cmd
}

// start set-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var setPermissionsOverrides []func(
	*cobra.Command,
	*jobs.JobPermissionsRequest,
)

func newSetPermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var setPermissionsReq jobs.JobPermissionsRequest
	var setPermissionsJson flags.JsonFlag

	cmd.Flags().Var(&setPermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "set-permissions JOB_ID"
	cmd.Short = `Set job permissions.`
	cmd.Long = `Set job permissions.
  
  Sets permissions on an object, replacing existing permissions if they exist.
  Deletes all direct permissions if none are specified. Objects can inherit
  permissions from their root object.

  Arguments:
    JOB_ID: The job for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := setPermissionsJson.Unmarshal(&setPermissionsReq)
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
		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No JOB_ID argument specified. Loading names for Jobs drop-down."
			names, err := w.Jobs.BaseJobSettingsNameToJobIdMap(ctx, jobs.ListJobsRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Jobs drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The job for which to get or manage permissions")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the job for which to get or manage permissions")
		}
		setPermissionsReq.JobId = args[0]

		response, err := w.Jobs.SetPermissions(ctx, setPermissionsReq)
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

// start submit command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var submitOverrides []func(
	*cobra.Command,
	*jobs.SubmitRun,
)

func newSubmit() *cobra.Command {
	cmd := &cobra.Command{}

	var submitReq jobs.SubmitRun
	var submitJson flags.JsonFlag

	var submitSkipWait bool
	var submitTimeout time.Duration

	cmd.Flags().BoolVar(&submitSkipWait, "no-wait", submitSkipWait, `do not wait to reach TERMINATED or SKIPPED state`)
	cmd.Flags().DurationVar(&submitTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach TERMINATED or SKIPPED state`)

	cmd.Flags().Var(&submitJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list
	cmd.Flags().StringVar(&submitReq.BudgetPolicyId, "budget-policy-id", submitReq.BudgetPolicyId, `The user specified id of the budget policy to use for this one-time run.`)
	// TODO: complex arg: email_notifications
	// TODO: array: environments
	// TODO: complex arg: git_source
	// TODO: complex arg: health
	cmd.Flags().StringVar(&submitReq.IdempotencyToken, "idempotency-token", submitReq.IdempotencyToken, `An optional token that can be used to guarantee the idempotency of job run requests.`)
	// TODO: complex arg: notification_settings
	// TODO: complex arg: queue
	// TODO: complex arg: run_as
	cmd.Flags().StringVar(&submitReq.RunName, "run-name", submitReq.RunName, `An optional name for the run.`)
	// TODO: array: tasks
	cmd.Flags().IntVar(&submitReq.TimeoutSeconds, "timeout-seconds", submitReq.TimeoutSeconds, `An optional timeout applied to each run of this job.`)
	// TODO: complex arg: webhook_notifications

	cmd.Use = "submit"
	cmd.Short = `Create and trigger a one-time run.`
	cmd.Long = `Create and trigger a one-time run.
  
  Submit a one-time run. This endpoint allows you to submit a workload directly
  without creating a job. Runs submitted using this endpoint don’t display in
  the UI. Use the jobs/runs/get API to check the run state after the job is
  submitted.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := submitJson.Unmarshal(&submitReq)
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
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range submitOverrides {
		fn(cmd, &submitReq)
	}

	return cmd
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*jobs.UpdateJob,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq jobs.UpdateJob
	var updateJson flags.JsonFlag

	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: fields_to_remove
	// TODO: complex arg: new_settings

	cmd.Use = "update JOB_ID"
	cmd.Short = `Update job settings partially.`
	cmd.Long = `Update job settings partially.
  
  Add, update, or remove specific settings of an existing job. Use the [_Reset_
  endpoint](:method:jobs/reset) to overwrite all job settings.

  Arguments:
    JOB_ID: The canonical identifier of the job to update. This field is required.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'job_id' in your JSON input")
			}
			return nil
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnosticsToErrorOut(ctx, diags)
				if err != nil {
					return err
				}
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
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateOverrides {
		fn(cmd, &updateReq)
	}

	return cmd
}

// start update-permissions command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updatePermissionsOverrides []func(
	*cobra.Command,
	*jobs.JobPermissionsRequest,
)

func newUpdatePermissions() *cobra.Command {
	cmd := &cobra.Command{}

	var updatePermissionsReq jobs.JobPermissionsRequest
	var updatePermissionsJson flags.JsonFlag

	cmd.Flags().Var(&updatePermissionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list

	cmd.Use = "update-permissions JOB_ID"
	cmd.Short = `Update job permissions.`
	cmd.Long = `Update job permissions.
  
  Updates the permissions on a job. Jobs can inherit permissions from their root
  object.

  Arguments:
    JOB_ID: The job for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updatePermissionsJson.Unmarshal(&updatePermissionsReq)
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
		if len(args) == 0 {
			promptSpinner := cmdio.Spinner(ctx)
			promptSpinner <- "No JOB_ID argument specified. Loading names for Jobs drop-down."
			names, err := w.Jobs.BaseJobSettingsNameToJobIdMap(ctx, jobs.ListJobsRequest{})
			close(promptSpinner)
			if err != nil {
				return fmt.Errorf("failed to load names for Jobs drop-down. Please manually specify required arguments. Original error: %w", err)
			}
			id, err := cmdio.Select(ctx, names, "The job for which to get or manage permissions")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the job for which to get or manage permissions")
		}
		updatePermissionsReq.JobId = args[0]

		response, err := w.Jobs.UpdatePermissions(ctx, updatePermissionsReq)
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

// end service Jobs
