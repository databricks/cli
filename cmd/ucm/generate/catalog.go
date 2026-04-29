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

func NewGenerateCatalogCommand() *cobra.Command {
	var existingCatalogName string
	var outputDir string
	var force bool

	cmd := &cobra.Command{
		Use:   "catalog",
		Short: "Generate ucm configuration for an existing Unity Catalog catalog",
		Long: `Generate ucm configuration for an existing Unity Catalog catalog.

Fetches the catalog by name and writes a per-resource YAML fragment to
--output-dir that you can include from your ucm.yml.

Example:
  databricks ucm generate catalog --existing-catalog-name prod --key prod_catalog`,
		Args:    root.NoArgs,
		PreRunE: root.MustWorkspaceClient,
	}

	cmd.Flags().StringVar(&existingCatalogName, "existing-catalog-name", "", "Name of the existing catalog to import.")
	cmd.Flags().StringVar(&outputDir, "output-dir", ".", "Directory to write the generated configuration into.")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing files in --output-dir.")
	_ = cmd.MarkFlagRequired("existing-catalog-name")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		if w == nil {
			return fmt.Errorf("workspace client not configured")
		}

		info, err := w.Catalogs.GetByName(ctx, existingCatalogName)
		if err != nil {
			return fmt.Errorf("fetch catalog %q: %w", existingCatalogName, err)
		}

		res := &resources.Catalog{
			CreateCatalog: catalog.CreateCatalog{
				Name:        info.Name,
				Comment:     info.Comment,
				StorageRoot: info.StorageRoot,
			},
			Tags: copyMap(info.Properties),
		}

		key := getKey(cmd, info.Name)
		outPath, err := writeResourceYAML(outputDir, "catalogs", key, res, force)
		if err != nil {
			return err
		}

		cmdio.LogString(ctx, fmt.Sprintf("Wrote catalog %q to %s", key, filepath.ToSlash(outPath)))
		return nil
	}

	return cmd
}

// copyMap returns a copy of in, or nil when in is empty. Centralised here so
// the per-kind subcommands don't each duplicate the empty-vs-nil rule.
func copyMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
