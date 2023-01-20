package bundle

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/databricks/bricks/bundle/config"
	"github.com/databricks/bricks/bundle/schema"
	"github.com/spf13/cobra"
)

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Generates JSON schema for a bundle config",

	RunE: func(cmd *cobra.Command, args []string) error {
		docs, err := schema.GetBundleDocs()
		if err != nil {
			return err
		}
		dummyBundleConfig := config.Root{}
		schema, err := schema.New(reflect.TypeOf(dummyBundleConfig), docs)
		if err != nil {
			return err
		}
		jsonSchema, err := json.MarshalIndent(schema, "", "    ")
		if err != nil {
			return err
		}
		fmt.Println(string(jsonSchema))
		return nil
	},
}

func init() {
	AddCommand(schemaCmd)
}
