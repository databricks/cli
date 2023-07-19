package template

import (
	"context"
	"path/filepath"
)

const schemaFileName = "databricks_template_schema.json"
const templateDirName = "template"
const libraryDirName = "library"

func Materialize(templateRoot, instanceRoot, configPath string) error {
	// read the file containing schema for template input parameters
	schema, err := ReadSchema(filepath.Join(templateRoot, schemaFileName))
	if err != nil {
		return err
	}

	// read user config to initialize the template with
	config, err := schema.ReadConfig(configPath)
	if err != nil {
		return err
	}

	r, err := newRenderer(context.TODO(), config, filepath.Join(templateRoot, libraryDirName), instanceRoot, filepath.Join(templateRoot, templateDirName))
	if err != nil {
		return err
	}

	// materialize the template
	return walk(r, ".")
}
