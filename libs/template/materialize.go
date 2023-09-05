package template

import (
	"context"
	"path/filepath"
)

const libraryDirName = "library"
const templateDirName = "template"
const schemaFileName = "databricks_template_schema.json"

// This function materializes the input templates as a project, using user defined
// configurations.
// Parameters:
//
//	ctx: 			context containing a cmdio object. This is used to prompt the user
//	configFilePath: file path containing user defined config values
//	templateRoot: 	root of the template definition
//	projectDir: 	root of directory where to initialize the project
func Materialize(ctx context.Context, configFilePath, templateRoot, projectDir string) error {
	templatePath := filepath.Join(templateRoot, templateDirName)
	libraryPath := filepath.Join(templateRoot, libraryDirName)
	schemaPath := filepath.Join(templateRoot, schemaFileName)

	config, err := newConfig(ctx, schemaPath)
	if err != nil {
		return err
	}

	// Read and assign config values from file
	if configFilePath != "" {
		err = config.assignValuesFromFile(configFilePath)
		if err != nil {
			return err
		}
	}

	// Prompt user for any missing config values. Assign default values if
	// terminal is not TTY
	err = config.promptOrAssignDefaultValues()
	if err != nil {
		return err
	}

	err = config.validate()
	if err != nil {
		return err
	}

	// Walk and render the template, since input configuration is complete
	r, err := newRenderer(ctx, config.values, templatePath, libraryPath, projectDir)
	if err != nil {
		return err
	}
	err = r.walk()
	if err != nil {
		return err
	}
	return r.persistToDisk()
}
