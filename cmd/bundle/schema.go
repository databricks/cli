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
		docs, err := schema.BundleDocs(openapi)
		if err != nil {
			return err
		}
		schema, err := schema.New(reflect.TypeOf(config.Root{}), docs)
		if err != nil {
			return err
		}
		result, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			return err
		}
		if onlyDocs {
			result, err = json.MarshalIndent(docs, "", "  ")
			if err != nil {
				return err
			}
		}
		cmd.OutOrStdout().Write(result)
		return nil
	},
}

var openapi string
var onlyDocs bool

func init() {
	AddCommand(schemaCmd)
	schemaCmd.Flags().StringVar(&openapi, "openapi", "", "path to a databricks openapi spec")
	schemaCmd.Flags().BoolVar(&onlyDocs, "only-docs", false, "only generate descriptions for the schema")
}
