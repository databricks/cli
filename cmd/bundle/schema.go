package bundle

import (
	"encoding/json"
	"os"
	"reflect"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/schema"
	"github.com/spf13/cobra"
)

func newSchemaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Generate JSON Schema for bundle configuration",
	}

	var openapi string
	var outputFile string
	var onlyDocs bool
	cmd.Flags().StringVar(&openapi, "openapi", "", "path to a databricks openapi spec")
	cmd.Flags().BoolVar(&onlyDocs, "only-docs", false, "only generate descriptions for the schema")
	cmd.Flags().StringVar(&outputFile, "output-file", "", "File path to write the schema to. If not specified, the schema will be written to stdout.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// If no openapi spec is provided, try to use the environment variable.
		// This environment variable is set during CLI code generation.
		if openapi == "" {
			openapi = os.Getenv("DATABRICKS_OPENAPI_SPEC")
		}
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

		// If outputFile is provided, write to that file.
		if outputFile != "" {
			f, err := os.OpenFile(outputFile, os.O_TRUNC|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			defer f.Close()
			cmd.SetOut(f)
		}
		cmd.OutOrStdout().Write(result)
		return nil
	}

	return cmd
}
