package template

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/events"
)

const (
	libraryDirName  = "library"
	templateDirName = "template"
	schemaFileName  = "databricks_template_schema.json"
)

type TemplateOpts struct {
	// file path containing user defined config values
	ConfigFilePath string
	// root of the template definition
	TemplateFS fs.FS
	// filer to use for writing the initialized template
	OutputFiler filer.Filer
	// If true, we'll include the enum template args in the telemetry payload.
	IsDatabricksOwned bool
	// Name of the template. For non-Databricks owned templates, this is set to
	// "custom".
	Name string
}

type Template struct {
	TemplateOpts

	// internal object used to prompt user for config values and store them.
	config *config

	// internal object user to render the template.
	renderer *renderer
}

// This function resolves input to use to materialize the template in two steps.
//  1. First, this function loads any user specified input configuration if the user
//     has provided a config file path.
//  2. For any values that are required by the template but not provided in the config
//     file, this function prompts the user for them.
func (t *Template) resolveTemplateInput(ctx context.Context) error {
	if _, err := fs.Stat(t.TemplateFS, schemaFileName); errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("not a bundle template: expected to find a template schema file at %s", schemaFileName)
	}

	var err error
	t.config, err = newConfig(ctx, t.TemplateFS, schemaFileName)
	if err != nil {
		return err
	}

	// Read and assign config values from file
	if t.ConfigFilePath != "" {
		err = t.config.assignValuesFromFile(t.ConfigFilePath)
		if err != nil {
			return err
		}
	}

	helpers := loadHelpers(ctx)
	t.renderer, err = newRenderer(ctx, t.config.values, helpers, t.TemplateFS, templateDirName, libraryDirName)
	if err != nil {
		return err
	}

	// Print welcome message
	welcome := t.config.schema.WelcomeMessage
	if welcome != "" {
		welcome, err = t.renderer.executeTemplate(welcome)
		if err != nil {
			return err
		}
		cmdio.LogString(ctx, welcome)
	}

	// Prompt user for any missing config values. Assign default values if
	// terminal is not TTY
	err = t.config.promptOrAssignDefaultValues(t.renderer)
	if err != nil {
		return err
	}
	return t.config.validate()
}

func (t *Template) printSuccessMessage(ctx context.Context) error {
	success := t.config.schema.SuccessMessage
	if success == "" {
		cmdio.LogString(ctx, "✨ Successfully initialized template")
		return nil
	}

	success, err := t.renderer.executeTemplate(success)
	if err != nil {
		return err
	}
	cmdio.LogString(ctx, success)
	return nil
}

func (t *Template) logTelemetry(ctx context.Context) error {
	// Only log telemetry input for Databricks owned templates. This is to prevent
	// accidentally collecting PII from custom user templates.
	templateEnumArgs := map[string]string{}
	if t.IsDatabricksOwned {
		templateEnumArgs = t.config.enumValues()
	} else {
		t.Name = "custom"
	}

	event := telemetry.DatabricksCliLog{
		BundleInitEvent: &events.BundleInitEvent{
			Uuid:             bundleUuid,
			TemplateName:     t.Name,
			TemplateEnumArgs: templateEnumArgs,
		},
	}

	return telemetry.Log(ctx, telemetry.FrontendLogEntry{
		DatabricksCliLog: event,
	})
}

// This function materializes the input templates as a project, using user defined
// configurations.
func (t *Template) Materialize(ctx context.Context) error {
	err := t.resolveTemplateInput(ctx)
	if err != nil {
		return err
	}

	// Walk the template file tree and compute in-memory representations of the
	// output files.
	err = t.renderer.walk()
	if err != nil {
		return err
	}

	// Flush the output files to disk.
	err = t.renderer.persistToDisk(ctx, t.OutputFiler)
	if err != nil {
		return err
	}

	err = t.printSuccessMessage(ctx)
	if err != nil {
		return err
	}

	return t.logTelemetry(ctx)
}
