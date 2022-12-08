package experiments

import (
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/bricks/project"
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
	// TODO: complex arg: tags

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create experiment.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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
	// TODO: complex arg: order_by
	searchCmd.Flags().StringVar(&searchReq.PageToken, "page-token", "", `Token indicating the page of experiments to fetch.`)
	// TODO: complex arg: view_type

}

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: `Search experiments.`,

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
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

	PreRunE: project.Configure, // TODO: improve logic for bundle/non-bundle invocations
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := project.Get(ctx).WorkspacesClient()
		err := w.Experiments.Update(ctx, updateReq)
		if err != nil {
			return err
		}

		return nil
	},
}
