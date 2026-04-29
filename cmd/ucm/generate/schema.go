package generate

import (
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/ucm/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

// NewGenerateSchemaCommand returns the `ucm generate schema` cobra subcommand.
func NewGenerateSchemaCommand() *cobra.Command {
	var existingSchemaName string
	var outputDir string
	var force bool

	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Generate ucm configuration for an existing Unity Catalog schema",
		Long: `Generate ucm configuration for an existing Unity Catalog schema.

Fetches the schema by its full name (catalog.schema) and writes a
per-resource YAML fragment to --output-dir that you can include from your
ucm.yml.

Example:
  databricks ucm generate schema --existing-schema-name prod.events --key events_schema`,
		Args:    root.NoArgs,
		PreRunE: root.MustWorkspaceClient,
	}

	cmd.Flags().StringVar(&existingSchemaName, "existing-schema-name", "", "Full name (catalog.schema) of the existing schema to import.")
	cmd.Flags().StringVar(&outputDir, "output-dir", ".", "Directory to write the generated configuration into.")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing files in --output-dir.")
	_ = cmd.MarkFlagRequired("existing-schema-name")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		if w == nil {
			return fmt.Errorf("workspace client not configured")
		}

		info, err := w.Schemas.GetByFullName(ctx, existingSchemaName)
		if err != nil {
			return fmt.Errorf("fetch schema %q: %w", existingSchemaName, err)
		}

		res := &resources.Schema{
			CreateSchema: catalog.CreateSchema{
				Name:        info.Name,
				CatalogName: info.CatalogName,
				Comment:     info.Comment,
			},
			Tags: copyMap(info.Properties),
		}

		key := getKey(cmd, existingSchemaName)
		outPath, err := writeResourceYAML(outputDir, "schemas", key, res, force)
		if err != nil {
			return err
		}

		cmdio.LogString(ctx, fmt.Sprintf("Wrote schema %q to %s", key, filepath.ToSlash(outPath)))
		return nil
	}

	return cmd
}
