package bundle

import (
	"encoding/json"
	"reflect"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/schema"
	"github.com/spf13/cobra"
)

func newSchemaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Generate JSON Schema for bundle configuration",
		Args:  cobra.NoArgs,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Load embedded schema descriptions.
		docs, err := schema.LoadBundleDescriptions()
		if err != nil {
			return err
		}

		// Generate the JSON schema from the bundle configuration struct in Go.
		schema, err := schema.New(reflect.TypeOf(config.Root{}), docs)
		if err != nil {
			return err
		}

		// Print the JSON schema to stdout.
		result, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(result)
		return nil
	}

	return cmd
}
