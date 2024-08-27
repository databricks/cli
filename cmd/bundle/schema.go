package bundle

import (
	_ "embed"

	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

//go:embed _generated/jsonschema.json
var b []byte

func newSchemaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Generate JSON Schema for bundle configuration",
		Args:  root.NoArgs,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		_, err := cmd.OutOrStdout().Write(b)
		return err
	}

	return cmd
}
