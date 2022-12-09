package registered_models

import (
	"github.com/databricks/bricks/lib/jsonflag"
	"github.com/databricks/bricks/lib/sdk"
	"github.com/databricks/bricks/lib/ui"
	"github.com/databricks/databricks-sdk-go/service/mlflow"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use: "registered-models",
}

// start create command

var createReq mlflow.CreateRegisteredModelRequest
var createJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(createCmd)
	// TODO: short flags
	createCmd.Flags().Var(&createJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	createCmd.Flags().StringVar(&createReq.Description, "description", createReq.Description, `Optional description for registered model.`)
	createCmd.Flags().StringVar(&createReq.Name, "name", createReq.Name, `Register models under this name.`)
	// TODO: array: tags

}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: `Create a model.`,
	Long: `Create a model.
  
  Creates a new registered model with the name specified in the request body.
  
  Throws RESOURCE_ALREADY_EXISTS if a registered model with the given name
  exists.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = createJson.Unmarshall(&createReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.RegisteredModels.Create(ctx, createReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start delete command

var deleteReq mlflow.DeleteRegisteredModelRequest

func init() {
	Cmd.AddCommand(deleteCmd)
	// TODO: short flags

	deleteCmd.Flags().StringVar(&deleteReq.Name, "name", deleteReq.Name, `Registered model unique name identifier.`)

}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: `Delete a model.`,
	Long: `Delete a model.
  
  Deletes a registered model.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.RegisteredModels.Delete(ctx, deleteReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start delete-tag command

var deleteTagReq mlflow.DeleteRegisteredModelTagRequest

func init() {
	Cmd.AddCommand(deleteTagCmd)
	// TODO: short flags

	deleteTagCmd.Flags().StringVar(&deleteTagReq.Key, "key", deleteTagReq.Key, `Name of the tag.`)
	deleteTagCmd.Flags().StringVar(&deleteTagReq.Name, "name", deleteTagReq.Name, `Name of the registered model that the tag was logged under.`)

}

var deleteTagCmd = &cobra.Command{
	Use:   "delete-tag",
	Short: `Delete a model tag.`,
	Long: `Delete a model tag.
  
  Deletes the tag for a registered model.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.RegisteredModels.DeleteTag(ctx, deleteTagReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start get command

var getReq mlflow.GetRegisteredModelRequest

func init() {
	Cmd.AddCommand(getCmd)
	// TODO: short flags

	getCmd.Flags().StringVar(&getReq.Name, "name", getReq.Name, `Registered model unique name identifier.`)

}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: `Get a model.`,
	Long: `Get a model.
  
  Gets the registered model that matches the specified ID.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.RegisteredModels.Get(ctx, getReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start get-latest-versions command

var getLatestVersionsReq mlflow.GetLatestVersionsRequest
var getLatestVersionsJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(getLatestVersionsCmd)
	// TODO: short flags
	getLatestVersionsCmd.Flags().Var(&getLatestVersionsJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	getLatestVersionsCmd.Flags().StringVar(&getLatestVersionsReq.Name, "name", getLatestVersionsReq.Name, `Registered model unique name identifier.`)
	// TODO: array: stages

}

var getLatestVersionsCmd = &cobra.Command{
	Use:   "get-latest-versions",
	Short: `Get the latest version.`,
	Long: `Get the latest version.
  
  Gets the latest version of a registered model.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = getLatestVersionsJson.Unmarshall(&getLatestVersionsReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.RegisteredModels.GetLatestVersionsAll(ctx, getLatestVersionsReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start list command

var listReq mlflow.ListRegisteredModelsRequest

func init() {
	Cmd.AddCommand(listCmd)
	// TODO: short flags

	listCmd.Flags().IntVar(&listReq.MaxResults, "max-results", listReq.MaxResults, `Maximum number of registered models desired.`)
	listCmd.Flags().StringVar(&listReq.PageToken, "page-token", listReq.PageToken, `Pagination token to go to the next page based on a previous query.`)

}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: `List models.`,
	Long: `List models.
  
  Lists all available registered models, up to the limit specified in
  __max_results__.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.RegisteredModels.ListAll(ctx, listReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start rename command

var renameReq mlflow.RenameRegisteredModelRequest

func init() {
	Cmd.AddCommand(renameCmd)
	// TODO: short flags

	renameCmd.Flags().StringVar(&renameReq.Name, "name", renameReq.Name, `Registered model unique name identifier.`)
	renameCmd.Flags().StringVar(&renameReq.NewName, "new-name", renameReq.NewName, `If provided, updates the name for this registered_model.`)

}

var renameCmd = &cobra.Command{
	Use:   "rename",
	Short: `Rename a model.`,
	Long: `Rename a model.
  
  Renames a registered model.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.RegisteredModels.Rename(ctx, renameReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start search command

var searchReq mlflow.SearchRegisteredModelsRequest
var searchJson jsonflag.JsonFlag

func init() {
	Cmd.AddCommand(searchCmd)
	// TODO: short flags
	searchCmd.Flags().Var(&searchJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	searchCmd.Flags().StringVar(&searchReq.Filter, "filter", searchReq.Filter, `String filter condition, like "name LIKE 'my-model-name'".`)
	searchCmd.Flags().IntVar(&searchReq.MaxResults, "max-results", searchReq.MaxResults, `Maximum number of models desired.`)
	// TODO: array: order_by
	searchCmd.Flags().StringVar(&searchReq.PageToken, "page-token", searchReq.PageToken, `Pagination token to go to the next page based on a previous search query.`)

}

var searchCmd = &cobra.Command{
	Use:   "search",
	Short: `Search models.`,
	Long: `Search models.
  
  Search for registered models based on the specified __filter__.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		err = searchJson.Unmarshall(&searchReq)
		if err != nil {
			return err
		}
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		response, err := w.RegisteredModels.SearchAll(ctx, searchReq)
		if err != nil {
			return err
		}
		return ui.Render(cmd, response)
	},
}

// start set-tag command

var setTagReq mlflow.SetRegisteredModelTagRequest

func init() {
	Cmd.AddCommand(setTagCmd)
	// TODO: short flags

	setTagCmd.Flags().StringVar(&setTagReq.Key, "key", setTagReq.Key, `Name of the tag.`)
	setTagCmd.Flags().StringVar(&setTagReq.Name, "name", setTagReq.Name, `Unique name of the model.`)
	setTagCmd.Flags().StringVar(&setTagReq.Value, "value", setTagReq.Value, `String value of the tag being logged.`)

}

var setTagCmd = &cobra.Command{
	Use:   "set-tag",
	Short: `Set a tag.`,
	Long: `Set a tag.
  
  Sets a tag on a registered model.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.RegisteredModels.SetTag(ctx, setTagReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// start update command

var updateReq mlflow.UpdateRegisteredModelRequest

func init() {
	Cmd.AddCommand(updateCmd)
	// TODO: short flags

	updateCmd.Flags().StringVar(&updateReq.Description, "description", updateReq.Description, `If provided, updates the description for this registered_model.`)
	updateCmd.Flags().StringVar(&updateReq.Name, "name", updateReq.Name, `Registered model unique name identifier.`)

}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: `Update model.`,
	Long: `Update model.
  
  Updates a registered model.`,

	PreRunE: sdk.PreWorkspaceClient,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := sdk.WorkspaceClient(ctx)
		err = w.RegisteredModels.Update(ctx, updateReq)
		if err != nil {
			return err
		}
		return nil
	},
}

// end service RegisteredModels
