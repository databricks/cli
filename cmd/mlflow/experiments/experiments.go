// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package experiments

import (
	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/mlflow"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "experiments",
}

// start create command

var createReq mlflow.CreateExperiment
var createJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().StringVar(&createReq.ArtifactLocation, "artifact-location", createReq.ArtifactLocation, `Location where all artifacts for the experiment are stored.`)
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

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = createJson.Unmarshall(&createReq)
		if err != nil {
			return err
		}
		createReq.Name = args[0]

		response, err := w.Experiments.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq mlflow.DeleteExperiment

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete EXPERIMENT_ID",
	Short: `Delete an experiment.`,
	Long: `Delete an experiment.
  
  Marks an experiment and associated metadata, runs, metrics, params, and tags
  for deletion. If the experiment uses FileStore, artifacts associated with
  experiment are also deleted.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		deleteReq.ExperimentId = args[0]

		err = w.Experiments.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq mlflow.GetExperimentRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get EXPERIMENT_ID",
	Short: `Get an experiment.`,
	Long: `Get an experiment.
  
  Gets metadata for an experiment. This method works on deleted experiments.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		getReq.ExperimentId = args[0]

		response, err := w.Experiments.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start get-by-name command

var getByNameReq mlflow.GetByNameRequest

func init() {
	Cmd.AddCommand(getByNameCmd)
	// TODO: short flags

}

var getByNameCmd = &cobra.Command{
	Use:   "get-by-name EXPERIMENT_NAME",
	Short: `Get metadata.`,
	Long: `Get metadata.
  
  "Gets metadata for an experiment.
  
  This endpoint will return deleted experiments, but prefers the active
  experiment if an active and deleted experiment share the same name. If
  multiple deleted\nexperiments share the same name, the API will return one of
  them.
  
  Throws RESOURCE_DOES_NOT_EXIST if no experiment with the specified name
  exists.S`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		getByNameReq.ExperimentName = args[0]

		response, err := w.Experiments.GetByName(ctx, getByNameReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list command

var listReq mlflow.ListExperimentsRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	listCmd.Flags().IntVar(&listReq.MaxResults, "max-results", listReq.MaxResults, `Maximum number of experiments desired.`)
	listCmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Token indicating the page of experiments to fetch.`)
	listCmd.Flags().StringVar(&listReq.ViewType, "view-type", listReq.ViewType, `Qualifier for type of experiments to be returned.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List experiments.`,
	Long: `List experiments.
  
  Gets a list of all experiments.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(0),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)

		response, err := w.Experiments.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start restore command

var restoreReq mlflow.RestoreExperiment

func init() {
	Cmd.AddCommand(restoreCmd)
	// TODO: short flags

}

var restoreCmd = &cobra.Command{
	Use:   "restore EXPERIMENT_ID",
	Short: `Restores an experiment.`,
	Long: `Restores an experiment.
  
  "Restore an experiment marked for deletion. This also restores\nassociated
  metadata, runs, metrics, params, and tags. If experiment uses FileStore,
  underlying\nartifacts associated with experiment are also restored.\n\nThrows
  RESOURCE_DOES_NOT_EXIST if experiment was never created or was permanently
  deleted.",`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		restoreReq.ExperimentId = args[0]

		err = w.Experiments.Restore(ctx, restoreReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start search command

var searchReq mlflow.SearchExperiments
var searchJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(searchCmd)
	// TODO: short flags
	searchCmd.Flags().Var(&searchJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	searchCmd.Flags().StringVar(&searchReq.Filter, "filter", searchReq.Filter, `String representing a SQL filter condition (e.g.`)
	searchCmd.Flags().Int64Var(&searchReq.MaxResults, "max-results", searchReq.MaxResults, `Maximum number of experiments desired.`)
	// TODO: array: order_by
	searchCmd.Flags().StringVar(&searchReq.PageToken, "page-token", searchReq.PageToken, `Token indicating the page of experiments to fetch.`)
	searchCmd.Flags().Var(&searchReq.ViewType, "view-type", `Qualifier for type of experiments to be returned.`)

}

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: `Search experiments.`,
	Long: `Search experiments.
  
  Searches for experiments that satisfy specified search criteria.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = searchJson.Unmarshall(&searchReq)
		if err != nil {
			return err
		}

		response, err := w.Experiments.SearchAll(ctx, searchReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start set-experiment-tag command

var setExperimentTagReq mlflow.SetExperimentTag

func init() {
	Cmd.AddCommand(setExperimentTagCmd)
	// TODO: short flags

}

var setExperimentTagCmd = &cobra.Command{
	Use:   "set-experiment-tag EXPERIMENT_ID KEY VALUE",
	Short: `Set a tag.`,
	Long: `Set a tag.
  
  Sets a tag on an experiment. Experiment tags are metadata that can be updated.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(3),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		setExperimentTagReq.ExperimentId = args[0]
		setExperimentTagReq.Key = args[1]
		setExperimentTagReq.Value = args[2]

		err = w.Experiments.SetExperimentTag(ctx, setExperimentTagReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start update command

var updateReq mlflow.UpdateExperiment

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().StringVar(&updateReq.NewName, "new-name", updateReq.NewName, `If provided, the experiment's name is changed to the new name.`)

}

var updateCmd = &cobra.Command{
	Use:   "update EXPERIMENT_ID",
	Short: `Update an experiment.`,
	Long: `Update an experiment.
  
  Updates experiment metadata.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(1),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		updateReq.ExperimentId = args[0]

		err = w.Experiments.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service Experiments
