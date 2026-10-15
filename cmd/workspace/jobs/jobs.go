// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package jobs

import (
	"errors"
	"fmt"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgen"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/sdk-go/options/lro"
	"github.com/spf13/cobra"

	accessmanagement "github.com/databricks/cli/libs/sdktypes/accessmanagement/v1"
	jobs "github.com/databricks/cli/libs/sdktypes/jobs/v2"
	sdkaccessmanagement "github.com/databricks/sdk-go/accessmanagement/v1"
	sdkjobs "github.com/databricks/sdk-go/jobs/v2"
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
  tasks using notebooks, JARS, Spark Declarative Pipelines, or Python, Scala,
  Spark submit, and Java applications.

  You should never hard code secrets or store them in plain text. Use the
  [Secrets CLI] to manage secrets in the [Databricks CLI]. Use the [Secrets
  utility] to reference secrets in notebooks and jobs.

  [Databricks CLI]: https://docs.databricks.com/dev-tools/cli/index.html
  [Secrets CLI]: https://docs.databricks.com/dev-tools/cli/secrets-cli.html
  [Secrets utility]: https://docs.databricks.com/dev-tools/databricks-utils.html#dbutils-secrets`,
		GroupID: "lakeflow",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
	*jobs.CancelAllRunsRequest,
)

func newCancelAllRuns() *cobra.Command {
	cmd := &cobra.Command{}
	var cancelAllRunsReq jobs.CancelAllRunsRequest

	var cancelAllRunsJson flags.JsonFlag

	cmd.Flags().Var(&cancelAllRunsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	flags.BoolPtrVar(cmd.Flags(), &cancelAllRunsReq.AllQueuedRuns, "all-queued-runs", `Optional boolean parameter to cancel all queued runs.`)
	flags.Int64PtrVar(cmd.Flags(), &cancelAllRunsReq.JobId, "job-id", `The canonical identifier of the job to cancel all runs of.`)

	cmd.Use = "cancel-all-runs"
	cmd.Short = `Cancel all runs of a job.`
	cmd.Long = `Cancel all runs of a job.

  Cancels all active runs of a job. The runs are canceled asynchronously, so it
  doesn't prevent new runs from being started.`

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
			diags := cancelAllRunsJson.Unmarshal(&cancelAllRunsReq)
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

		sdkReq, err := jobs.CancelAllRunsRequestToSDK(&cancelAllRunsReq)
		if err != nil {
			return err
		}
		response, err := w.Jobs.CancelAllRuns(ctx, *sdkReq)
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
	*jobs.CancelRunRequest,
)

func newCancelRun() *cobra.Command {
	cmd := &cobra.Command{}
	var cancelRunReq jobs.CancelRunRequest

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
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := cancelRunJson.Unmarshal(&cancelRunReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			if len(args) == 0 {
				sp := cmdio.NewSpinner(ctx)
				sp.Update("No RUN_ID argument specified. Loading names for Jobs drop-down.")
				names, err := cmdgen.NamedIdMap(ctx, w.Jobs.ListJobsIter(ctx, sdkjobs.ListJobsRequest{}), []string{"Settings", "Name"}, []string{"JobId"})
				sp.Close()
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
				return errors.New("expected to have this field is required")
			}
			var cancelRunRunIdArg int64
			_, err = fmt.Sscan(args[0], &cancelRunRunIdArg)
			if err != nil {
				return fmt.Errorf("invalid RUN_ID: %s", args[0])
			}
			cancelRunReq.RunId = &cancelRunRunIdArg
		}

		sdkReq, err := jobs.CancelRunRequestToSDK(&cancelRunReq)
		if err != nil {
			return err
		}
		wait, err := w.Jobs.CancelRun(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if cancelRunSkipWait {
			return cmdio.Render(ctx, map[string]any{"run_id": wait.GetRunId()})
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for cancel-run to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(cancelRunTimeout))
		sp.Close()
		if err != nil {
			return err
		}

		out, err := jobs.GetRunResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*jobs.CreateJobRequest,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}
	var createReq jobs.CreateJobRequest

	var createJson flags.JsonFlag

	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create"
	cmd.Short = `Create a new job.`
	cmd.Long = `Create a new job.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createJson.Unmarshal(&createReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return errors.New("please provide command input in JSON format by specifying the --json flag")
		}

		sdkReq, err := jobs.CreateJobRequestToSDK(&createReq)
		if err != nil {
			return err
		}
		response, err := w.Jobs.CreateJob(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := jobs.CreateJobResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*jobs.DeleteJobRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteReq jobs.DeleteJobRequest

	var deleteJson flags.JsonFlag

	cmd.Flags().Var(&deleteJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "delete JOB_ID"
	cmd.Short = `Delete a job.`
	cmd.Long = `Delete a job.

  Deletes a job.

  Arguments:
    JOB_ID: The canonical identifier of the job to delete. This field is required.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'job_id' in your JSON input")
			}
			return nil
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := deleteJson.Unmarshal(&deleteReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			if len(args) == 0 {
				sp := cmdio.NewSpinner(ctx)
				sp.Update("No JOB_ID argument specified. Loading names for Jobs drop-down.")
				names, err := cmdgen.NamedIdMap(ctx, w.Jobs.ListJobsIter(ctx, sdkjobs.ListJobsRequest{}), []string{"Settings", "Name"}, []string{"JobId"})
				sp.Close()
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
				return errors.New("expected to have the canonical identifier of the job to delete")
			}
			var deleteJobIdArg int64
			_, err = fmt.Sscan(args[0], &deleteJobIdArg)
			if err != nil {
				return fmt.Errorf("invalid JOB_ID: %s", args[0])
			}
			deleteReq.JobId = &deleteJobIdArg
		}

		sdkReq, err := jobs.DeleteJobRequestToSDK(&deleteReq)
		if err != nil {
			return err
		}
		response, err := w.Jobs.DeleteJob(ctx, *sdkReq)
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
	*jobs.DeleteRunRequest,
)

func newDeleteRun() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteRunReq jobs.DeleteRunRequest

	var deleteRunJson flags.JsonFlag

	cmd.Flags().Var(&deleteRunJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "delete-run RUN_ID"
	cmd.Short = `Delete a job run.`
	cmd.Long = `Delete a job run.

  Deletes a non-active run. Returns an error if the run is active.

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
		return nil
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
		} else {
			if len(args) == 0 {
				sp := cmdio.NewSpinner(ctx)
				sp.Update("No RUN_ID argument specified. Loading names for Jobs drop-down.")
				names, err := cmdgen.NamedIdMap(ctx, w.Jobs.ListJobsIter(ctx, sdkjobs.ListJobsRequest{}), []string{"Settings", "Name"}, []string{"JobId"})
				sp.Close()
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
				return errors.New("expected to have id of the run to delete")
			}
			var deleteRunRunIdArg int64
			_, err = fmt.Sscan(args[0], &deleteRunRunIdArg)
			if err != nil {
				return fmt.Errorf("invalid RUN_ID: %s", args[0])
			}
			deleteRunReq.RunId = &deleteRunRunIdArg
		}

		sdkReq, err := jobs.DeleteRunRequestToSDK(&deleteRunReq)
		if err != nil {
			return err
		}
		response, err := w.Jobs.DeleteRun(ctx, *sdkReq)
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

	flags.EnumVar(cmd.Flags(), &exportRunReq.ViewsToExport, "views-to-export", `Which views to export (CODE, DASHBOARDS, or ALL). Supported values: [ALL, CODE, DASHBOARDS]`, "ViewsToExport", "ALL", "CODE", "DASHBOARDS")

	cmd.Use = "export-run RUN_ID"
	cmd.Short = `Export and retrieve a job run.`
	cmd.Long = `Export and retrieve a job run.

  Export and retrieve the job run task.

  Arguments:
    RUN_ID: The canonical identifier for the run. This field is required.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No RUN_ID argument specified. Loading names for Jobs drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Jobs.ListJobsIter(ctx, sdkjobs.ListJobsRequest{}), []string{"Settings", "Name"}, []string{"JobId"})
			sp.Close()
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
			return errors.New("expected to have the canonical identifier for the run")
		}
		var exportRunRunIdArg int64
		_, err = fmt.Sscan(args[0], &exportRunRunIdArg)
		if err != nil {
			return fmt.Errorf("invalid RUN_ID: %s", args[0])
		}
		exportRunReq.RunId = &exportRunRunIdArg

		sdkReq, err := jobs.ExportRunRequestToSDK(&exportRunReq)
		if err != nil {
			return err
		}
		response, err := w.Jobs.ExportRun(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := jobs.ExportRunResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

	flags.BoolPtrVar(cmd.Flags(), &getReq.IncludeTriggerState, "include-trigger-state", `Flag that indicates that trigger state should be included in the response.`)
	flags.StringPtrVar(cmd.Flags(), &getReq.PageToken, "page-token", `Use next_page_token returned from the previous GetJob response to request the next page of the job's array properties.`)

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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No JOB_ID argument specified. Loading names for Jobs drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Jobs.ListJobsIter(ctx, sdkjobs.ListJobsRequest{}), []string{"Settings", "Name"}, []string{"JobId"})
			sp.Close()
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
			return errors.New("expected to have the canonical identifier of the job to retrieve information about")
		}
		var getJobIdArg int64
		_, err = fmt.Sscan(args[0], &getJobIdArg)
		if err != nil {
			return fmt.Errorf("invalid JOB_ID: %s", args[0])
		}
		getReq.JobId = &getJobIdArg

		sdkReq, err := jobs.GetJobRequestToSDK(&getReq)
		if err != nil {
			return err
		}
		response, err := w.Jobs.GetJob(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := jobs.GetJobResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*sdkaccessmanagement.ListPermissionLevelsRequest,
)

func newGetPermissionLevels() *cobra.Command {
	cmd := &cobra.Command{}
	var getPermissionLevelsReq sdkaccessmanagement.ListPermissionLevelsRequest

	cmd.Use = "get-permission-levels JOB_ID"
	cmd.Short = `Get job permission levels.`
	cmd.Long = `Get job permission levels.

  Gets the permission levels that a user can have on an object.

  Arguments:
    JOB_ID: The job for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No JOB_ID argument specified. Loading names for Jobs drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Jobs.ListJobsIter(ctx, sdkjobs.ListJobsRequest{}), []string{"Settings", "Name"}, []string{"JobId"})
			sp.Close()
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
			return errors.New("expected to have the job for which to get or manage permissions")
		}
		_ = args[0] // TODO: field not in sdk-go request: job_id

		getPermissionLevelsReq.RequestObjectType = new("jobs")

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

	cmd.Use = "get-permissions JOB_ID"
	cmd.Short = `Get job permissions.`
	cmd.Long = `Get job permissions.

  Gets the permissions of a job. Jobs can inherit permissions from their root
  object.

  Arguments:
    JOB_ID: The job for which to get or manage permissions.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No JOB_ID argument specified. Loading names for Jobs drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Jobs.ListJobsIter(ctx, sdkjobs.ListJobsRequest{}), []string{"Settings", "Name"}, []string{"JobId"})
			sp.Close()
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
			return errors.New("expected to have the job for which to get or manage permissions")
		}
		_ = args[0] // TODO: field not in sdk-go request: job_id

		getPermissionsReq.RequestObjectType = new("jobs")

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
	*jobs.GetRunRequest,
)

func newGetRun() *cobra.Command {
	cmd := &cobra.Command{}
	var getRunReq jobs.GetRunRequest

	flags.BoolPtrVar(cmd.Flags(), &getRunReq.IncludeHistory, "include-history", `Whether to include the repair history in the response.`)
	flags.BoolPtrVar(cmd.Flags(), &getRunReq.IncludeResolvedValues, "include-resolved-values", `Whether to include resolved parameter values in the response.`)
	flags.StringPtrVar(cmd.Flags(), &getRunReq.PageToken, "page-token", `Use next_page_token returned from the previous GetRun response to request the next page of the run's array properties.`)

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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No RUN_ID argument specified. Loading names for Jobs drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Jobs.ListJobsIter(ctx, sdkjobs.ListJobsRequest{}), []string{"Settings", "Name"}, []string{"JobId"})
			sp.Close()
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
			return errors.New("expected to have the canonical identifier of the run for which to retrieve the metadata")
		}
		var getRunRunIdArg int64
		_, err = fmt.Sscan(args[0], &getRunRunIdArg)
		if err != nil {
			return fmt.Errorf("invalid RUN_ID: %s", args[0])
		}
		getRunReq.RunId = &getRunRunIdArg

		sdkReq, err := jobs.GetRunRequestToSDK(&getRunReq)
		if err != nil {
			return err
		}
		response, err := w.Jobs.GetRun(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := jobs.GetRunResponseFromSDK(response)
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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No RUN_ID argument specified. Loading names for Jobs drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Jobs.ListJobsIter(ctx, sdkjobs.ListJobsRequest{}), []string{"Settings", "Name"}, []string{"JobId"})
			sp.Close()
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
			return errors.New("expected to have the canonical identifier for the run")
		}
		var getRunOutputRunIdArg int64
		_, err = fmt.Sscan(args[0], &getRunOutputRunIdArg)
		if err != nil {
			return fmt.Errorf("invalid RUN_ID: %s", args[0])
		}
		getRunOutputReq.RunId = &getRunOutputRunIdArg

		sdkReq, err := jobs.GetRunOutputRequestToSDK(&getRunOutputReq)
		if err != nil {
			return err
		}
		response, err := w.Jobs.GetRunOutput(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := jobs.GetRunOutputResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listLimit int

	flags.BoolPtrVar(cmd.Flags(), &listReq.ExpandTasks, "expand-tasks", `Whether to include task and cluster details in the response.`)
	flags.IntPtrVar(cmd.Flags(), &listReq.Limit, "page-size", `The number of jobs to return.`)
	cmd.Flags().Lookup("page-size").Hidden = true
	flags.StringPtrVar(cmd.Flags(), &listReq.Name, "name", `A filter on the list based on the exact (case insensitive) job name.`)
	flags.IntPtrVar(cmd.Flags(), &listReq.Offset, "offset", `The offset of the first job to return, relative to the most recently created job.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list"
	cmd.Short = `List jobs.`
	cmd.Long = `List jobs.

  Retrieves a list of jobs.`

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

		sdkReq, err := jobs.ListJobsRequestToSDK(&listReq)
		if err != nil {
			return err
		}
		if listLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listLimit)
		}
		if listLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listLimit)
		}
		response := w.Jobs.ListJobsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, jobs.BaseJobFromSDK))
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

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listRunsLimit int

	flags.BoolPtrVar(cmd.Flags(), &listRunsReq.ActiveOnly, "active-only", `If active_only is true, only active runs are included in the results; otherwise, lists both active and completed runs.`)
	flags.BoolPtrVar(cmd.Flags(), &listRunsReq.CompletedOnly, "completed-only", `If completed_only is true, only completed runs are included in the results; otherwise, lists both active and completed runs.`)
	flags.BoolPtrVar(cmd.Flags(), &listRunsReq.ExpandTasks, "expand-tasks", `Whether to include task and cluster details in the response.`)
	flags.Int64PtrVar(cmd.Flags(), &listRunsReq.JobId, "job-id", `The job for which to list runs.`)
	flags.IntPtrVar(cmd.Flags(), &listRunsReq.Limit, "page-size", `The number of runs to return.`)
	cmd.Flags().Lookup("page-size").Hidden = true
	flags.IntPtrVar(cmd.Flags(), &listRunsReq.Offset, "offset", `The offset of the first run to return, relative to the most recent run.`)
	flags.EnumVar(cmd.Flags(), &listRunsReq.RunType, "run-type", `The type of runs to return. Supported values: [JOB_RUN, SUBMIT_RUN, WORKFLOW_RUN]`, "RunType", "JOB_RUN", "SUBMIT_RUN", "WORKFLOW_RUN")
	flags.Int64PtrVar(cmd.Flags(), &listRunsReq.StartTimeFrom, "start-time-from", `Show runs that started _at or after_ this value.`)
	flags.Int64PtrVar(cmd.Flags(), &listRunsReq.StartTimeTo, "start-time-to", `Show runs that started _at or before_ this value.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listRunsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	flags.StringPtrVar(cmd.Flags(), &listRunsReq.PageToken, "page-token", `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-runs"
	cmd.Short = `List job runs.`
	cmd.Long = `List job runs.

  List runs in descending order by end time. If a run has not finished, it falls
  back to start time.`

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

		sdkReq, err := jobs.ListRunsRequestToSDK(&listRunsReq)
		if err != nil {
			return err
		}
		if listRunsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listRunsLimit)
		}
		if listRunsLimit > 0 {
			ctx = cmdio.WithLimit(ctx, listRunsLimit)
		}
		response := w.Jobs.ListRunsIter(ctx, *sdkReq)
		return cmdio.RenderIterator(ctx, cmdgen.MapSeq(response, jobs.BaseRunFromSDK))
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
	*jobs.RepairRunRequest,
)

func newRepairRun() *cobra.Command {
	cmd := &cobra.Command{}
	var repairRunReq jobs.RepairRunRequest

	var repairRunJson flags.JsonFlag

	var repairRunSkipWait bool
	var repairRunTimeout time.Duration

	cmd.Flags().BoolVar(&repairRunSkipWait, "no-wait", repairRunSkipWait, `do not wait to reach TERMINATED or SKIPPED state`)
	cmd.Flags().DurationVar(&repairRunTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach TERMINATED or SKIPPED state`)

	cmd.Flags().Var(&repairRunJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: dbt_commands
	// TODO: array: jar_params
	// TODO: map via StringToStringVar: job_parameters
	flags.Int64PtrVar(cmd.Flags(), &repairRunReq.LatestRepairId, "latest-repair-id", `The ID of the latest repair.`)
	// TODO: map via StringToStringVar: notebook_params
	flags.EnumVar(cmd.Flags(), &repairRunReq.PerformanceTarget, "performance-target", `The performance mode on a serverless job. Supported values: [PERFORMANCE_OPTIMIZED, STANDARD]`, "PerformanceTarget", "PERFORMANCE_OPTIMIZED", "STANDARD")
	// TODO: complex arg: pipeline_params
	// TODO: map via StringToStringVar: python_named_params
	// TODO: array: python_params
	flags.BoolPtrVar(cmd.Flags(), &repairRunReq.RerunAllFailedTasks, "rerun-all-failed-tasks", `If true, repair all failed tasks.`)
	flags.BoolPtrVar(cmd.Flags(), &repairRunReq.RerunDependentTasks, "rerun-dependent-tasks", `If true, repair all tasks that depend on the tasks in rerun_tasks, even if they were previously successful.`)
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
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := repairRunJson.Unmarshal(&repairRunReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			if len(args) == 0 {
				sp := cmdio.NewSpinner(ctx)
				sp.Update("No RUN_ID argument specified. Loading names for Jobs drop-down.")
				names, err := cmdgen.NamedIdMap(ctx, w.Jobs.ListJobsIter(ctx, sdkjobs.ListJobsRequest{}), []string{"Settings", "Name"}, []string{"JobId"})
				sp.Close()
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
				return errors.New("expected to have the job run id of the run to repair")
			}
			var repairRunRunIdArg int64
			_, err = fmt.Sscan(args[0], &repairRunRunIdArg)
			if err != nil {
				return fmt.Errorf("invalid RUN_ID: %s", args[0])
			}
			repairRunReq.RunId = &repairRunRunIdArg

		}

		sdkReq, err := jobs.RepairRunRequestToSDK(&repairRunReq)
		if err != nil {
			return err
		}
		wait, err := w.Jobs.Repair(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if repairRunSkipWait {
			return cmdio.Render(ctx, map[string]any{"run_id": wait.GetRunId()})
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for repair-run to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(repairRunTimeout))
		sp.Close()
		if err != nil {
			return err
		}

		out, err := jobs.GetRunResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*jobs.ResetJobRequest,
)

func newReset() *cobra.Command {
	cmd := &cobra.Command{}
	var resetReq jobs.ResetJobRequest

	var resetJson flags.JsonFlag

	cmd.Flags().Var(&resetJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "reset"
	cmd.Short = `Update all job settings (reset).`
	cmd.Long = `Update all job settings (reset).

  Overwrite all settings for the given job. Use the [_Update_
  endpoint](:method:jobs/update) to update job settings partially.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := resetJson.Unmarshal(&resetReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			return errors.New("please provide command input in JSON format by specifying the --json flag")
		}

		sdkReq, err := jobs.ResetJobRequestToSDK(&resetReq)
		if err != nil {
			return err
		}
		response, err := w.Jobs.ResetJob(ctx, *sdkReq)
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
	*jobs.RunNowRequest,
)

func newRunNow() *cobra.Command {
	cmd := &cobra.Command{}
	var runNowReq jobs.RunNowRequest

	var runNowJson flags.JsonFlag

	var runNowSkipWait bool
	var runNowTimeout time.Duration

	cmd.Flags().BoolVar(&runNowSkipWait, "no-wait", runNowSkipWait, `do not wait to reach TERMINATED or SKIPPED state`)
	cmd.Flags().DurationVar(&runNowTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach TERMINATED or SKIPPED state`)

	cmd.Flags().Var(&runNowJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: dbt_commands
	flags.StringPtrVar(cmd.Flags(), &runNowReq.IdempotencyToken, "idempotency-token", `An optional token to guarantee the idempotency of job run requests.`)
	// TODO: array: jar_params
	// TODO: map via StringToStringVar: job_parameters
	// TODO: map via StringToStringVar: notebook_params
	// TODO: array: only
	flags.EnumVar(cmd.Flags(), &runNowReq.PerformanceTarget, "performance-target", `The performance mode on a serverless job. Supported values: [PERFORMANCE_OPTIMIZED, STANDARD]`, "PerformanceTarget", "PERFORMANCE_OPTIMIZED", "STANDARD")
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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'job_id' in your JSON input")
			}
			return nil
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := runNowJson.Unmarshal(&runNowReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			if len(args) == 0 {
				sp := cmdio.NewSpinner(ctx)
				sp.Update("No JOB_ID argument specified. Loading names for Jobs drop-down.")
				names, err := cmdgen.NamedIdMap(ctx, w.Jobs.ListJobsIter(ctx, sdkjobs.ListJobsRequest{}), []string{"Settings", "Name"}, []string{"JobId"})
				sp.Close()
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
				return errors.New("expected to have the id of the job to be executed")
			}
			var runNowJobIdArg int64
			_, err = fmt.Sscan(args[0], &runNowJobIdArg)
			if err != nil {
				return fmt.Errorf("invalid JOB_ID: %s", args[0])
			}
			runNowReq.JobId = &runNowJobIdArg

		}

		sdkReq, err := jobs.RunNowRequestToSDK(&runNowReq)
		if err != nil {
			return err
		}
		wait, err := w.Jobs.RunNow(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if runNowSkipWait {
			return cmdio.Render(ctx, map[string]any{"run_id": wait.GetRunId()})
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for run-now to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(runNowTimeout))
		sp.Close()
		if err != nil {
			return err
		}

		out, err := jobs.GetRunResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*accessmanagement.SetObjectPermissionsRequest,
)

func newSetPermissions() *cobra.Command {
	cmd := &cobra.Command{}
	var setPermissionsReq accessmanagement.SetObjectPermissionsRequest

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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No JOB_ID argument specified. Loading names for Jobs drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Jobs.ListJobsIter(ctx, sdkjobs.ListJobsRequest{}), []string{"Settings", "Name"}, []string{"JobId"})
			sp.Close()
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
			return errors.New("expected to have the job for which to get or manage permissions")
		}
		_ = args[0] // TODO: field not in sdk-go request: job_id

		setPermissionsReq.RequestObjectType = new("jobs")

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

// start submit command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var submitOverrides []func(
	*cobra.Command,
	*jobs.SubmitRunRequest,
)

func newSubmit() *cobra.Command {
	cmd := &cobra.Command{}
	var submitReq jobs.SubmitRunRequest

	var submitJson flags.JsonFlag

	var submitSkipWait bool
	var submitTimeout time.Duration

	cmd.Flags().BoolVar(&submitSkipWait, "no-wait", submitSkipWait, `do not wait to reach TERMINATED or SKIPPED state`)
	cmd.Flags().DurationVar(&submitTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach TERMINATED or SKIPPED state`)

	cmd.Flags().Var(&submitJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: access_control_list
	flags.StringPtrVar(cmd.Flags(), &submitReq.BudgetPolicyId, "budget-policy-id", `The user specified id of the budget policy to use for this one-time run.`)
	// TODO: complex arg: email_notifications
	// TODO: array: environment_variables
	// TODO: array: environments
	// TODO: complex arg: git_source
	// TODO: complex arg: health
	flags.StringPtrVar(cmd.Flags(), &submitReq.IdempotencyToken, "idempotency-token", `An optional token that can be used to guarantee the idempotency of job run requests.`)
	// TODO: complex arg: notification_settings
	flags.EnumVar(cmd.Flags(), &submitReq.PerformanceTarget, "performance-target", `The performance mode on a serverless one-time run. Supported values: [PERFORMANCE_OPTIMIZED, STANDARD]`, "PerformanceTarget", "PERFORMANCE_OPTIMIZED", "STANDARD")
	// TODO: complex arg: queue
	// TODO: complex arg: run_as
	flags.StringPtrVar(cmd.Flags(), &submitReq.RunName, "run-name", `An optional name for the run.`)
	// TODO: array: tasks
	flags.IntPtrVar(cmd.Flags(), &submitReq.TimeoutSeconds, "timeout-seconds", `An optional timeout applied to each run of this job.`)
	flags.StringPtrVar(cmd.Flags(), &submitReq.UsagePolicyId, "usage-policy-id", `The user specified id of the usage policy to use for this one-time run.`)
	// TODO: complex arg: webhook_notifications

	cmd.Use = "submit"
	cmd.Short = `Create and trigger a one-time run.`
	cmd.Long = `Create and trigger a one-time run.

  Submit a one-time run. This endpoint allows you to submit a workload directly
  without creating a job. Runs submitted using this endpoint don’t display in
  the UI. Use the jobs/runs/get API to check the run state after the job is
  submitted.

  **Important:** Jobs submitted using this endpoint are not saved as a job. They
  do not show up in the Jobs UI, and do not retry when they fail. Because they
  are not saved, Databricks cannot auto-optimize serverless compute in case of
  failure. If your job fails, you may want to use classic compute to specify the
  compute needs for the job. Alternatively, use the POST /jobs/create and
  POST /jobs/run-now endpoints to create and run a saved job.`

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
			diags := submitJson.Unmarshal(&submitReq)
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

		sdkReq, err := jobs.SubmitRunRequestToSDK(&submitReq)
		if err != nil {
			return err
		}
		wait, err := w.Jobs.SubmitRun(ctx, *sdkReq)
		if err != nil {
			return err
		}
		if submitSkipWait {
			return cmdio.Render(ctx, map[string]any{"run_id": wait.GetRunId()})
		}
		sp := cmdio.NewSpinner(ctx)
		sp.Update("Waiting for submit to complete...")
		response, err := wait.Wait(ctx, lro.WithTimeout(submitTimeout))
		sp.Close()
		if err != nil {
			return err
		}

		out, err := jobs.GetRunResponseFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
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
	*jobs.UpdateJobRequest,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}
	var updateReq jobs.UpdateJobRequest

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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, no positional arguments are allowed. Provide 'job_id' in your JSON input")
			}
			return nil
		}
		return nil
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateJson.Unmarshal(&updateReq)
			if diags.HasError() {
				return diags.Error()
			}
			if len(diags) > 0 {
				err := cmdio.RenderDiagnostics(ctx, diags)
				if err != nil {
					return err
				}
			}
		} else {
			if len(args) == 0 {
				sp := cmdio.NewSpinner(ctx)
				sp.Update("No JOB_ID argument specified. Loading names for Jobs drop-down.")
				names, err := cmdgen.NamedIdMap(ctx, w.Jobs.ListJobsIter(ctx, sdkjobs.ListJobsRequest{}), []string{"Settings", "Name"}, []string{"JobId"})
				sp.Close()
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
				return errors.New("expected to have the canonical identifier of the job to update")
			}
			var updateJobIdArg int64
			_, err = fmt.Sscan(args[0], &updateJobIdArg)
			if err != nil {
				return fmt.Errorf("invalid JOB_ID: %s", args[0])
			}
			updateReq.JobId = &updateJobIdArg

		}

		sdkReq, err := jobs.UpdateJobRequestToSDK(&updateReq)
		if err != nil {
			return err
		}
		response, err := w.Jobs.UpdateJob(ctx, *sdkReq)
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
	*accessmanagement.UpdateObjectPermissionsRequest,
)

func newUpdatePermissions() *cobra.Command {
	cmd := &cobra.Command{}
	var updatePermissionsReq accessmanagement.UpdateObjectPermissionsRequest

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
	cmd.Annotations["launch_stage"] = "GA"
	cmd.Annotations["launch_stage_display"] = "GA"

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
		if len(args) == 0 {
			sp := cmdio.NewSpinner(ctx)
			sp.Update("No JOB_ID argument specified. Loading names for Jobs drop-down.")
			names, err := cmdgen.NamedIdMap(ctx, w.Jobs.ListJobsIter(ctx, sdkjobs.ListJobsRequest{}), []string{"Settings", "Name"}, []string{"JobId"})
			sp.Close()
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
			return errors.New("expected to have the job for which to get or manage permissions")
		}
		_ = args[0] // TODO: field not in sdk-go request: job_id

		updatePermissionsReq.RequestObjectType = new("jobs")

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

// end service Jobs
