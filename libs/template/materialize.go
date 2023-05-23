package template

import (
	"path/filepath"
)

const ConfigFileName = "config.json"
const schemaFileName = "schema.json"
const templateDirName = "template"

func Materialize(templateRoot, instanceRoot, configPath string) error {
	// read the file containing schema for template input parameters
	schema, err := ReadSchema(filepath.Join(templateRoot, schemaFileName))
	if err != nil {
		return err
	}

	// read user config to initalize the template with
	config, err := schema.ReadConfig(configPath)
	if err != nil {
		return err
	}

	// materialize the template
	return walkFileTree(config, filepath.Join(templateRoot, templateDirName), instanceRoot)
}
