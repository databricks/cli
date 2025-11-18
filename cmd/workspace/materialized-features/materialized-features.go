// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package materialized_features

import (
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "materialized-features",
		Short: `Materialized Features are columns in tables and views that can be directly used as features to train and serve ML models.`,
		Long: `Materialized Features are columns in tables and views that can be directly
  used as features to train and serve ML models.`,
		GroupID: "ml",
		Annotations: map[string]string{
			"package": "ml",
		},

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreateFeatureTag())
	cmd.AddCommand(newDeleteFeatureTag())
	cmd.AddCommand(newGetFeatureLineage())
	cmd.AddCommand(newGetFeatureTag())
	cmd.AddCommand(newListFeatureTags())
	cmd.AddCommand(newUpdateFeatureTag())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-feature-tag command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createFeatureTagOverrides []func(
	*cobra.Command,
	*ml.CreateFeatureTagRequest,
)

func newCreateFeatureTag() *cobra.Command {
	cmd := &cobra.Command{}

	var createFeatureTagReq ml.CreateFeatureTagRequest
	createFeatureTagReq.FeatureTag = ml.FeatureTag{}
	var createFeatureTagJson flags.JsonFlag

	cmd.Flags().Var(&createFeatureTagJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createFeatureTagReq.FeatureTag.Value, "value", createFeatureTagReq.FeatureTag.Value, ``)

	cmd.Use = "create-feature-tag TABLE_NAME FEATURE_NAME KEY"
	cmd.Short = `Create a feature tag.`
	cmd.Long = `Create a feature tag.
  
  Creates a FeatureTag.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only TABLE_NAME, FEATURE_NAME as positional arguments. Provide 'key' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createFeatureTagJson.Unmarshal(&createFeatureTagReq.FeatureTag)
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
		createFeatureTagReq.TableName = args[0]
		createFeatureTagReq.FeatureName = args[1]
		if !cmd.Flags().Changed("json") {
			createFeatureTagReq.FeatureTag.Key = args[2]
		}

		response, err := w.MaterializedFeatures.CreateFeatureTag(ctx, createFeatureTagReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createFeatureTagOverrides {
		fn(cmd, &createFeatureTagReq)
	}

	return cmd
}

// start delete-feature-tag command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteFeatureTagOverrides []func(
	*cobra.Command,
	*ml.DeleteFeatureTagRequest,
)

func newDeleteFeatureTag() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteFeatureTagReq ml.DeleteFeatureTagRequest

	cmd.Use = "delete-feature-tag TABLE_NAME FEATURE_NAME KEY"
	cmd.Short = `Delete a feature tag.`
	cmd.Long = `Delete a feature tag.
  
  Deletes a FeatureTag.

  Arguments:
    TABLE_NAME: The name of the feature table.
    FEATURE_NAME: The name of the feature within the feature table.
    KEY: The key of the tag to delete.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteFeatureTagReq.TableName = args[0]
		deleteFeatureTagReq.FeatureName = args[1]
		deleteFeatureTagReq.Key = args[2]

		err = w.MaterializedFeatures.DeleteFeatureTag(ctx, deleteFeatureTagReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteFeatureTagOverrides {
		fn(cmd, &deleteFeatureTagReq)
	}

	return cmd
}

// start get-feature-lineage command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getFeatureLineageOverrides []func(
	*cobra.Command,
	*ml.GetFeatureLineageRequest,
)

func newGetFeatureLineage() *cobra.Command {
	cmd := &cobra.Command{}

	var getFeatureLineageReq ml.GetFeatureLineageRequest

	cmd.Use = "get-feature-lineage TABLE_NAME FEATURE_NAME"
	cmd.Short = `Get Feature Lineage.`
	cmd.Long = `Get Feature Lineage.

  Arguments:
    TABLE_NAME: The full name of the feature table in Unity Catalog.
    FEATURE_NAME: The name of the feature.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getFeatureLineageReq.TableName = args[0]
		getFeatureLineageReq.FeatureName = args[1]

		response, err := w.MaterializedFeatures.GetFeatureLineage(ctx, getFeatureLineageReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getFeatureLineageOverrides {
		fn(cmd, &getFeatureLineageReq)
	}

	return cmd
}

// start get-feature-tag command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getFeatureTagOverrides []func(
	*cobra.Command,
	*ml.GetFeatureTagRequest,
)

func newGetFeatureTag() *cobra.Command {
	cmd := &cobra.Command{}

	var getFeatureTagReq ml.GetFeatureTagRequest

	cmd.Use = "get-feature-tag TABLE_NAME FEATURE_NAME KEY"
	cmd.Short = `Get a feature tag.`
	cmd.Long = `Get a feature tag.
  
  Gets a FeatureTag.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getFeatureTagReq.TableName = args[0]
		getFeatureTagReq.FeatureName = args[1]
		getFeatureTagReq.Key = args[2]

		response, err := w.MaterializedFeatures.GetFeatureTag(ctx, getFeatureTagReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getFeatureTagOverrides {
		fn(cmd, &getFeatureTagReq)
	}

	return cmd
}

// start list-feature-tags command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listFeatureTagsOverrides []func(
	*cobra.Command,
	*ml.ListFeatureTagsRequest,
)

func newListFeatureTags() *cobra.Command {
	cmd := &cobra.Command{}

	var listFeatureTagsReq ml.ListFeatureTagsRequest

	cmd.Flags().IntVar(&listFeatureTagsReq.PageSize, "page-size", listFeatureTagsReq.PageSize, `The maximum number of results to return.`)
	cmd.Flags().StringVar(&listFeatureTagsReq.PageToken, "page-token", listFeatureTagsReq.PageToken, `Pagination token to go to the next page based on a previous query.`)

	cmd.Use = "list-feature-tags TABLE_NAME FEATURE_NAME"
	cmd.Short = `List all feature tags.`
	cmd.Long = `List all feature tags.
  
  Lists FeatureTags.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		listFeatureTagsReq.TableName = args[0]
		listFeatureTagsReq.FeatureName = args[1]

		response := w.MaterializedFeatures.ListFeatureTags(ctx, listFeatureTagsReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listFeatureTagsOverrides {
		fn(cmd, &listFeatureTagsReq)
	}

	return cmd
}

// start update-feature-tag command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateFeatureTagOverrides []func(
	*cobra.Command,
	*ml.UpdateFeatureTagRequest,
)

func newUpdateFeatureTag() *cobra.Command {
	cmd := &cobra.Command{}

	var updateFeatureTagReq ml.UpdateFeatureTagRequest
	updateFeatureTagReq.FeatureTag = ml.FeatureTag{}
	var updateFeatureTagJson flags.JsonFlag

	cmd.Flags().Var(&updateFeatureTagJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateFeatureTagReq.UpdateMask, "update-mask", updateFeatureTagReq.UpdateMask, `The list of fields to update.`)
	cmd.Flags().StringVar(&updateFeatureTagReq.FeatureTag.Value, "value", updateFeatureTagReq.FeatureTag.Value, ``)

	cmd.Use = "update-feature-tag TABLE_NAME FEATURE_NAME KEY"
	cmd.Short = `Update a feature tag.`
	cmd.Long = `Update a feature tag.
  
  Updates a FeatureTag.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateFeatureTagJson.Unmarshal(&updateFeatureTagReq.FeatureTag)
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
		updateFeatureTagReq.TableName = args[0]
		updateFeatureTagReq.FeatureName = args[1]
		updateFeatureTagReq.Key = args[2]

		response, err := w.MaterializedFeatures.UpdateFeatureTag(ctx, updateFeatureTagReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateFeatureTagOverrides {
		fn(cmd, &updateFeatureTagReq)
	}

	return cmd
}

// end service MaterializedFeatures
