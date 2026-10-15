// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package data_classification

import (
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"

	dataclassification "github.com/databricks/cli/libs/sdktypes/dataclassification/v1"
	sdkdataclassification "github.com/databricks/sdk-go/dataclassification/v1"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "data-classification",
		Short: `*Public Preview* Manage data classification for Unity Catalog catalogs.`,
		Long: `This command is in Public Preview and may change without notice.

Manage data classification for Unity Catalog catalogs. Data classification
  automatically identifies and tags sensitive data (PII) in Unity Catalog
  tables. Each catalog can have at most one configuration resource that controls
  scanning behavior and auto-tagging rules.`,
		GroupID: "dataclassification",
		RunE:    root.ReportUnknownSubcommand,
	}

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	// Add methods
	cmd.AddCommand(newCreateCatalogConfig())
	cmd.AddCommand(newDeleteCatalogConfig())
	cmd.AddCommand(newGetCatalogConfig())
	cmd.AddCommand(newUpdateCatalogConfig())

	// Apply optional overrides to this command.
	for _, fn := range cmdOverrides {
		fn(cmd)
	}

	return cmd
}

// start create-catalog-config command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var createCatalogConfigOverrides []func(
	*cobra.Command,
	*dataclassification.CreateCatalogConfigRequest,
)

func newCreateCatalogConfig() *cobra.Command {
	cmd := &cobra.Command{}
	var createCatalogConfigReq dataclassification.CreateCatalogConfigRequest
	createCatalogConfigReq.CatalogConfig = &dataclassification.CatalogConfig{}

	var createCatalogConfigJson flags.JsonFlag

	cmd.Flags().Var(&createCatalogConfigJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: auto_tag_configs
	// TODO: complex arg: excluded_schemas
	// TODO: complex arg: included_schemas
	flags.StringPtrVar(cmd.Flags(), &createCatalogConfigReq.CatalogConfig.Name, "name", `Resource name in the format: catalogs/{catalog_name}/config.`)

	cmd.Use = "create-catalog-config PARENT"
	cmd.Short = `*Public Preview* Create config for a catalog.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Create config for a catalog.

  Create Data Classification configuration for a catalog.

  Creates a new config resource, which enables Data Classification for the
  specified catalog. - The config must not already exist for the catalog.

  Arguments:
    PARENT: Parent resource in the format: catalogs/{catalog_name}`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := createCatalogConfigJson.Unmarshal(&createCatalogConfigReq.CatalogConfig)
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
		createCatalogConfigReq.Parent = new(args[0])

		sdkReq, err := dataclassification.CreateCatalogConfigRequestToSDK(&createCatalogConfigReq)
		if err != nil {
			return err
		}
		response, err := w.DataClassification.CreateCatalogConfig(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := dataclassification.CatalogConfigFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range createCatalogConfigOverrides {
		fn(cmd, &createCatalogConfigReq)
	}

	return cmd
}

// start delete-catalog-config command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var deleteCatalogConfigOverrides []func(
	*cobra.Command,
	*sdkdataclassification.DeleteCatalogConfigRequest,
)

func newDeleteCatalogConfig() *cobra.Command {
	cmd := &cobra.Command{}
	var deleteCatalogConfigReq sdkdataclassification.DeleteCatalogConfigRequest

	cmd.Use = "delete-catalog-config NAME"
	cmd.Short = `*Public Preview* Delete config for a catalog.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Delete config for a catalog.

  Delete Data Classification configuration for a catalog.

  Arguments:
    NAME: Resource name in the format: catalogs/{catalog_name}/config`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		deleteCatalogConfigReq.Name = new(args[0])

		sdkReq := &deleteCatalogConfigReq
		err = w.DataClassification.DeleteCatalogConfig(ctx, *sdkReq)
		if err != nil {
			return err
		}
		return nil
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range deleteCatalogConfigOverrides {
		fn(cmd, &deleteCatalogConfigReq)
	}

	return cmd
}

// start get-catalog-config command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var getCatalogConfigOverrides []func(
	*cobra.Command,
	*sdkdataclassification.GetCatalogConfigRequest,
)

func newGetCatalogConfig() *cobra.Command {
	cmd := &cobra.Command{}
	var getCatalogConfigReq sdkdataclassification.GetCatalogConfigRequest

	cmd.Use = "get-catalog-config NAME"
	cmd.Short = `*Public Preview* Get config for a catalog.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Get config for a catalog.

  Get the Data Classification configuration for a catalog.

  Arguments:
    NAME: Resource name in the format: catalogs/{catalog_name}/config`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		getCatalogConfigReq.Name = new(args[0])

		sdkReq := &getCatalogConfigReq
		response, err := w.DataClassification.GetCatalogConfig(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := dataclassification.CatalogConfigFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range getCatalogConfigOverrides {
		fn(cmd, &getCatalogConfigReq)
	}

	return cmd
}

// start update-catalog-config command

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var updateCatalogConfigOverrides []func(
	*cobra.Command,
	*dataclassification.UpdateCatalogConfigRequest,
)

func newUpdateCatalogConfig() *cobra.Command {
	cmd := &cobra.Command{}
	var updateCatalogConfigReq dataclassification.UpdateCatalogConfigRequest
	updateCatalogConfigReq.CatalogConfig = &dataclassification.CatalogConfig{}

	var updateCatalogConfigJson flags.JsonFlag

	cmd.Flags().Var(&updateCatalogConfigJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: auto_tag_configs
	// TODO: complex arg: excluded_schemas
	// TODO: complex arg: included_schemas
	flags.StringPtrVar(cmd.Flags(), &updateCatalogConfigReq.CatalogConfig.Name, "name", `Resource name in the format: catalogs/{catalog_name}/config.`)

	cmd.Use = "update-catalog-config NAME UPDATE_MASK"
	cmd.Short = `*Public Preview* Update config for a catalog.`
	cmd.Long = `This command is in Public Preview and may change without notice.

Update config for a catalog.

  Update the Data Classification configuration for a catalog. - The config must
  already exist for the catalog. - Updates fields specified in the update_mask.
  Use update_mask field to perform partial updates of the configuration.

  Arguments:
    NAME: Resource name in the format: catalogs/{catalog_name}/config.
    UPDATE_MASK: Field mask specifying which fields to update.`

	cmd.Annotations = make(map[string]string)
	cmd.Annotations["launch_stage"] = "PUBLIC_PREVIEW"
	cmd.Annotations["launch_stage_display"] = "Public Preview"

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		_ = w

		if cmd.Flags().Changed("json") {
			diags := updateCatalogConfigJson.Unmarshal(&updateCatalogConfigReq.CatalogConfig)
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
		updateCatalogConfigReq.CatalogConfig.Name = new(args[0])
		if args[1] != "" {
			updateCatalogConfigReq.UpdateMask = new(args[1])
		}

		sdkReq, err := dataclassification.UpdateCatalogConfigRequestToSDK(&updateCatalogConfigReq)
		if err != nil {
			return err
		}
		response, err := w.DataClassification.UpdateCatalogConfig(ctx, *sdkReq)
		if err != nil {
			return err
		}

		out, err := dataclassification.CatalogConfigFromSDK(response)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, out)
	}

	// Disable completions since they are not applicable.
	// Can be overridden by manual implementation in `override.go`.
	cmd.ValidArgsFunction = cobra.NoFileCompletions

	// Apply optional overrides to this command.
	for _, fn := range updateCatalogConfigOverrides {
		fn(cmd, &updateCatalogConfigReq)
	}

	return cmd
}

// end service DataClassification
