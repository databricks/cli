package init

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/template"
	"github.com/spf13/cobra"
)

const ConfigFileName = "config.json"
const SchemaFileName = "schema.json"
const TemplateDirName = "template"

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
		schema := template.Schema{}
		err = json.Unmarshal(schemaBytes, &schema)
		if err != nil {
			return err
		}

		// read user config to initalize the template with
		var config map[string]interface{}
		b, err := os.ReadFile(filepath.Join(targetDir, ConfigFileName))
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
		return template.WalkFileTree(config, filepath.Join(args[0], TemplateDirName), targetDir)
	},
}

var targetDir string

func init() {
	initCmd.Flags().StringVar(&targetDir, "target-dir", ".", "path to directory template will be initialized in")
	root.RootCmd.AddCommand(initCmd)
}
