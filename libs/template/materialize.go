package template

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/filer"
)

const (
	libraryDirName  = "library"
	templateDirName = "template"
	schemaFileName  = "databricks_template_schema.json"
)

// This function materializes the input templates as a project, using user defined
// configurations.
// Parameters:
//
//	ctx: 			context containing a cmdio object. This is used to prompt the user
//	configFilePath: file path containing user defined config values
//	templateFS: 	root of the template definition
//	outputFiler: 	filer to use for writing the initialized template
func Materialize(ctx context.Context, configFilePath string, templateFS fs.FS, outputFiler filer.Filer) error {
	if _, err := fs.Stat(templateFS, schemaFileName); errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("not a bundle template: expected to find a template schema file at %s", schemaFileName)
	}

	config, err := newConfig(ctx, templateFS, schemaFileName)
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

	helpers := loadHelpers(ctx)
	r, err := newRenderer(ctx, config.values, helpers, templateFS, templateDirName, libraryDirName)
	if err != nil {
		return err
	}

	// Print welcome message
	welcome := config.schema.WelcomeMessage
	if welcome != "" {
		welcome, err = r.executeTemplate(welcome)
		if err != nil {
			return err
		}
		cmdio.LogString(ctx, welcome)
	}

	// Prompt user for any missing config values. Assign default values if
	// terminal is not TTY
	err = config.promptOrAssignDefaultValues(r)
	if err != nil {
		return err
	}
	err = config.validate()
	if err != nil {
		return err
	}

	// Walk and render the template, since input configuration is complete
	err = r.walk()
	if err != nil {
		return err
	}

	err = r.persistToDisk(ctx, outputFiler)
	if err != nil {
		return err
	}

	success := config.schema.SuccessMessage
	if success == "" {
		cmdio.LogString(ctx, "✨ Successfully initialized template")
	} else {
		success, err = r.executeTemplate(success)
		if err != nil {
			return err
		}
		cmdio.LogString(ctx, success)
	}
	return nil
}
