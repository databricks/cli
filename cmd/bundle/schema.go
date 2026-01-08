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

Common use cases:
- Configure IDE/editor validation for databricks.yml files
- Set up autocomplete and IntelliSense for bundle configuration`,
		Args: root.NoArgs,
	}

	var refOnly bool
	cmd.Flags().BoolVar(&refOnly, "ref-only", false, "Output the schema without interpolation patterns")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		data := schema.Bytes
		if refOnly {
			data = schema.BytesRefOnly
		}
		_, err := cmd.OutOrStdout().Write(data)
		return err
	}

	return cmd
}
