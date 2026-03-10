// Code generated from OpenAPI specs by Databricks SDK Generator. DO NOT EDIT.

package data_classification

import (
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/service/dataclassification"
	"github.com/spf13/cobra"
)

// Slice with functions to override default command behavior.
// Functions can be added from the `init()` function in manually curated files in this directory.
var cmdOverrides []func(*cobra.Command)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "data-classification",
		Short: `Manage data classification for Unity Catalog catalogs.`,
		Long: `Manage data classification for Unity Catalog catalogs. Data classification
  automatically identifies and tags sensitive data (PII) in Unity Catalog
  tables. Each catalog can have at most one configuration resource that controls
  scanning behavior and auto-tagging rules.`,
		GroupID: "dataclassification",

		// This service is being previewed; hide from help output.
		Hidden: true,
		RunE:   root.ReportUnknownSubcommand,
	}

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
	createCatalogConfigReq.CatalogConfig = dataclassification.CatalogConfig{}
	var createCatalogConfigJson flags.JsonFlag

	cmd.Flags().Var(&createCatalogConfigJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: auto_tag_configs
	// TODO: complex arg: included_schemas
	cmd.Flags().StringVar(&createCatalogConfigReq.CatalogConfig.Name, "name", createCatalogConfigReq.CatalogConfig.Name, `Resource name in the format: catalogs/{catalog_name}/config.`)

	cmd.Use = "create-catalog-config PARENT"
	cmd.Short = `Create config for a catalog.`
	cmd.Long = `Create config for a catalog.
  
  Create Data Classification configuration for a catalog.
  
  Creates a new config resource, which enables Data Classification for the
  specified catalog. - The config must not already exist for the catalog.

  Arguments:
    PARENT: Parent resource in the format: catalogs/{catalog_name}`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

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
		createCatalogConfigReq.Parent = args[0]

		response, err := w.DataClassification.CreateCatalogConfig(ctx, createCatalogConfigReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	*dataclassification.DeleteCatalogConfigRequest,
)

func newDeleteCatalogConfig() *cobra.Command {
	cmd := &cobra.Command{}

	var deleteCatalogConfigReq dataclassification.DeleteCatalogConfigRequest

	cmd.Use = "delete-catalog-config NAME"
	cmd.Short = `Delete config for a catalog.`
	cmd.Long = `Delete config for a catalog.
  
  Delete Data Classification configuration for a catalog.

  Arguments:
    NAME: Resource name in the format: catalogs/{catalog_name}/config`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		deleteCatalogConfigReq.Name = args[0]

		err = w.DataClassification.DeleteCatalogConfig(ctx, deleteCatalogConfigReq)
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
	*dataclassification.GetCatalogConfigRequest,
)

func newGetCatalogConfig() *cobra.Command {
	cmd := &cobra.Command{}

	var getCatalogConfigReq dataclassification.GetCatalogConfigRequest

	cmd.Use = "get-catalog-config NAME"
	cmd.Short = `Get config for a catalog.`
	cmd.Long = `Get config for a catalog.
  
  Get the Data Classification configuration for a catalog.

  Arguments:
    NAME: Resource name in the format: catalogs/{catalog_name}/config`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(1)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		getCatalogConfigReq.Name = args[0]

		response, err := w.DataClassification.GetCatalogConfig(ctx, getCatalogConfigReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
	updateCatalogConfigReq.CatalogConfig = dataclassification.CatalogConfig{}
	var updateCatalogConfigJson flags.JsonFlag

	cmd.Flags().Var(&updateCatalogConfigJson, "json", `either inline JSON string or @path/to/file.json with request body`)

	// TODO: array: auto_tag_configs
	// TODO: complex arg: included_schemas
	cmd.Flags().StringVar(&updateCatalogConfigReq.CatalogConfig.Name, "name", updateCatalogConfigReq.CatalogConfig.Name, `Resource name in the format: catalogs/{catalog_name}/config.`)

	cmd.Use = "update-catalog-config NAME UPDATE_MASK"
	cmd.Short = `Update config for a catalog.`
	cmd.Long = `Update config for a catalog.
  
  Update the Data Classification configuration for a catalog. - The config must
  already exist for the catalog. - Updates fields specified in the update_mask.
  Use update_mask field to perform partial updates of the configuration.

  Arguments:
    NAME: Resource name in the format: catalogs/{catalog_name}/config.
    UPDATE_MASK: Field mask specifying which fields to update.`

	cmd.Annotations = make(map[string]string)

	cmd.Args = func(cmd *cobra.Command, args []string) error {
		check := root.ExactArgs(2)
		return check(cmd, args)
	}

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) (err error) {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

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
		updateCatalogConfigReq.Name = args[0]
		if args[1] != "" {
			updateMaskArray := strings.Split(args[1], ",")
			updateCatalogConfigReq.UpdateMask = *fieldmask.New(updateMaskArray)
		}

		response, err := w.DataClassification.UpdateCatalogConfig(ctx, updateCatalogConfigReq)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, response)
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
