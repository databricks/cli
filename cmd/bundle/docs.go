package bundle

import (
	"encoding/json"

	"github.com/databricks/bricks/bundle/schema"
	"github.com/spf13/cobra"
)

var docsCmd = &cobra.Command{
	Use:   "docs",
	Short: "Generate JSON Schema docs for bundle config",

	RunE: func(cmd *cobra.Command, args []string) error {
		docs, err := schema.InitializeBundleDocs()
		if err != nil {
			return err
		}
		docs.Properties["resources"] = nil
		jsonSchema, err := json.MarshalIndent(docs, "", "  ")
		if err != nil {
			return err
		}
		cmd.OutOrStdout().Write(jsonSchema)
		return nil
	},
}

func init() {
	AddCommand(docsCmd)
}
