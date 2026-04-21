package ucm

import (
	"github.com/databricks/cli/cmd/root"
	ucmschema "github.com/databricks/cli/ucm/schema"
	"github.com/spf13/cobra"
)

func newSchemaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Print the JSON schema for ucm.yml.",
		Long: `Print the JSON schema for ucm.yml.

Pipe into a file and point your editor at it for autocomplete and validation:

  databricks ucm schema > ucm.schema.json
`,
		Args: root.NoArgs,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		out, err := ucmschema.Generate()
		if err != nil {
			return err
		}
		if _, err := cmd.OutOrStdout().Write(out); err != nil {
			return err
		}
		_, err = cmd.OutOrStdout().Write([]byte{'\n'})
		return err
	}

	return cmd
}
