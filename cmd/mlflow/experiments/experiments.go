package experiments

import (
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/mlflow"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "experiments",
}

var createReq mlflow.CreateExperiment

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags

	createCmd.Flags().StringVar(&createReq.ArtifactLocation, "artifact-location", "", `Location where all artifacts for the experiment are stored.`)
	createCmd.Flags().StringVar(&createReq.Name, "name", "", `Experiment name.`)
	// TODO: array: tags

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create experiment.`,
	Long: `Create experiment.
  
  Creates an experiment with a name. Returns the ID of the newly created
  experiment. Validates that another experiment with the same name does not
  already exist and fails if another experiment with the same name already
  exists.
  
  Throws RESOURCE_ALREADY_EXISTS if a experiment with the given name exists.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Experiments.Create(ctx, createReq)
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

var deleteReq mlflow.DeleteExperiment

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.ExperimentId, "experiment-id", "", `ID of the associated experiment.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete an experiment.`,
	Long: `Delete an experiment.
  
  Marks an experiment and associated metadata, runs, metrics, params, and tags
  for deletion. If the experiment uses FileStore, artifacts associated with
  experiment are also deleted.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Experiments.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var getReq mlflow.GetExperimentRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.ExperimentId, "experiment-id", "", `ID of the associated experiment.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get an experiment.`,
	Long: `Get an experiment.
  
  Gets metadata for an experiment. This method works on deleted experiments.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Experiments.Get(ctx, getReq)
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

var getByNameReq mlflow.GetByNameRequest

func init() {
	Cmd.AddCommand(getByNameCmd)
	// TODO: short flags

	getByNameCmd.Flags().StringVar(&getByNameReq.ExperimentName, "experiment-name", "", `Name of the associated experiment.`)

}

var getByNameCmd = &cobra.Command{
	Use:   "get-by-name",
	Short: `Get metadata.`,
	Long: `Get metadata.
  
  "Gets metadata for an experiment.
  
  This endpoint will return deleted experiments, but prefers the active
  experiment if an active and deleted experiment share the same name. If
  multiple deleted\nexperiments share the same name, the API will return one of
  them.
  
  Throws RESOURCE_DOES_NOT_EXIST if no experiment with the specified name
  exists.S`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Experiments.GetByName(ctx, getByNameReq)
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

var listReq mlflow.ListExperimentsRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	listCmd.Flags().IntVar(&listReq.MaxResults, "max-results", 0, `Maximum number of experiments desired.`)
	listCmd.Flags().StringVar(&listReq.PageToken, "page-token", "", `Token indicating the page of experiments to fetch.`)
	listCmd.Flags().StringVar(&listReq.ViewType, "view-type", "", `Qualifier for type of experiments to be returned.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List experiments.`,
	Long: `List experiments.
  
  Gets a list of all experiments.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Experiments.ListAll(ctx, listReq)
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

var restoreReq mlflow.RestoreExperiment

func init() {
	Cmd.AddCommand(restoreCmd)
	// TODO: short flags

	restoreCmd.Flags().StringVar(&restoreReq.ExperimentId, "experiment-id", "", `ID of the associated experiment.`)

}

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: `Restores an experiment.`,
	Long: `Restores an experiment.
  
  "Restore an experiment marked for deletion. This also restores\nassociated
  metadata, runs, metrics, params, and tags. If experiment uses FileStore,
  underlying\nartifacts associated with experiment are also restored.\n\nThrows
  RESOURCE_DOES_NOT_EXIST if experiment was never created or was permanently
  deleted.",`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Experiments.Restore(ctx, restoreReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var searchReq mlflow.SearchExperiments

func init() {
	Cmd.AddCommand(searchCmd)
	// TODO: short flags

	searchCmd.Flags().StringVar(&searchReq.Filter, "filter", "", `String representing a SQL filter condition (e.g.`)
	searchCmd.Flags().Int64Var(&searchReq.MaxResults, "max-results", 0, `Maximum number of experiments desired.`)
	// TODO: array: order_by
	searchCmd.Flags().StringVar(&searchReq.PageToken, "page-token", "", `Token indicating the page of experiments to fetch.`)
	searchCmd.Flags().Var(&searchReq.ViewType, "view-type", `Qualifier for type of experiments to be returned.`)

}

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: `Search experiments.`,
	Long: `Search experiments.
  
  Searches for experiments that satisfy specified search criteria.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.Experiments.SearchAll(ctx, searchReq)
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

var setExperimentTagReq mlflow.SetExperimentTag

func init() {
	Cmd.AddCommand(setExperimentTagCmd)
	// TODO: short flags

	setExperimentTagCmd.Flags().StringVar(&setExperimentTagReq.ExperimentId, "experiment-id", "", `ID of the experiment under which to log the tag.`)
	setExperimentTagCmd.Flags().StringVar(&setExperimentTagReq.Key, "key", "", `Name of the tag.`)
	setExperimentTagCmd.Flags().StringVar(&setExperimentTagReq.Value, "value", "", `String value of the tag being logged.`)

}

var setExperimentTagCmd = &cobra.Command{
	Use:   "set-experiment-tag",
	Short: `Set a tag.`,
	Long: `Set a tag.
  
  Sets a tag on an experiment. Experiment tags are metadata that can be updated.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Experiments.SetExperimentTag(ctx, setExperimentTagReq)
		if err != nil {
			return err
		}

		return nil
	},
}

var updateReq mlflow.UpdateExperiment

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().StringVar(&updateReq.ExperimentId, "experiment-id", "", `ID of the associated experiment.`)
	updateCmd.Flags().StringVar(&updateReq.NewName, "new-name", "", `If provided, the experiment's name is changed to the new name.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update an experiment.`,
	Long: `Update an experiment.
  
  Updates experiment metadata.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err := w.Experiments.Update(ctx, updateReq)
		if err != nil {
			return err
		}

		return nil
	},
}
