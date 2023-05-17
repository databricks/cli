package init

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/databricks/bricks/cmd/root"
	"github.com/spf13/cobra"
)

const ConfigFileName = "config.json"
const SchemaFileName = "schema.json"
const TemplateDirname = "template"

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Template",
	Long:  `Initialize template`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		templateLocation := args[0]

		// read the file containing schema for template input parameters
		schemaBytes, err := os.ReadFile(filepath.Join(templateLocation, SchemaFileName))
		if err != nil {
			return err
		}
		schema := Schema{}
		err = json.Unmarshal(schemaBytes, &schema)
		if err != nil {
			return err
		}

		// read user config to initalize the template with
		var config map[string]interface{}
		b, err := os.ReadFile(ConfigFileName)
		if err != nil {
			return err
		}
		err = json.Unmarshal(b, &config)
		if err != nil {
			return err
		}

		// cast any fields that are supported to be integers. The json unmarshalling
		// for a generic map converts all numbers to floating point
		err = schema.CastFloatToInt(config)
		if err != nil {
			return err
		}

		// validate config according to schema
		err = schema.ValidateConfig(config)
		if err != nil {
			return err
		}

		// materialize the template
		return walkFileTree(config, filepath.Join(args[0], TemplateDirname), ".")
	},
}

func init() {
	root.RootCmd.AddCommand(initCmd)
}
