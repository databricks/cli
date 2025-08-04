package bundle

import (
	"github.com/databricks/cli/bundle/schema"
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func newSchemaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Generate JSON Schema for bundle configuration",
		Long: `Generate JSON Schema for bundle configuration to enable validation and autocomplete.

This command outputs the JSON Schema that describes the structure and validation
rules for Databricks Asset Bundle configuration files.

COMMON USE CASES:
- Configure IDE/editor validation for databricks.yml files
- Set up autocomplete and IntelliSense for bundle configuration`,
		Args: root.NoArgs,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		_, err := cmd.OutOrStdout().Write(schema.Bytes)
		return err
	}

	return cmd
}
