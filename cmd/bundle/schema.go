package bundle

import (
	"encoding/json"
	"reflect"

	"github.com/databricks/bricks/bundle/config"
	"github.com/databricks/bricks/bundle/schema"
	"github.com/spf13/cobra"
)

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Generate JSON Schema for bundle configuration",

	RunE: func(cmd *cobra.Command, args []string) error {
		docs, err := schema.GetBundleDocs()
		if err != nil {
			return err
		}
		schema, err := schema.New(reflect.TypeOf(config.Root{}), docs)
		if err != nil {
			return err
		}
		jsonSchema, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(jsonSchema)
		return nil
	},
}

func init() {
	AddCommand(schemaCmd)
}
