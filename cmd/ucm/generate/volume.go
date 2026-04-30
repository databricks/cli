package generate

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/ucm/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

// NewGenerateVolumeCommand returns the `ucm generate volume` cobra subcommand.
func NewGenerateVolumeCommand() *cobra.Command {
	var existingVolumeName string
	var outputDir string
	var force bool

	cmd := &cobra.Command{
		Use:   "volume",
		Short: "Generate ucm configuration for an existing Unity Catalog volume",
		Long: `Generate ucm configuration for an existing Unity Catalog volume.

Fetches the volume by its full name (catalog.schema.volume) and writes a
per-resource YAML fragment to --output-dir that you can include from your
ucm.yml.

Note: Managed volumes have their server-assigned storage_location dropped
from the output; UC re-derives it on deploy.

Example:
  databricks ucm generate volume --existing-volume-name prod.raw.landing --key landing_volume`,
		Args:    root.NoArgs,
		PreRunE: root.MustWorkspaceClient,
	}

	cmd.Flags().StringVar(&existingVolumeName, "existing-volume-name", "", "Full name (catalog.schema.volume) of the existing volume to import.")
	cmd.Flags().StringVar(&outputDir, "output-dir", ".", "Directory to write the generated configuration into.")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing files in --output-dir.")
	_ = cmd.MarkFlagRequired("existing-volume-name")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		if w == nil {
			return fmt.Errorf("workspace client not configured")
		}

		info, err := w.Volumes.ReadByName(ctx, existingVolumeName)
		if err != nil {
			return fmt.Errorf("fetch volume %q: %w", existingVolumeName, err)
		}

		// Managed volumes echo back a server-assigned storage_location; the
		// ucm Volume model (matching the SDK CreateVolumeRequestContent
		// validation) refuses to set it on MANAGED, so drop it here so the
		// per-kind direct-SDK scan emits a deploy-clean shape.
		storage := info.StorageLocation
		if strings.EqualFold(string(info.VolumeType), "MANAGED") {
			storage = ""
		}

		res := &resources.Volume{
			CreateVolumeRequestContent: catalog.CreateVolumeRequestContent{
				Name:            info.Name,
				CatalogName:     info.CatalogName,
				SchemaName:      info.SchemaName,
				VolumeType:      info.VolumeType,
				StorageLocation: storage,
				Comment:         info.Comment,
			},
		}

		key := getKey(cmd, existingVolumeName)
		outPath, err := writeResourceYAML(outputDir, "volumes", key, res, force)
		if err != nil {
			return err
		}

		cmdio.LogString(ctx, fmt.Sprintf("Wrote volume %q to %s", key, filepath.ToSlash(outPath)))
		return nil
	}

	return cmd
}
