// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package materialized_features

import (
	"errors"
	"fmt"

	"maps"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
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

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

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
	*map[string]any,
)

func newCreateFeatureTag() *cobra.Command {
	cmd := &cobra.Command{}
	var createFeatureTagReq map[string]any
	createFeatureTagReq = map[string]any{}
	var createFeatureTagFeatureNameRaw string
	var createFeatureTagTableNameRaw string
	var createFeatureTagKeyRaw string
	var createFeatureTagValueRaw string

	var createFeatureTagJson flags.JsonFlag

	cmd.Flags().Var(&createFeatureTagJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createFeatureTagValueRaw, "value", createFeatureTagValueRaw, ``)

	cmd.Use = "create-feature-tag TABLE_NAME FEATURE_NAME KEY"
	cmd.Short = `Create a feature tag.`
	cmd.Long = `Create a feature tag.

  Creates a FeatureTag.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return errors.New("when --json flag is specified, provide only TABLE_NAME, FEATURE_NAME as positional arguments. Provide 'key' in your JSON input")
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
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createFeatureTagJson.Unmarshal(&createFeatureTagReq)
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
		createFeatureTagTableNameRaw = args[0]
		createFeatureTagFeatureNameRaw = args[1]
		if !cmd.Flags().Changed("json") {
			createFeatureTagKeyRaw = args[2]
		}

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("key") || createFeatureTagKeyRaw != "" {
			createFeatureTagReq["key"] = createFeatureTagKeyRaw
		}
		if cmd.Flags().Changed("value") || createFeatureTagValueRaw != "" {
			createFeatureTagReq["value"] = createFeatureTagValueRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/feature-store/feature-tables/%s/features/%s/tags", fmt.Sprint(createFeatureTagTableNameRaw), fmt.Sprint(createFeatureTagFeatureNameRaw))

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "POST", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, createFeatureTagReq, &response)
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
	*map[string]any,
)

func newDeleteFeatureTag() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteFeatureTagReq map[string]any
	deleteFeatureTagReq = map[string]any{}
	var deleteFeatureTagFeatureNameRaw string
	var deleteFeatureTagKeyRaw string
	var deleteFeatureTagTableNameRaw string

	cmd.Use = "delete-feature-tag TABLE_NAME FEATURE_NAME KEY"
	cmd.Short = `Delete a feature tag.`
	cmd.Long = `Delete a feature tag.

  Deletes a FeatureTag.

  Arguments:
    TABLE_NAME: The name of the feature table.
    FEATURE_NAME: The name of the feature within the feature table.
    KEY: The key of the tag to delete.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteFeatureTagTableNameRaw = args[0]
		deleteFeatureTagFeatureNameRaw = args[1]
		deleteFeatureTagKeyRaw = args[2]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/feature-store/feature-tables/%s/features/%s/tags/%s", fmt.Sprint(deleteFeatureTagTableNameRaw), fmt.Sprint(deleteFeatureTagFeatureNameRaw), fmt.Sprint(deleteFeatureTagKeyRaw))

		maps.Copy(rawQuery, deleteFeatureTagReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		err = api.Do(ctx, "DELETE", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, nil)
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
	*map[string]any,
)

func newGetFeatureLineage() *cobra.Command {
	cmd := &cobra.Command{}
	var getFeatureLineageReq map[string]any
	getFeatureLineageReq = map[string]any{}
	var getFeatureLineageFeatureNameRaw string
	var getFeatureLineageTableNameRaw string

	cmd.Use = "get-feature-lineage TABLE_NAME FEATURE_NAME"
	cmd.Short = `Get Feature Lineage.`
	cmd.Long = `Get Feature Lineage.

  Arguments:
    TABLE_NAME: The full name of the feature table in Unity Catalog.
    FEATURE_NAME: The name of the feature.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getFeatureLineageTableNameRaw = args[0]
		getFeatureLineageFeatureNameRaw = args[1]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/feature-store/feature-tables/%s/features/%s/lineage", fmt.Sprint(getFeatureLineageTableNameRaw), fmt.Sprint(getFeatureLineageFeatureNameRaw))

		maps.Copy(rawQuery, getFeatureLineageReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
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
	*map[string]any,
)

func newGetFeatureTag() *cobra.Command {
	cmd := &cobra.Command{}
	var getFeatureTagReq map[string]any
	getFeatureTagReq = map[string]any{}
	var getFeatureTagFeatureNameRaw string
	var getFeatureTagKeyRaw string
	var getFeatureTagTableNameRaw string

	cmd.Use = "get-feature-tag TABLE_NAME FEATURE_NAME KEY"
	cmd.Short = `Get a feature tag.`
	cmd.Long = `Get a feature tag.

  Gets a FeatureTag.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getFeatureTagTableNameRaw = args[0]
		getFeatureTagFeatureNameRaw = args[1]
		getFeatureTagKeyRaw = args[2]

		rawQuery := map[string]any{}
		rawPath := fmt.Sprintf("/api/2.0/feature-store/feature-tables/%s/features/%s/tags/%s", fmt.Sprint(getFeatureTagTableNameRaw), fmt.Sprint(getFeatureTagFeatureNameRaw), fmt.Sprint(getFeatureTagKeyRaw))

		maps.Copy(rawQuery, getFeatureTagReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
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
	*map[string]any,
)

func newListFeatureTags() *cobra.Command {
	cmd := &cobra.Command{}
	var listFeatureTagsReq map[string]any
	listFeatureTagsReq = map[string]any{}
	var listFeatureTagsFeatureNameRaw string
	var listFeatureTagsPageSizeRaw int
	var listFeatureTagsPageTokenRaw string
	var listFeatureTagsTableNameRaw string

	// Registered for all paginated methods. Validated at call time in the
	// method-call template. Paginated list methods never have Wait or LRO
	// branches, so the method-call path is always reached.
	var listFeatureTagsLimit int

	cmd.Flags().IntVar(&listFeatureTagsPageSizeRaw, "page-size", listFeatureTagsPageSizeRaw, `The maximum number of results to return.`)

	// Limit flag for total result capping.
	cmd.Flags().IntVar(&listFeatureTagsLimit, "limit", 0, `Maximum number of results to return.`)

	// Hidden pagination flags (internal API parameters).
	cmd.Flags().StringVar(&listFeatureTagsPageTokenRaw, "page-token", listFeatureTagsPageTokenRaw, `Pagination token.`)
	cmd.Flags().Lookup("page-token").Hidden = true

	cmd.Use = "list-feature-tags TABLE_NAME FEATURE_NAME"
	cmd.Short = `List all feature tags.`
	cmd.Long = `List all feature tags.

  Lists FeatureTags.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		listFeatureTagsTableNameRaw = args[0]
		listFeatureTagsFeatureNameRaw = args[1]

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("page-size") || listFeatureTagsPageSizeRaw != 0 {
			rawQuery["page_size"] = listFeatureTagsPageSizeRaw
		}
		if cmd.Flags().Changed("page-token") || listFeatureTagsPageTokenRaw != "" {
			rawQuery["page_token"] = listFeatureTagsPageTokenRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/feature-store/feature-tables/%s/features/%s/tags", fmt.Sprint(listFeatureTagsTableNameRaw), fmt.Sprint(listFeatureTagsFeatureNameRaw))

		maps.Copy(rawQuery, listFeatureTagsReq)
		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "GET", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, nil, &response)
		if err != nil {
			return err
		}
		if listFeatureTagsLimit < 0 {
			return fmt.Errorf("--limit must be a non-negative integer, got %d", listFeatureTagsLimit)
		}
		return cmdio.Render(ctx, response)
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
	*map[string]any,
)

func newUpdateFeatureTag() *cobra.Command {
	cmd := &cobra.Command{}
	var updateFeatureTagReq map[string]any
	updateFeatureTagReq = map[string]any{}
	var updateFeatureTagFeatureNameRaw string
	var updateFeatureTagKeyRaw string
	var updateFeatureTagTableNameRaw string
	var updateFeatureTagUpdateMaskRaw string
	var updateFeatureTagValueRaw string

	var updateFeatureTagJson flags.JsonFlag

	cmd.Flags().Var(&updateFeatureTagJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateFeatureTagUpdateMaskRaw, "update-mask", updateFeatureTagUpdateMaskRaw, `The list of fields to update.`)
	cmd.Flags().StringVar(&updateFeatureTagValueRaw, "value", updateFeatureTagValueRaw, ``)

	cmd.Use = "update-feature-tag TABLE_NAME FEATURE_NAME KEY"
	cmd.Short = `Update a feature tag.`
	cmd.Long = `Update a feature tag.

  Updates a FeatureTag.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PRIVATE_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Private Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(3)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateFeatureTagJson.Unmarshal(&updateFeatureTagReq)
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
		updateFeatureTagTableNameRaw = args[0]
		updateFeatureTagFeatureNameRaw = args[1]
		updateFeatureTagKeyRaw = args[2]

		rawQuery := map[string]any{}
		if cmd.Flags().Changed("update-mask") || updateFeatureTagUpdateMaskRaw != "" {
			rawQuery["update_mask"] = updateFeatureTagUpdateMaskRaw
		}
		if cmd.Flags().Changed("key") || updateFeatureTagKeyRaw != "" {
			updateFeatureTagReq["key"] = updateFeatureTagKeyRaw
		}
		if cmd.Flags().Changed("value") || updateFeatureTagValueRaw != "" {
			updateFeatureTagReq["value"] = updateFeatureTagValueRaw
		}
		rawPath := fmt.Sprintf("/api/2.0/feature-store/feature-tables/%s/features/%s/tags/%s", fmt.Sprint(updateFeatureTagTableNameRaw), fmt.Sprint(updateFeatureTagFeatureNameRaw), fmt.Sprint(updateFeatureTagKeyRaw))

		api, err := apiclient.New(w.Config)
		if err != nil {
			return err
		}
		var response any
		err = api.Do(ctx, "PATCH", rawPath, auth.WorkspaceIDHeaders(w.Config), rawQuery, updateFeatureTagReq, &response)
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
