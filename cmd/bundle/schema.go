package bundle

import (
	"github.com/databricks/cli/bundle/schema"
	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

func newSchemaCommand(hidden bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "schema",
		Short:  "Generate JSON Schema for bundle configuration",
		Args:   root.NoArgs,
		Hidden: hidden,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		_, err := cmd.OutOrStdout().Write(schema.Bytes)
		return err
	}

	return cmd
}
