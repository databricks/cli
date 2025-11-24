// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package feature_engineering

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
		Use:     "feature-engineering",
		Short:   `[description].`,
		Long:    `[description]`,
		GroupID: "ml",
		Annotations: map[string]string{
			"package": "ml",
		},

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

	// Add methods
	cmd.AddCommand(newCreateFeature())
	cmd.AddCommand(newCreateMaterializedFeature())
	cmd.AddCommand(newDeleteFeature())
	cmd.AddCommand(newDeleteMaterializedFeature())
	cmd.AddCommand(newGetFeature())
	cmd.AddCommand(newGetMaterializedFeature())
	cmd.AddCommand(newListFeatures())
	cmd.AddCommand(newListMaterializedFeatures())
	cmd.AddCommand(newUpdateFeature())
	cmd.AddCommand(newUpdateMaterializedFeature())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-feature command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createFeatureOverrides []func(
	*cobra.Command,
	*ml.CreateFeatureRequest,
)

func newCreateFeature() *cobra.Command {
	cmd := &cobra.Command{}

	var createFeatureReq ml.CreateFeatureRequest
	createFeatureReq.Feature = ml.Feature{}
	var createFeatureJson flags.JsonFlag

	cmd.Flags().Var(&createFeatureJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&createFeatureReq.Feature.Description, "description", createFeatureReq.Feature.Description, `The description of the feature.`)
	cmd.Flags().StringVar(&createFeatureReq.Feature.FilterCondition, "filter-condition", createFeatureReq.Feature.FilterCondition, `The filter condition applied to the source data before aggregation.`)

	cmd.Use = "create-feature FULL_NAME SOURCE INPUTS FUNCTION TIME_WINDOW"
	cmd.Short = `Create a feature.`
	cmd.Long = `Create a feature.

  Create a Feature.

  Arguments:
    FULL_NAME: The full three-part name (catalog, schema, name) of the feature.
    SOURCE: The data source of the feature.
    INPUTS: The input columns from which the feature is computed.
    FUNCTION: The function by which the feature is computed.
    TIME_WINDOW: The time window in which the feature is computed.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'full_name', 'source', 'inputs', 'function', 'time_window' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(5)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createFeatureJson.Unmarshal(&createFeatureReq.Feature)
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
		if !cmd.Flags().Changed("json") {
			createFeatureReq.Feature.FullName = args[0]
		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[1], &createFeatureReq.Feature.Source)
			if err != nil {
				return fmt.Errorf("invalid SOURCE: %s", args[1])
			}

		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[2], &createFeatureReq.Feature.Inputs)
			if err != nil {
				return fmt.Errorf("invalid INPUTS: %s", args[2])
			}

		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[3], &createFeatureReq.Feature.Function)
			if err != nil {
				return fmt.Errorf("invalid FUNCTION: %s", args[3])
			}

		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[4], &createFeatureReq.Feature.TimeWindow)
			if err != nil {
				return fmt.Errorf("invalid TIME_WINDOW: %s", args[4])
			}

		}

		response, err := w.FeatureEngineering.CreateFeature(ctx, createFeatureReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createFeatureOverrides {
		fn(cmd, &createFeatureReq)
	}

	return cmd
}

// start create-materialized-feature command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createMaterializedFeatureOverrides []func(
	*cobra.Command,
	*ml.CreateMaterializedFeatureRequest,
)

func newCreateMaterializedFeature() *cobra.Command {
	cmd := &cobra.Command{}

	var createMaterializedFeatureReq ml.CreateMaterializedFeatureRequest
	createMaterializedFeatureReq.MaterializedFeature = ml.MaterializedFeature{}
	var createMaterializedFeatureJson flags.JsonFlag

	cmd.Flags().Var(&createMaterializedFeatureJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: offline_store_config
	// TODO: complex arg: online_store_config
	cmd.Flags().Var(&createMaterializedFeatureReq.MaterializedFeature.PipelineScheduleState, "pipeline-schedule-state", `The schedule state of the materialization pipeline. Supported values: [ACTIVE, PAUSED, SNAPSHOT]`)

	cmd.Use = "create-materialized-feature FEATURE_NAME"
	cmd.Short = `Create a materialized feature.`
	cmd.Long = `Create a materialized feature.

  Arguments:
    FEATURE_NAME: The full name of the feature in Unity Catalog.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(0)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, no positional arguments are required. Provide 'feature_name' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := createMaterializedFeatureJson.Unmarshal(&createMaterializedFeatureReq.MaterializedFeature)
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
		if !cmd.Flags().Changed("json") {
			createMaterializedFeatureReq.MaterializedFeature.FeatureName = args[0]
		}

		response, err := w.FeatureEngineering.CreateMaterializedFeature(ctx, createMaterializedFeatureReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createMaterializedFeatureOverrides {
		fn(cmd, &createMaterializedFeatureReq)
	}

	return cmd
}

// start delete-feature command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteFeatureOverrides []func(
	*cobra.Command,
	*ml.DeleteFeatureRequest,
)

func newDeleteFeature() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteFeatureReq ml.DeleteFeatureRequest

	cmd.Use = "delete-feature FULL_NAME"
	cmd.Short = `Delete a feature.`
	cmd.Long = `Delete a feature.

  Delete a Feature.

  Arguments:
    FULL_NAME: Name of the feature to delete.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteFeatureReq.FullName = args[0]

		err = w.FeatureEngineering.DeleteFeature(ctx, deleteFeatureReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteFeatureOverrides {
		fn(cmd, &deleteFeatureReq)
	}

	return cmd
}

// start delete-materialized-feature command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteMaterializedFeatureOverrides []func(
	*cobra.Command,
	*ml.DeleteMaterializedFeatureRequest,
)

func newDeleteMaterializedFeature() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteMaterializedFeatureReq ml.DeleteMaterializedFeatureRequest

	cmd.Use = "delete-materialized-feature MATERIALIZED_FEATURE_ID"
	cmd.Short = `Delete a materialized feature.`
	cmd.Long = `Delete a materialized feature.

  Arguments:
    MATERIALIZED_FEATURE_ID: The ID of the materialized feature to delete.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteMaterializedFeatureReq.MaterializedFeatureId = args[0]

		err = w.FeatureEngineering.DeleteMaterializedFeature(ctx, deleteMaterializedFeatureReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteMaterializedFeatureOverrides {
		fn(cmd, &deleteMaterializedFeatureReq)
	}

	return cmd
}

// start get-feature command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getFeatureOverrides []func(
	*cobra.Command,
	*ml.GetFeatureRequest,
)

func newGetFeature() *cobra.Command {
	cmd := &cobra.Command{}

	var getFeatureReq ml.GetFeatureRequest

	cmd.Use = "get-feature FULL_NAME"
	cmd.Short = `Get a feature.`
	cmd.Long = `Get a feature.

  Get a Feature.

  Arguments:
    FULL_NAME: Name of the feature to get.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getFeatureReq.FullName = args[0]

		response, err := w.FeatureEngineering.GetFeature(ctx, getFeatureReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getFeatureOverrides {
		fn(cmd, &getFeatureReq)
	}

	return cmd
}

// start get-materialized-feature command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getMaterializedFeatureOverrides []func(
	*cobra.Command,
	*ml.GetMaterializedFeatureRequest,
)

func newGetMaterializedFeature() *cobra.Command {
	cmd := &cobra.Command{}

	var getMaterializedFeatureReq ml.GetMaterializedFeatureRequest

	cmd.Use = "get-materialized-feature MATERIALIZED_FEATURE_ID"
	cmd.Short = `Get a materialized feature.`
	cmd.Long = `Get a materialized feature.

  Arguments:
    MATERIALIZED_FEATURE_ID: The ID of the materialized feature.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getMaterializedFeatureReq.MaterializedFeatureId = args[0]

		response, err := w.FeatureEngineering.GetMaterializedFeature(ctx, getMaterializedFeatureReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getMaterializedFeatureOverrides {
		fn(cmd, &getMaterializedFeatureReq)
	}

	return cmd
}

// start list-features command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listFeaturesOverrides []func(
	*cobra.Command,
	*ml.ListFeaturesRequest,
)

func newListFeatures() *cobra.Command {
	cmd := &cobra.Command{}

	var listFeaturesReq ml.ListFeaturesRequest

	cmd.Flags().IntVar(&listFeaturesReq.PageSize, "page-size", listFeaturesReq.PageSize, `The maximum number of results to return.`)
	cmd.Flags().StringVar(&listFeaturesReq.PageToken, "page-token", listFeaturesReq.PageToken, `Pagination token to go to the next page based on a previous query.`)

	cmd.Use = "list-features"
	cmd.Short = `List features.`
	cmd.Long = `List features.

  List Features.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.FeatureEngineering.ListFeatures(ctx, listFeaturesReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listFeaturesOverrides {
		fn(cmd, &listFeaturesReq)
	}

	return cmd
}

// start list-materialized-features command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var listMaterializedFeaturesOverrides []func(
	*cobra.Command,
	*ml.ListMaterializedFeaturesRequest,
)

func newListMaterializedFeatures() *cobra.Command {
	cmd := &cobra.Command{}

	var listMaterializedFeaturesReq ml.ListMaterializedFeaturesRequest

	cmd.Flags().StringVar(&listMaterializedFeaturesReq.FeatureName, "feature-name", listMaterializedFeaturesReq.FeatureName, `Filter by feature name.`)
	cmd.Flags().IntVar(&listMaterializedFeaturesReq.PageSize, "page-size", listMaterializedFeaturesReq.PageSize, `The maximum number of results to return.`)
	cmd.Flags().StringVar(&listMaterializedFeaturesReq.PageToken, "page-token", listMaterializedFeaturesReq.PageToken, `Pagination token to go to the next page based on a previous query.`)

	cmd.Use = "list-materialized-features"
	cmd.Short = `List materialized features.`
	cmd.Long = `List materialized features.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(0)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		response := w.FeatureEngineering.ListMaterializedFeatures(ctx, listMaterializedFeaturesReq)
		return cmdio.RenderIterator(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range listMaterializedFeaturesOverrides {
		fn(cmd, &listMaterializedFeaturesReq)
	}

	return cmd
}

// start update-feature command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateFeatureOverrides []func(
	*cobra.Command,
	*ml.UpdateFeatureRequest,
)

func newUpdateFeature() *cobra.Command {
	cmd := &cobra.Command{}

	var updateFeatureReq ml.UpdateFeatureRequest
	updateFeatureReq.Feature = ml.Feature{}
	var updateFeatureJson flags.JsonFlag

	cmd.Flags().Var(&updateFeatureJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	cmd.Flags().StringVar(&updateFeatureReq.Feature.Description, "description", updateFeatureReq.Feature.Description, `The description of the feature.`)
	cmd.Flags().StringVar(&updateFeatureReq.Feature.FilterCondition, "filter-condition", updateFeatureReq.Feature.FilterCondition, `The filter condition applied to the source data before aggregation.`)

	cmd.Use = "update-feature FULL_NAME UPDATE_MASK SOURCE INPUTS FUNCTION TIME_WINDOW"
	cmd.Short = `Update a feature's description (all other fields are immutable).`
	cmd.Long = `Update a feature's description (all other fields are immutable).

  Update a Feature.

  Arguments:
    FULL_NAME: The full three-part name (catalog, schema, name) of the feature.
    UPDATE_MASK: The list of fields to update.
    SOURCE: The data source of the feature.
    INPUTS: The input columns from which the feature is computed.
    FUNCTION: The function by which the feature is computed.
    TIME_WINDOW: The time window in which the feature is computed.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only FULL_NAME, UPDATE_MASK as positional arguments. Provide 'full_name', 'source', 'inputs', 'function', 'time_window' in your JSON input")
			}
			return nil
		}
		check := root.ExactArgs(6)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if cmd.Flags().Changed("json") {
			diags := updateFeatureJson.Unmarshal(&updateFeatureReq.Feature)
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
		updateFeatureReq.FullName = args[0]
		updateFeatureReq.UpdateMask = args[1]
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[2], &updateFeatureReq.Feature.Source)
			if err != nil {
				return fmt.Errorf("invalid SOURCE: %s", args[2])
			}

		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[3], &updateFeatureReq.Feature.Inputs)
			if err != nil {
				return fmt.Errorf("invalid INPUTS: %s", args[3])
			}

		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[4], &updateFeatureReq.Feature.Function)
			if err != nil {
				return fmt.Errorf("invalid FUNCTION: %s", args[4])
			}

		}
		if !cmd.Flags().Changed("json") {
			_, err = fmt.Sscan(args[5], &updateFeatureReq.Feature.TimeWindow)
			if err != nil {
				return fmt.Errorf("invalid TIME_WINDOW: %s", args[5])
			}

		}

		response, err := w.FeatureEngineering.UpdateFeature(ctx, updateFeatureReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateFeatureOverrides {
		fn(cmd, &updateFeatureReq)
	}

	return cmd
}

// start update-materialized-feature command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateMaterializedFeatureOverrides []func(
	*cobra.Command,
	*ml.UpdateMaterializedFeatureRequest,
)

func newUpdateMaterializedFeature() *cobra.Command {
	cmd := &cobra.Command{}

	var updateMaterializedFeatureReq ml.UpdateMaterializedFeatureRequest
	updateMaterializedFeatureReq.MaterializedFeature = ml.MaterializedFeature{}
	var updateMaterializedFeatureJson flags.JsonFlag

	cmd.Flags().Var(&updateMaterializedFeatureJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: complex arg: offline_store_config
	// TODO: complex arg: online_store_config
	cmd.Flags().Var(&updateMaterializedFeatureReq.MaterializedFeature.PipelineScheduleState, "pipeline-schedule-state", `The schedule state of the materialization pipeline. Supported values: [ACTIVE, PAUSED, SNAPSHOT]`)

	cmd.Use = "update-materialized-feature MATERIALIZED_FEATURE_ID UPDATE_MASK FEATURE_NAME"
	cmd.Short = `Update a materialized feature.`
	cmd.Long = `Update a materialized feature.

  Update a materialized feature (pause/resume).

  Arguments:
    MATERIALIZED_FEATURE_ID: Unique identifier for the materialized feature.
    UPDATE_MASK: Provide the materialization feature fields which should be updated.
      Currently, only the pipeline_state field can be updated.
    FEATURE_NAME: The full name of the feature in Unity Catalog.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("json") {
			err := root.ExactArgs(2)(cmd, args)
			if err != nil {
				return fmt.Errorf("when --json flag is specified, provide only MATERIALIZED_FEATURE_ID, UPDATE_MASK as positional arguments. Provide 'feature_name' in your JSON input")
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
			diags := updateMaterializedFeatureJson.Unmarshal(&updateMaterializedFeatureReq.MaterializedFeature)
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
		updateMaterializedFeatureReq.MaterializedFeatureId = args[0]
		updateMaterializedFeatureReq.UpdateMask = args[1]
		if !cmd.Flags().Changed("json") {
			updateMaterializedFeatureReq.MaterializedFeature.FeatureName = args[2]
		}

		response, err := w.FeatureEngineering.UpdateMaterializedFeature(ctx, updateMaterializedFeatureReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateMaterializedFeatureOverrides {
		fn(cmd, &updateMaterializedFeatureReq)
	}

	return cmd
}

// end service FeatureEngineering
