package template

import (
	"path/filepath"
)

const ConfigFileName = "config.json"
const schemaFileName = "schema.json"
const templateDirName = "template"

func Materialize(templatePath, instancePath string) error {
	// read the file containing schema for template input parameters
	schema, err := ReadSchema(filepath.Join(templatePath, schemaFileName))
	if err != nil {
		return err
	}

	// read user config to initalize the template with
	config, err := schema.ReadConfig(filepath.Join(instancePath, ConfigFileName))
	if err != nil {
		return err
	}

	// materialize the template
	return WalkFileTree(config, filepath.Join(templatePath, templateDirName), instancePath)
}
