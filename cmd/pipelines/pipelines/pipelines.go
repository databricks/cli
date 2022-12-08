package pipelines

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
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

var createPipelineReq pipelines.CreatePipeline

func init() {
	Cmd.AddCommand(createPipelineCmd)
	// TODO: short flags

	createPipelineCmd.Flags().BoolVar(&createPipelineReq.AllowDuplicateNames, "allow-duplicate-names", false, `If false, deployment will fail if name conflicts with that of another pipeline.`)
	createPipelineCmd.Flags().StringVar(&createPipelineReq.Catalog, "catalog", "", `Catalog in UC to add tables to.`)
	createPipelineCmd.Flags().StringVar(&createPipelineReq.Channel, "channel", "", `DLT Release Channel that specifies which version to use.`)
	// TODO: array: clusters
	// TODO: map via StringToStringVar: configuration
	createPipelineCmd.Flags().BoolVar(&createPipelineReq.Continuous, "continuous", false, `Whether the pipeline is continuous or triggered.`)
	createPipelineCmd.Flags().BoolVar(&createPipelineReq.Development, "development", false, `Whether the pipeline is in Development mode.`)
	createPipelineCmd.Flags().BoolVar(&createPipelineReq.DryRun, "dry-run", false, ``)
	createPipelineCmd.Flags().StringVar(&createPipelineReq.Edition, "edition", "", `Pipeline product edition.`)
	// TODO: complex arg: filters
	createPipelineCmd.Flags().StringVar(&createPipelineReq.Id, "id", "", `Unique identifier for this pipeline.`)
	// TODO: array: libraries
	createPipelineCmd.Flags().StringVar(&createPipelineReq.Name, "name", "", `Friendly identifier for this pipeline.`)
	createPipelineCmd.Flags().BoolVar(&createPipelineReq.Photon, "photon", false, `Whether Photon is enabled for this pipeline.`)
	createPipelineCmd.Flags().StringVar(&createPipelineReq.Storage, "storage", "", `DBFS root directory for storing checkpoints and tables.`)
	createPipelineCmd.Flags().StringVar(&createPipelineReq.Target, "target", "", `Target schema (database) to add tables in this pipeline to.`)
	// TODO: complex arg: trigger

}

var createPipelineCmd = &cobra.Command{
	Use:   "create-pipeline",
	Short: `Create a pipeline.`,
	Long: `Create a pipeline.
  
  Creates a new data processing pipeline based on the requested configuration.
  If successful, this method returns the ID of the new pipeline.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Pipelines.CreatePipeline(ctx, createPipelineReq)
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

var deletePipelineReq pipelines.DeletePipeline

func init() {
	Cmd.AddCommand(deletePipelineCmd)
	// TODO: short flags

	deletePipelineCmd.Flags().StringVar(&deletePipelineReq.PipelineId, "pipeline-id", "", ``)

}

var deletePipelineCmd = &cobra.Command{
	Use:   "delete-pipeline",
	Short: `Delete a pipeline.`,
	Long: `Delete a pipeline.
  
  Deletes a pipeline.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Pipelines.DeletePipeline(ctx, deletePipelineReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var getPipelineReq pipelines.GetPipeline

func init() {
	Cmd.AddCommand(getPipelineCmd)
	// TODO: short flags

	getPipelineCmd.Flags().StringVar(&getPipelineReq.PipelineId, "pipeline-id", "", ``)

}

var getPipelineCmd = &cobra.Command{
	Use:   "get-pipeline",
	Short: `Get a pipeline.`,
	Long:  `Get a pipeline.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Pipelines.GetPipeline(ctx, getPipelineReq)
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

var getUpdateReq pipelines.GetUpdate

func init() {
	Cmd.AddCommand(getUpdateCmd)
	// TODO: short flags

	getUpdateCmd.Flags().StringVar(&getUpdateReq.PipelineId, "pipeline-id", "", `The ID of the pipeline.`)
	getUpdateCmd.Flags().StringVar(&getUpdateReq.UpdateId, "update-id", "", `The ID of the update.`)

}

var getUpdateCmd = &cobra.Command{
	Use:   "get-update",
	Short: `Get a pipeline update.`,
	Long: `Get a pipeline update.
  
  Gets an update from an active pipeline.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Pipelines.GetUpdate(ctx, getUpdateReq)
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

var listPipelinesReq pipelines.ListPipelines

func init() {
	Cmd.AddCommand(listPipelinesCmd)
	// TODO: short flags

	listPipelinesCmd.Flags().StringVar(&listPipelinesReq.Filter, "filter", "", `Select a subset of results based on the specified criteria.`)
	listPipelinesCmd.Flags().IntVar(&listPipelinesReq.MaxResults, "max-results", 0, `The maximum number of entries to return in a single page.`)
	// TODO: array: order_by
	listPipelinesCmd.Flags().StringVar(&listPipelinesReq.PageToken, "page-token", "", `Page token returned by previous call.`)

}

var listPipelinesCmd = &cobra.Command{
	Use:   "list-pipelines",
	Short: `List pipelines.`,
	Long: `List pipelines.
  
  Lists pipelines defined in the Delta Live Tables system.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Pipelines.ListPipelinesAll(ctx, listPipelinesReq)
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

var listUpdatesReq pipelines.ListUpdates

func init() {
	Cmd.AddCommand(listUpdatesCmd)
	// TODO: short flags

	listUpdatesCmd.Flags().IntVar(&listUpdatesReq.MaxResults, "max-results", 0, `Max number of entries to return in a single page.`)
	listUpdatesCmd.Flags().StringVar(&listUpdatesReq.PageToken, "page-token", "", `Page token returned by previous call.`)
	listUpdatesCmd.Flags().StringVar(&listUpdatesReq.PipelineId, "pipeline-id", "", `The pipeline to return updates for.`)
	listUpdatesCmd.Flags().StringVar(&listUpdatesReq.UntilUpdateId, "until-update-id", "", `If present, returns updates until and including this update_id.`)

}

var listUpdatesCmd = &cobra.Command{
	Use:   "list-updates",
	Short: `List pipeline updates.`,
	Long: `List pipeline updates.
  
  List updates for an active pipeline.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Pipelines.ListUpdates(ctx, listUpdatesReq)
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

var resetPipelineReq pipelines.ResetPipeline

func init() {
	Cmd.AddCommand(resetPipelineCmd)
	// TODO: short flags

	resetPipelineCmd.Flags().StringVar(&resetPipelineReq.PipelineId, "pipeline-id", "", ``)

}

var resetPipelineCmd = &cobra.Command{
	Use:   "reset-pipeline",
	Short: `Reset a pipeline.`,
	Long: `Reset a pipeline.
  
  Resets a pipeline.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Pipelines.ResetPipeline(ctx, resetPipelineReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var startUpdateReq pipelines.StartUpdate

func init() {
	Cmd.AddCommand(startUpdateCmd)
	// TODO: short flags

	startUpdateCmd.Flags().Var(&startUpdateReq.Cause, "cause", ``)
	startUpdateCmd.Flags().BoolVar(&startUpdateReq.FullRefresh, "full-refresh", false, `If true, this update will reset all tables before running.`)
	// TODO: array: full_refresh_selection
	startUpdateCmd.Flags().StringVar(&startUpdateReq.PipelineId, "pipeline-id", "", ``)
	// TODO: array: refresh_selection

}

var startUpdateCmd = &cobra.Command{
	Use:   "start-update",
	Short: `Queue a pipeline update.`,
	Long: `Queue a pipeline update.
  
  Starts or queues a pipeline update.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Pipelines.StartUpdate(ctx, startUpdateReq)
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

var stopPipelineReq pipelines.StopPipeline

func init() {
	Cmd.AddCommand(stopPipelineCmd)
	// TODO: short flags

	stopPipelineCmd.Flags().StringVar(&stopPipelineReq.PipelineId, "pipeline-id", "", ``)

}

var stopPipelineCmd = &cobra.Command{
	Use:   "stop-pipeline",
	Short: `Stop a pipeline.`,
	Long: `Stop a pipeline.
  
  Stops a pipeline.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Pipelines.StopPipeline(ctx, stopPipelineReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var updatePipelineReq pipelines.EditPipeline

func init() {
	Cmd.AddCommand(updatePipelineCmd)
	// TODO: short flags

	updatePipelineCmd.Flags().BoolVar(&updatePipelineReq.AllowDuplicateNames, "allow-duplicate-names", false, `If false, deployment will fail if name has changed and conflicts the name of another pipeline.`)
	updatePipelineCmd.Flags().StringVar(&updatePipelineReq.Catalog, "catalog", "", `Catalog in UC to add tables to.`)
	updatePipelineCmd.Flags().StringVar(&updatePipelineReq.Channel, "channel", "", `DLT Release Channel that specifies which version to use.`)
	// TODO: array: clusters
	// TODO: map via StringToStringVar: configuration
	updatePipelineCmd.Flags().BoolVar(&updatePipelineReq.Continuous, "continuous", false, `Whether the pipeline is continuous or triggered.`)
	updatePipelineCmd.Flags().BoolVar(&updatePipelineReq.Development, "development", false, `Whether the pipeline is in Development mode.`)
	updatePipelineCmd.Flags().StringVar(&updatePipelineReq.Edition, "edition", "", `Pipeline product edition.`)
	updatePipelineCmd.Flags().Int64Var(&updatePipelineReq.ExpectedLastModified, "expected-last-modified", 0, `If present, the last-modified time of the pipeline settings before the edit.`)
	// TODO: complex arg: filters
	updatePipelineCmd.Flags().StringVar(&updatePipelineReq.Id, "id", "", `Unique identifier for this pipeline.`)
	// TODO: array: libraries
	updatePipelineCmd.Flags().StringVar(&updatePipelineReq.Name, "name", "", `Friendly identifier for this pipeline.`)
	updatePipelineCmd.Flags().BoolVar(&updatePipelineReq.Photon, "photon", false, `Whether Photon is enabled for this pipeline.`)
	updatePipelineCmd.Flags().StringVar(&updatePipelineReq.PipelineId, "pipeline-id", "", `Unique identifier for this pipeline.`)
	updatePipelineCmd.Flags().StringVar(&updatePipelineReq.Storage, "storage", "", `DBFS root directory for storing checkpoints and tables.`)
	updatePipelineCmd.Flags().StringVar(&updatePipelineReq.Target, "target", "", `Target schema (database) to add tables in this pipeline to.`)
	// TODO: complex arg: trigger

}

var updatePipelineCmd = &cobra.Command{
	Use:   "update-pipeline",
	Short: `Edit a pipeline.`,
	Long: `Edit a pipeline.
  
  Updates a pipeline with the supplied configuration.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Pipelines.UpdatePipeline(ctx, updatePipelineReq)
		if err != nil {
			return err
		}

		return nil
	},
}
