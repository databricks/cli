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

func NewGenerateExternalLocationCommand() *cobra.Command {
	var existingExternalLocationName string
	var outputDir string
	var force bool

	cmd := &cobra.Command{
		Use:   "external-location",
		Short: "Generate ucm configuration for an existing Unity Catalog external location",
		Long: `Generate ucm configuration for an existing Unity Catalog external location.

Fetches the external location by name and writes a per-resource YAML
fragment to --output-dir that you can include from your ucm.yml.

Example:
  databricks ucm generate external-location --existing-external-location-name prod_loc --key prod_loc`,
		Args:    root.NoArgs,
		PreRunE: root.MustWorkspaceClient,
	}

	cmd.Flags().StringVar(&existingExternalLocationName, "existing-external-location-name", "", "Name of the existing external location to import.")
	cmd.Flags().StringVar(&outputDir, "output-dir", ".", "Directory to write the generated configuration into.")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing files in --output-dir.")
	_ = cmd.MarkFlagRequired("existing-external-location-name")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		if w == nil {
			return fmt.Errorf("workspace client not configured")
		}

		info, err := w.ExternalLocations.GetByName(ctx, existingExternalLocationName)
		if err != nil {
			return fmt.Errorf("fetch external location %q: %w", existingExternalLocationName, err)
		}

		res := &resources.ExternalLocation{
			CreateExternalLocation: catalog.CreateExternalLocation{
				Name:           info.Name,
				Url:            info.Url,
				CredentialName: info.CredentialName,
				Comment:        info.Comment,
				ReadOnly:       info.ReadOnly,
				Fallback:       info.Fallback,
			},
		}

		key := getKey(cmd, info.Name)
		outPath, err := writeResourceYAML(outputDir, "external_locations", key, res, force)
		if err != nil {
			return err
		}

		cmdio.LogString(ctx, fmt.Sprintf("Wrote external location %q to %s", key, filepath.ToSlash(outPath)))
		return nil
	}

	return cmd
}
