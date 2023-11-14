package template

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/databricks/cli/libs/cmdio"
)

const libraryDirName = "library"
const templateDirName = "template"
const schemaFileName = "databricks_template_schema.json"

//go:embed all:templates
var builtinTemplates embed.FS

// This function materializes the input templates as a project, using user defined
// configurations.
// Parameters:
//
//	ctx: 			context containing a cmdio object. This is used to prompt the user
//	configFilePath: file path containing user defined config values
//	templateRoot: 	root of the template definition
//	outputDir: 	root of directory where to initialize the template
func Materialize(ctx context.Context, configFilePath, templateRoot, outputDir string) error {
	// Use a temporary directory in case any builtin templates like default-python are used
	tempDir, err := os.MkdirTemp("", "templates")
	defer os.RemoveAll(tempDir)
	if err != nil {
		return err
	}
	templateRoot, err = prepareBuiltinTemplates(templateRoot, tempDir)
	if err != nil {
		return err
	}

	templatePath := filepath.Join(templateRoot, templateDirName)
	libraryPath := filepath.Join(templateRoot, libraryDirName)
	schemaPath := filepath.Join(templateRoot, schemaFileName)
	helpers := loadHelpers(ctx)

	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		return fmt.Errorf("expected to find a template schema file at %s. Valid bundle templates are expected to contain a schema file", schemaPath)
	}

	config, err := newConfig(ctx, schemaPath)
	if err != nil {
		return err
	}

	// Print welcome message
	welcome := config.schema.WelcomeMessage
	if welcome != "" {
		cmdio.LogString(ctx, welcome)
	}

	// Read and assign config values from file
	if configFilePath != "" {
		err = config.assignValuesFromFile(configFilePath)
		if err != nil {
			return err
		}
	}

	r, err := newRenderer(ctx, config.values, helpers, templatePath, libraryPath, outputDir)
	if err != nil {
		return err
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

	err = r.persistToDisk()
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

// If the given templateRoot matches
func prepareBuiltinTemplates(templateRoot string, tempDir string) (string, error) {
	_, err := fs.Stat(builtinTemplates, path.Join("templates", templateRoot))
	if err != nil {
		// The given path doesn't appear to be using out built-in templates
		return templateRoot, nil
	}

	// We have a built-in template with the same name as templateRoot!
	// Now we need to make a fully copy of the builtin templates to a real file system
	// since template.Parse() doesn't support embed.FS.
	err = fs.WalkDir(builtinTemplates, "templates", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		targetPath := filepath.Join(tempDir, path)
		if entry.IsDir() {
			return os.Mkdir(targetPath, 0755)
		} else {
			content, err := fs.ReadFile(builtinTemplates, path)
			if err != nil {
				return err
			}
			return os.WriteFile(targetPath, content, 0644)
		}
	})

	if err != nil {
		return "", err
	}

	return filepath.Join(tempDir, "templates", templateRoot), nil
}
