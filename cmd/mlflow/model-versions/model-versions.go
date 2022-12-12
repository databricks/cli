// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package model_versions

import (
	"fmt"

	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/mlflow"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "model-versions",
}

// start create command

var createReq mlflow.CreateModelVersionRequest
var createJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().StringVar(&createReq.Description, "description", createReq.Description, `Optional description for model version.`)
	createCmd.Flags().StringVar(&createReq.RunId, "run-id", createReq.RunId, `MLflow run ID for correlation, if source was generated by an experiment run in MLflow tracking server.`)
	createCmd.Flags().StringVar(&createReq.RunLink, "run-link", createReq.RunLink, `MLflow run link - this is the exact link of the run that generated this model version, potentially hosted at another instance of MLflow.`)
	// TODO: array: tags

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a model version.`,
	Long: `Create a model version.
  
  Creates a model version.`,

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
		createReq.Source = args[1]

		response, err := w.ModelVersions.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq mlflow.DeleteModelVersionRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

}

var deleteCmd = &cobra.Command{
	Use:   "delete NAME VERSION",
	Short: `Delete a model version.`,
	Long: `Delete a model version.
  
  Deletes a model version.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		deleteReq.Name = args[0]
		deleteReq.Version = args[1]

		err = w.ModelVersions.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start delete-tag command

var deleteTagReq mlflow.DeleteModelVersionTagRequest

func init() {
	Cmd.AddCommand(deleteTagCmd)
	// TODO: short flags

}

var deleteTagCmd = &cobra.Command{
	Use:   "delete-tag NAME VERSION KEY",
	Short: `Delete a model version tag.`,
	Long: `Delete a model version tag.
  
  Deletes a model version tag.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(3),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		deleteTagReq.Name = args[0]
		deleteTagReq.Version = args[1]
		deleteTagReq.Key = args[2]

		err = w.ModelVersions.DeleteTag(ctx, deleteTagReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq mlflow.GetModelVersionRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

}

var getCmd = &cobra.Command{
	Use:   "get NAME VERSION",
	Short: `Get a model version.`,
	Long: `Get a model version.
  
  Get a model version.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		getReq.Name = args[0]
		getReq.Version = args[1]

		response, err := w.ModelVersions.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start get-download-uri command

var getDownloadUriReq mlflow.GetModelVersionDownloadUriRequest

func init() {
	Cmd.AddCommand(getDownloadUriCmd)
	// TODO: short flags

}

var getDownloadUriCmd = &cobra.Command{
	Use:   "get-download-uri NAME VERSION",
	Short: `Get a model version URI.`,
	Long: `Get a model version URI.
  
  Gets a URI to download the model version.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		getDownloadUriReq.Name = args[0]
		getDownloadUriReq.Version = args[1]

		response, err := w.ModelVersions.GetDownloadUri(ctx, getDownloadUriReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start search command

var searchReq mlflow.SearchModelVersionsRequest
var searchJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(searchCmd)
	// TODO: short flags
	searchCmd.Flags().Var(&searchJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	searchCmd.Flags().StringVar(&searchReq.Filter, "filter", searchReq.Filter, `String filter condition, like "name='my-model-name'".`)
	searchCmd.Flags().IntVar(&searchReq.MaxResults, "max-results", searchReq.MaxResults, `Maximum number of models desired.`)
	// TODO: array: order_by
	searchCmd.Flags().StringVar(&searchReq.PageToken, "page-token", searchReq.PageToken, `Pagination token to go to next page based on previous search query.`)

}

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: `Searches model versions.`,
	Long: `Searches model versions.
  
  Searches for specific model versions based on the supplied __filter__.`,

	Annotations: map[string]string{},
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = searchJson.Unmarshall(&searchReq)
		if err != nil {
			return err
		}

		response, err := w.ModelVersions.SearchAll(ctx, searchReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start set-tag command

var setTagReq mlflow.SetModelVersionTagRequest

func init() {
	Cmd.AddCommand(setTagCmd)
	// TODO: short flags

}

var setTagCmd = &cobra.Command{
	Use:   "set-tag NAME VERSION KEY VALUE",
	Short: `Set a version tag.`,
	Long: `Set a version tag.
  
  Sets a model version tag.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(4),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		setTagReq.Name = args[0]
		setTagReq.Version = args[1]
		setTagReq.Key = args[2]
		setTagReq.Value = args[3]

		err = w.ModelVersions.SetTag(ctx, setTagReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start transition-stage command

var transitionStageReq mlflow.TransitionModelVersionStage

func init() {
	Cmd.AddCommand(transitionStageCmd)
	// TODO: short flags

}

var transitionStageCmd = &cobra.Command{
	Use:   "transition-stage NAME VERSION STAGE ARCHIVE_EXISTING_VERSIONS",
	Short: `Transition a stage.`,
	Long: `Transition a stage.
  
  Transition to the next model stage.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(4),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		transitionStageReq.Name = args[0]
		transitionStageReq.Version = args[1]
		transitionStageReq.Stage = args[2]
		_, err = fmt.Sscan(args[3], &transitionStageReq.ArchiveExistingVersions)
		if err != nil {
			return fmt.Errorf("invalid ARCHIVE_EXISTING_VERSIONS: %s", args[3])
		}

		response, err := w.ModelVersions.TransitionStage(ctx, transitionStageReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start update command

var updateReq mlflow.UpdateModelVersionRequest

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().StringVar(&updateReq.Description, "description", updateReq.Description, `If provided, updates the description for this registered_model.`)

}

var updateCmd = &cobra.Command{
	Use:   "update NAME VERSION",
	Short: `Update model version.`,
	Long: `Update model version.
  
  Updates the model version.`,

	Annotations: map[string]string{},
	Args:        cobra.ExactArgs(2),
	PreRunE:     sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		updateReq.Name = args[0]
		updateReq.Version = args[1]

		err = w.ModelVersions.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service ModelVersions
