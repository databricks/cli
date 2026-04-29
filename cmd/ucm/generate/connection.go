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

func NewGenerateConnectionCommand() *cobra.Command {
	var existingConnectionName string
	var outputDir string
	var force bool

	cmd := &cobra.Command{
		Use:   "connection",
		Short: "Generate ucm configuration for an existing Unity Catalog connection",
		Long: `Generate ucm configuration for an existing Unity Catalog connection.

Fetches the connection by name and writes a per-resource YAML fragment to
--output-dir that you can include from your ucm.yml.

Example:
  databricks ucm generate connection --existing-connection-name prod_conn --key prod_conn`,
		Args:    root.NoArgs,
		PreRunE: root.MustWorkspaceClient,
	}

	cmd.Flags().StringVar(&existingConnectionName, "existing-connection-name", "", "Name of the existing connection to import.")
	cmd.Flags().StringVar(&outputDir, "output-dir", ".", "Directory to write the generated configuration into.")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing files in --output-dir.")
	_ = cmd.MarkFlagRequired("existing-connection-name")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)
		if w == nil {
			return fmt.Errorf("workspace client not configured")
		}

		info, err := w.Connections.GetByName(ctx, existingConnectionName)
		if err != nil {
			return fmt.Errorf("fetch connection %q: %w", existingConnectionName, err)
		}

		res := &resources.Connection{
			CreateConnection: catalog.CreateConnection{
				Name:           info.Name,
				ConnectionType: info.ConnectionType,
				Options:        copyMap(info.Options),
				Comment:        info.Comment,
				Properties:     copyMap(info.Properties),
				ReadOnly:       info.ReadOnly,
			},
		}

		key := getKey(cmd, info.Name)
		outPath, err := writeResourceYAML(outputDir, "connections", key, res, force)
		if err != nil {
			return err
		}

		cmdio.LogString(ctx, fmt.Sprintf("Wrote connection %q to %s", key, filepath.ToSlash(outPath)))
		return nil
	}

	return cmd
}
