// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package pipelines

import (
	"fmt"
	"time"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/libs/cmdio"
	"github.com/databricks/bricks/libs/flags"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
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
}

// start create command

var createReq pipelines.CreatePipeline
var createJson flags.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().BoolVar(&createReq.AllowDuplicateNames, "allow-duplicate-names", createReq.AllowDuplicateNames, `If false, deployment will fail if name conflicts with that of another pipeline.`)
	createCmd.Flags().StringVar(&createReq.Catalog, "catalog", createReq.Catalog, `A catalog in Unity Catalog to publish data from this pipeline to.`)
	createCmd.Flags().StringVar(&createReq.Channel, "channel", createReq.Channel, `DLT Release Channel that specifies which version to use.`)
	// TODO: array: clusters
	// TODO: map via StringToStringVar: configuration
	createCmd.Flags().BoolVar(&createReq.Continuous, "continuous", createReq.Continuous, `Whether the pipeline is continuous or triggered.`)
	createCmd.Flags().BoolVar(&createReq.Development, "development", createReq.Development, `Whether the pipeline is in Development mode.`)
	createCmd.Flags().BoolVar(&createReq.DryRun, "dry-run", createReq.DryRun, ``)
	createCmd.Flags().StringVar(&createReq.Edition, "edition", createReq.Edition, `Pipeline product edition.`)
	// TODO: complex arg: filters
	createCmd.Flags().StringVar(&createReq.Id, "id", createReq.Id, `Unique identifier for this pipeline.`)
	// TODO: array: libraries
	createCmd.Flags().StringVar(&createReq.Name, "name", createReq.Name, `Friendly identifier for this pipeline.`)
	createCmd.Flags().BoolVar(&createReq.Photon, "photon", createReq.Photon, `Whether Photon is enabled for this pipeline.`)
	createCmd.Flags().StringVar(&createReq.Storage, "storage", createReq.Storage, `DBFS root directory for storing checkpoints and tables.`)
	createCmd.Flags().StringVar(&createReq.Target, "target", createReq.Target, `Target schema (database) to add tables in this pipeline to.`)
	// TODO: complex arg: trigger

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a pipeline.`,
	Long: `Create a pipeline.
  
  Creates a new data processing pipeline based on the requested configuration.
  If successful, this method returns the ID of the new pipeline.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		err = createJson.Unmarshall(&createReq)
		if err != nil {
			return err
		}

		response, err := w.Pipelines.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start delete command

var deleteReq pipelines.DeletePipelineRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete PIPELINE_ID",
	Short: `Delete a pipeline.`,
	Long: `Delete a pipeline.
  
  Deletes a pipeline.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Pipelines.PipelineStateInfoNameToPipelineIdMap(ctx, pipelines.ListPipelinesRequest{})
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have ")
		}
		deleteReq.PipelineId = args[0]

		err = w.Pipelines.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq pipelines.GetPipelineRequest

var getSkipWait bool
var getTimeout time.Duration

func init() {
	Cmd.AddCommand(getCmd)

	getCmd.Flags().BoolVar(&getSkipWait, "no-wait", getSkipWait, `do not wait to reach RUNNING state`)
	getCmd.Flags().DurationVar(&getTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get PIPELINE_ID",
	Short: `Get a pipeline.`,
	Long:  `Get a pipeline.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Pipelines.PipelineStateInfoNameToPipelineIdMap(ctx, pipelines.ListPipelinesRequest{})
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have ")
		}
		getReq.PipelineId = args[0]

		response, err := w.Pipelines.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start get-update command

var getUpdateReq pipelines.GetUpdateRequest

func init() {
	Cmd.AddCommand(getUpdateCmd)
	// TODO: short flags

}

var getUpdateCmd = &cobra.Command{
	Use:   "get-update PIPELINE_ID UPDATE_ID",
	Short: `Get a pipeline update.`,
	Long: `Get a pipeline update.
  
  Gets an update from an active pipeline.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		getUpdateReq.PipelineId = args[0]
		getUpdateReq.UpdateId = args[1]

		response, err := w.Pipelines.GetUpdate(ctx, getUpdateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start list-pipeline-events command

var listPipelineEventsReq pipelines.ListPipelineEventsRequest
var listPipelineEventsJson flags.JsonFlag

func init() {
	Cmd.AddCommand(listPipelineEventsCmd)
	// TODO: short flags
	listPipelineEventsCmd.Flags().Var(&listPipelineEventsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	listPipelineEventsCmd.Flags().StringVar(&listPipelineEventsReq.Filter, "filter", listPipelineEventsReq.Filter, `Criteria to select a subset of results, expressed using a SQL-like syntax.`)
	listPipelineEventsCmd.Flags().IntVar(&listPipelineEventsReq.MaxResults, "max-results", listPipelineEventsReq.MaxResults, `Max number of entries to return in a single page.`)
	// TODO: array: order_by
	listPipelineEventsCmd.Flags().StringVar(&listPipelineEventsReq.PageToken, "page-token", listPipelineEventsReq.PageToken, `Page token returned by previous call.`)

}

var listPipelineEventsCmd = &cobra.Command{
	Use:   "list-pipeline-events",
	Short: `List pipeline events.`,
	Long: `List pipeline events.
  
  Retrieves events for a pipeline.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		err = listPipelineEventsJson.Unmarshall(&listPipelineEventsReq)
		if err != nil {
			return err
		}
		listPipelineEventsReq.PipelineId = args[0]

		response, err := w.Pipelines.ListPipelineEventsAll(ctx, listPipelineEventsReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start list-pipelines command

var listPipelinesReq pipelines.ListPipelinesRequest
var listPipelinesJson flags.JsonFlag

func init() {
	Cmd.AddCommand(listPipelinesCmd)
	// TODO: short flags
	listPipelinesCmd.Flags().Var(&listPipelinesJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	listPipelinesCmd.Flags().StringVar(&listPipelinesReq.Filter, "filter", listPipelinesReq.Filter, `Select a subset of results based on the specified criteria.`)
	listPipelinesCmd.Flags().IntVar(&listPipelinesReq.MaxResults, "max-results", listPipelinesReq.MaxResults, `The maximum number of entries to return in a single page.`)
	// TODO: array: order_by
	listPipelinesCmd.Flags().StringVar(&listPipelinesReq.PageToken, "page-token", listPipelinesReq.PageToken, `Page token returned by previous call.`)

}

var listPipelinesCmd = &cobra.Command{
	Use:   "list-pipelines",
	Short: `List pipelines.`,
	Long: `List pipelines.
  
  Lists pipelines defined in the Delta Live Tables system.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		err = listPipelinesJson.Unmarshall(&listPipelinesReq)
		if err != nil {
			return err
		}

		response, err := w.Pipelines.ListPipelinesAll(ctx, listPipelinesReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start list-updates command

var listUpdatesReq pipelines.ListUpdatesRequest

func init() {
	Cmd.AddCommand(listUpdatesCmd)
	// TODO: short flags

	listUpdatesCmd.Flags().IntVar(&listUpdatesReq.MaxResults, "max-results", listUpdatesReq.MaxResults, `Max number of entries to return in a single page.`)
	listUpdatesCmd.Flags().StringVar(&listUpdatesReq.PageToken, "page-token", listUpdatesReq.PageToken, `Page token returned by previous call.`)
	listUpdatesCmd.Flags().StringVar(&listUpdatesReq.UntilUpdateId, "until-update-id", listUpdatesReq.UntilUpdateId, `If present, returns updates until and including this update_id.`)

}

var listUpdatesCmd = &cobra.Command{
	Use:   "list-updates PIPELINE_ID",
	Short: `List pipeline updates.`,
	Long: `List pipeline updates.
  
  List updates for an active pipeline.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Pipelines.PipelineStateInfoNameToPipelineIdMap(ctx, pipelines.ListPipelinesRequest{})
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "The pipeline to return updates for")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have the pipeline to return updates for")
		}
		listUpdatesReq.PipelineId = args[0]

		response, err := w.Pipelines.ListUpdates(ctx, listUpdatesReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start reset command

var resetReq pipelines.ResetRequest

var resetSkipWait bool
var resetTimeout time.Duration

func init() {
	Cmd.AddCommand(resetCmd)

	resetCmd.Flags().BoolVar(&resetSkipWait, "no-wait", resetSkipWait, `do not wait to reach RUNNING state`)
	resetCmd.Flags().DurationVar(&resetTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach RUNNING state`)
	// TODO: short flags

}

var resetCmd = &cobra.Command{
	Use:   "reset PIPELINE_ID",
	Short: `Reset a pipeline.`,
	Long: `Reset a pipeline.
  
  Resets a pipeline.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Pipelines.PipelineStateInfoNameToPipelineIdMap(ctx, pipelines.ListPipelinesRequest{})
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have ")
		}
		resetReq.PipelineId = args[0]

		if resetSkipWait {
			err = w.Pipelines.Reset(ctx, resetReq)
			if err != nil {
				return err
			}
			return nil
		}
		spinner := cmdio.Spinner(ctx)
		info, err := w.Pipelines.ResetAndWait(ctx, resetReq,
			retries.Timeout[pipelines.GetPipelineResponse](resetTimeout),
			func(i *retries.Info[pipelines.GetPipelineResponse]) {
				if i.Info == nil {
					return
				}
				statusMessage := i.Info.Cause
				spinner <- statusMessage
			})
		close(spinner)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, info)
	},
}

// start start-update command

var startUpdateReq pipelines.StartUpdate
var startUpdateJson flags.JsonFlag

func init() {
	Cmd.AddCommand(startUpdateCmd)
	// TODO: short flags
	startUpdateCmd.Flags().Var(&startUpdateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	startUpdateCmd.Flags().Var(&startUpdateReq.Cause, "cause", ``)
	startUpdateCmd.Flags().BoolVar(&startUpdateReq.FullRefresh, "full-refresh", startUpdateReq.FullRefresh, `If true, this update will reset all tables before running.`)
	// TODO: array: full_refresh_selection
	// TODO: array: refresh_selection

}

var startUpdateCmd = &cobra.Command{
	Use:   "start-update",
	Short: `Queue a pipeline update.`,
	Long: `Queue a pipeline update.
  
  Starts or queues a pipeline update.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		err = startUpdateJson.Unmarshall(&startUpdateReq)
		if err != nil {
			return err
		}
		startUpdateReq.PipelineId = args[0]

		response, err := w.Pipelines.StartUpdate(ctx, startUpdateReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	},
}

// start stop command

var stopReq pipelines.StopRequest

var stopSkipWait bool
var stopTimeout time.Duration

func init() {
	Cmd.AddCommand(stopCmd)

	stopCmd.Flags().BoolVar(&stopSkipWait, "no-wait", stopSkipWait, `do not wait to reach IDLE state`)
	stopCmd.Flags().DurationVar(&stopTimeout, "timeout", 20*time.Minute, `maximum amount of time to reach IDLE state`)
	// TODO: short flags

}

var stopCmd = &cobra.Command{
	Use:   "stop PIPELINE_ID",
	Short: `Stop a pipeline.`,
	Long: `Stop a pipeline.
  
  Stops a pipeline.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		if len(args) == 0 {
			names, err := w.Pipelines.PipelineStateInfoNameToPipelineIdMap(ctx, pipelines.ListPipelinesRequest{})
			if err != nil {
				return err
			}
			id, err := cmdio.Select(ctx, names, "")
			if err != nil {
				return err
			}
			args = append(args, id)
		}
		if len(args) != 1 {
			return fmt.Errorf("expected to have ")
		}
		stopReq.PipelineId = args[0]

		if stopSkipWait {
			err = w.Pipelines.Stop(ctx, stopReq)
			if err != nil {
				return err
			}
			return nil
		}
		spinner := cmdio.Spinner(ctx)
		info, err := w.Pipelines.StopAndWait(ctx, stopReq,
			retries.Timeout[pipelines.GetPipelineResponse](stopTimeout),
			func(i *retries.Info[pipelines.GetPipelineResponse]) {
				if i.Info == nil {
					return
				}
				statusMessage := i.Info.Cause
				spinner <- statusMessage
			})
		close(spinner)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, info)
	},
}

// start update command

var updateReq pipelines.EditPipeline
var updateJson flags.JsonFlag

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags
	updateCmd.Flags().Var(&updateJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	updateCmd.Flags().BoolVar(&updateReq.AllowDuplicateNames, "allow-duplicate-names", updateReq.AllowDuplicateNames, `If false, deployment will fail if name has changed and conflicts the name of another pipeline.`)
	updateCmd.Flags().StringVar(&updateReq.Catalog, "catalog", updateReq.Catalog, `A catalog in Unity Catalog to publish data from this pipeline to.`)
	updateCmd.Flags().StringVar(&updateReq.Channel, "channel", updateReq.Channel, `DLT Release Channel that specifies which version to use.`)
	// TODO: array: clusters
	// TODO: map via StringToStringVar: configuration
	updateCmd.Flags().BoolVar(&updateReq.Continuous, "continuous", updateReq.Continuous, `Whether the pipeline is continuous or triggered.`)
	updateCmd.Flags().BoolVar(&updateReq.Development, "development", updateReq.Development, `Whether the pipeline is in Development mode.`)
	updateCmd.Flags().StringVar(&updateReq.Edition, "edition", updateReq.Edition, `Pipeline product edition.`)
	updateCmd.Flags().Int64Var(&updateReq.ExpectedLastModified, "expected-last-modified", updateReq.ExpectedLastModified, `If present, the last-modified time of the pipeline settings before the edit.`)
	// TODO: complex arg: filters
	updateCmd.Flags().StringVar(&updateReq.Id, "id", updateReq.Id, `Unique identifier for this pipeline.`)
	// TODO: array: libraries
	updateCmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `Friendly identifier for this pipeline.`)
	updateCmd.Flags().BoolVar(&updateReq.Photon, "photon", updateReq.Photon, `Whether Photon is enabled for this pipeline.`)
	updateCmd.Flags().StringVar(&updateReq.PipelineId, "pipeline-id", updateReq.PipelineId, `Unique identifier for this pipeline.`)
	updateCmd.Flags().StringVar(&updateReq.Storage, "storage", updateReq.Storage, `DBFS root directory for storing checkpoints and tables.`)
	updateCmd.Flags().StringVar(&updateReq.Target, "target", updateReq.Target, `Target schema (database) to add tables in this pipeline to.`)
	// TODO: complex arg: trigger

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Edit a pipeline.`,
	Long: `Edit a pipeline.
  
  Updates a pipeline with the supplied configuration.`,

	Annotations: map[string]string{},
	PreRunE:     root.MustWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		err = updateJson.Unmarshall(&updateReq)
		if err != nil {
			return err
		}
		updateReq.PipelineId = args[0]

		err = w.Pipelines.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Pipelines
