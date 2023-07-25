// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package pipelines

import (
	"fmt"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipelines",
		Short: `The Delta Live Tables API allows you to create, edit, delete, start, and view details about pipelines.`,
		Long: `The Delta Live Tables API allows you to create, edit, delete, start, and view
  details about pipelines.
  
  Delta Live Tables is a framework for building reliable, maintainable, and
  testable data processing pipelines. You define the transformations to perform
  on your data, and Delta Live Tables manages task orchestration, cluster
  management, monitoring, data quality, and error handling.
  
  Instead of defining your data pipelines using a series of separate Apache
  Spark tasks, Delta Live Tables manages how your data is transformed based on a
  target schema you define for each processing step. You can also enforce data
  quality with Delta Live Tables expectations. Expectations allow you to define
  expected data quality and specify how to handle records that fail those
  expectations.`,
		GroupID: "pipelines",
		Annotations: map[string]string{
			"package": "pipelines",
		},
	}

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createOverrides []func(
	*cobra.Command,
	*pipelines.CreatePipeline,
)

func newCreate() *cobra.Command {
	cmd := &cobra.Command{}

	var createReq pipelines.CreatePipeline
	var createJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Use = "create"
	cmd.Short = `Create a pipeline.`
	cmd.Long = `Create a pipeline.
  
  Creates a new data processing pipeline based on the requested configuration.
  If successful, this method returns the ID of the new pipeline.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = createJson.Unmarshal(&createReq)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("please provide command input in JSON format by specifying the --json flag")
		}

		response, err := w.Pipelines.Create(ctx, createReq)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newCreate())
	})
}

// start delete command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteOverrides []func(
	*cobra.Command,
	*pipelines.DeletePipelineRequest,
)

func newDelete() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteReq pipelines.DeletePipelineRequest

	// TODO: short flags

	cmd.Use = "delete PIPELINE_ID"
	cmd.Short = `Delete a pipeline.`
	cmd.Long = `Delete a pipeline.
  
  Deletes a pipeline.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		deleteReq.PipelineId = args[0]

		err = w.Pipelines.Delete(ctx, deleteReq)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newDelete())
	})
}

// start get command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getOverrides []func(
	*cobra.Command,
	*pipelines.GetPipelineRequest,
)

func newGet() *cobra.Command {
	cmd := &cobra.Command{}

	var getReq pipelines.GetPipelineRequest

	var getSkipWait bool
	var getTimeout time.Duration

	cmd.Flags().BoolVar(&getSkipWait, "no-wait", getSkipWait, `do not wait to reach RUNNING state`)
	cmd.Flags().DurationVar(&getTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)
	// TODO: short flags

	cmd.Use = "get PIPELINE_ID"
	cmd.Short = `Get a pipeline.`
	cmd.Long = `Get a pipeline.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		getReq.PipelineId = args[0]

		response, err := w.Pipelines.Get(ctx, getReq)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newGet())
	})
}

// start get-update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getUpdateOverrides []func(
	*cobra.Command,
	*pipelines.GetUpdateRequest,
)

func newGetUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var getUpdateReq pipelines.GetUpdateRequest

	// TODO: short flags

	cmd.Use = "get-update PIPELINE_ID UPDATE_ID"
	cmd.Short = `Get a pipeline update.`
	cmd.Long = `Get a pipeline update.
  
  Gets an update from an active pipeline.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		getUpdateReq.PipelineId = args[0]
		getUpdateReq.UpdateId = args[1]

		response, err := w.Pipelines.GetUpdate(ctx, getUpdateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getUpdateOverrides {
		fn(cmd, &getUpdateReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newGetUpdate())
	})
}

// start list-pipeline-events command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listPipelineEventsOverrides []func(
	*cobra.Command,
	*pipelines.ListPipelineEventsRequest,
)

func newListPipelineEvents() *cobra.Command {
	cmd := &cobra.Command{}

	var listPipelineEventsReq pipelines.ListPipelineEventsRequest
	var listPipelineEventsJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&listPipelineEventsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&listPipelineEventsReq.Filter, "filter", listPipelineEventsReq.Filter, `Criteria to select a subset of results, expressed using a SQL-like syntax.`)
	cmd.Flags().IntVar(&listPipelineEventsReq.MaxResults, "max-results", listPipelineEventsReq.MaxResults, `Max number of entries to return in a single page.`)
	// TODO: array: order_by
	cmd.Flags().StringVar(&listPipelineEventsReq.PageToken, "page-token", listPipelineEventsReq.PageToken, `Page token returned by previous call.`)

	cmd.Use = "list-pipeline-events PIPELINE_ID"
	cmd.Short = `List pipeline events.`
	cmd.Long = `List pipeline events.
  
  Retrieves events for a pipeline.`

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
			err = listPipelineEventsJson.Unmarshal(&listPipelineEventsReq)
			if err != nil {
				return err
			}
		}
		listPipelineEventsReq.PipelineId = args[0]

		response, err := w.Pipelines.ListPipelineEventsAll(ctx, listPipelineEventsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listPipelineEventsOverrides {
		fn(cmd, &listPipelineEventsReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newListPipelineEvents())
	})
}

// start list-pipelines command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listPipelinesOverrides []func(
	*cobra.Command,
	*pipelines.ListPipelinesRequest,
)

func newListPipelines() *cobra.Command {
	cmd := &cobra.Command{}

	var listPipelinesReq pipelines.ListPipelinesRequest
	var listPipelinesJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&listPipelinesJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&listPipelinesReq.Filter, "filter", listPipelinesReq.Filter, `Select a subset of results based on the specified criteria.`)
	cmd.Flags().IntVar(&listPipelinesReq.MaxResults, "max-results", listPipelinesReq.MaxResults, `The maximum number of entries to return in a single page.`)
	// TODO: array: order_by
	cmd.Flags().StringVar(&listPipelinesReq.PageToken, "page-token", listPipelinesReq.PageToken, `Page token returned by previous call.`)

	cmd.Use = "list-pipelines"
	cmd.Short = `List pipelines.`
	cmd.Long = `List pipelines.
  
  Lists pipelines defined in the Delta Live Tables system.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(0)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = listPipelinesJson.Unmarshal(&listPipelinesReq)
			if err != nil {
				return err
			}
		} else {
		}

		response, err := w.Pipelines.ListPipelinesAll(ctx, listPipelinesReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listPipelinesOverrides {
		fn(cmd, &listPipelinesReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newListPipelines())
	})
}

// start list-updates command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listUpdatesOverrides []func(
	*cobra.Command,
	*pipelines.ListUpdatesRequest,
)

func newListUpdates() *cobra.Command {
	cmd := &cobra.Command{}

	var listUpdatesReq pipelines.ListUpdatesRequest

	// TODO: short flags

	cmd.Flags().IntVar(&listUpdatesReq.MaxResults, "max-results", listUpdatesReq.MaxResults, `Max number of entries to return in a single page.`)
	cmd.Flags().StringVar(&listUpdatesReq.PageToken, "page-token", listUpdatesReq.PageToken, `Page token returned by previous call.`)
	cmd.Flags().StringVar(&listUpdatesReq.UntilUpdateId, "until-update-id", listUpdatesReq.UntilUpdateId, `If present, returns updates until and including this update_id.`)

	cmd.Use = "list-updates PIPELINE_ID"
	cmd.Short = `List pipeline updates.`
	cmd.Long = `List pipeline updates.
  
  List updates for an active pipeline.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		listUpdatesReq.PipelineId = args[0]

		response, err := w.Pipelines.ListUpdates(ctx, listUpdatesReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listUpdatesOverrides {
		fn(cmd, &listUpdatesReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newListUpdates())
	})
}

// start reset command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var resetOverrides []func(
	*cobra.Command,
	*pipelines.ResetRequest,
)

func newReset() *cobra.Command {
	cmd := &cobra.Command{}

	var resetReq pipelines.ResetRequest

	var resetSkipWait bool
	var resetTimeout time.Duration

	cmd.Flags().BoolVar(&resetSkipWait, "no-wait", resetSkipWait, `do not wait to reach RUNNING state`)
	cmd.Flags().DurationVar(&resetTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)
	// TODO: short flags

	cmd.Use = "reset PIPELINE_ID"
	cmd.Short = `Reset a pipeline.`
	cmd.Long = `Reset a pipeline.
  
  Resets a pipeline.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		resetReq.PipelineId = args[0]

		wait, err := w.Pipelines.Reset(ctx, resetReq)
		if err != nil {
			return err
		}
		if resetSkipWait {
			return nil
		}
		spinner := cmdio.Spinner(ctx)
		info, err := wait.OnProgress(func(i *pipelines.GetPipelineResponse) {
			statusMessage := i.Cause
			spinner <- statusMessage
		}).GetWithTimeout(resetTimeout)
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
	for _, fn := range resetOverrides {
		fn(cmd, &resetReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newReset())
	})
}

// start start-update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var startUpdateOverrides []func(
	*cobra.Command,
	*pipelines.StartUpdate,
)

func newStartUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var startUpdateReq pipelines.StartUpdate
	var startUpdateJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&startUpdateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().Var(&startUpdateReq.Cause, "cause", ``)
	cmd.Flags().BoolVar(&startUpdateReq.FullRefresh, "full-refresh", startUpdateReq.FullRefresh, `If true, this update will reset all tables before running.`)
	// TODO: array: full_refresh_selection
	// TODO: array: refresh_selection

	cmd.Use = "start-update PIPELINE_ID"
	cmd.Short = `Queue a pipeline update.`
	cmd.Long = `Queue a pipeline update.
  
  Starts or queues a pipeline update.`

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
			err = startUpdateJson.Unmarshal(&startUpdateReq)
			if err != nil {
				return err
			}
		}
		startUpdateReq.PipelineId = args[0]

		response, err := w.Pipelines.StartUpdate(ctx, startUpdateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range startUpdateOverrides {
		fn(cmd, &startUpdateReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newStartUpdate())
	})
}

// start stop command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var stopOverrides []func(
	*cobra.Command,
	*pipelines.StopRequest,
)

func newStop() *cobra.Command {
	cmd := &cobra.Command{}

	var stopReq pipelines.StopRequest

	var stopSkipWait bool
	var stopTimeout time.Duration

	cmd.Flags().BoolVar(&stopSkipWait, "no-wait", stopSkipWait, `do not wait to reach IDLE state`)
	cmd.Flags().DurationVar(&stopTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach IDLE state`)
	// TODO: short flags

	cmd.Use = "stop PIPELINE_ID"
	cmd.Short = `Stop a pipeline.`
	cmd.Long = `Stop a pipeline.
  
  Stops a pipeline.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		stopReq.PipelineId = args[0]

		wait, err := w.Pipelines.Stop(ctx, stopReq)
		if err != nil {
			return err
		}
		if stopSkipWait {
			return nil
		}
		spinner := cmdio.Spinner(ctx)
		info, err := wait.OnProgress(func(i *pipelines.GetPipelineResponse) {
			statusMessage := i.Cause
			spinner <- statusMessage
		}).GetWithTimeout(stopTimeout)
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
	for _, fn := range stopOverrides {
		fn(cmd, &stopReq)
	}

	return cmd
}

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newStop())
	})
}

// start update command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateOverrides []func(
	*cobra.Command,
	*pipelines.EditPipeline,
)

func newUpdate() *cobra.Command {
	cmd := &cobra.Command{}

	var updateReq pipelines.EditPipeline
	var updateJson flags.JsonFlag

	// TODO: short flags
	cmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().BoolVar(&updateReq.AllowDuplicateNames, "allow-duplicate-names", updateReq.AllowDuplicateNames, `If false, deployment will fail if name has changed and conflicts the name of another pipeline.`)
	cmd.Flags().StringVar(&updateReq.Catalog, "catalog", updateReq.Catalog, `A catalog in Unity Catalog to publish data from this pipeline to.`)
	cmd.Flags().StringVar(&updateReq.Channel, "channel", updateReq.Channel, `DLT Release Channel that specifies which version to use.`)
	// TODO: array: clusters
	// TODO: map via StringToStringVar: configuration
	cmd.Flags().BoolVar(&updateReq.Continuous, "continuous", updateReq.Continuous, `Whether the pipeline is continuous or triggered.`)
	cmd.Flags().BoolVar(&updateReq.Development, "development", updateReq.Development, `Whether the pipeline is in Development mode.`)
	cmd.Flags().StringVar(&updateReq.Edition, "edition", updateReq.Edition, `Pipeline product edition.`)
	cmd.Flags().Int64Var(&updateReq.ExpectedLastModified, "expected-last-modified", updateReq.ExpectedLastModified, `If present, the last-modified time of the pipeline settings before the edit.`)
	// TODO: complex arg: filters
	cmd.Flags().StringVar(&updateReq.Id, "id", updateReq.Id, `Unique identifier for this pipeline.`)
	// TODO: array: libraries
	cmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `Friendly identifier for this pipeline.`)
	cmd.Flags().BoolVar(&updateReq.Photon, "photon", updateReq.Photon, `Whether Photon is enabled for this pipeline.`)
	cmd.Flags().StringVar(&updateReq.PipelineId, "pipeline-id", updateReq.PipelineId, `Unique identifier for this pipeline.`)
	cmd.Flags().BoolVar(&updateReq.Serverless, "serverless", updateReq.Serverless, `Whether serverless compute is enabled for this pipeline.`)
	cmd.Flags().StringVar(&updateReq.Storage, "storage", updateReq.Storage, `DBFS root directory for storing checkpoints and tables.`)
	cmd.Flags().StringVar(&updateReq.Target, "target", updateReq.Target, `Target schema (database) to add tables in this pipeline to.`)
	// TODO: complex arg: trigger

	cmd.Use = "update PIPELINE_ID"
	cmd.Short = `Edit a pipeline.`
	cmd.Long = `Edit a pipeline.
  
  Updates a pipeline with the supplied configuration.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := cobra.ExactArgs(1)
		if cmd.Flags().Changed("json") {
			check = cobra.ExactArgs(0)
		}
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			err = updateJson.Unmarshal(&updateReq)
			if err != nil {
				return err
			}
		} else {
			updateReq.PipelineId = args[0]
		}

		err = w.Pipelines.Update(ctx, updateReq)
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

func init() {
	cmdOverrides = append(cmdOverrides, func(cmd *cobra.Command) {
		cmd.AddCommand(newUpdate())
	})
}

// end service Pipelines
