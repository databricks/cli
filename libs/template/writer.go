package template

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
)

const (
	libraryDirName  = "library"
	templateDirName = "template"
	schemaFileName  = "databricks_template_schema.json"
)

type Writer interface {
	// Configure the writer with:
	// 1. The path to the config file (if any) that contains input values for the
	// template.
	// 2. The output directory where the template will be materialized.
	Configure(ctx context.Context, configPath, outputDir string) error

	// Materialize the template to the local file system.
	Materialize(ctx context.Context, r Reader) error

	// Log telemetry associated with the initialization of this template.
	LogTelemetry(ctx context.Context)
}

type defaultWriter struct {
	configPath   string
	outputFiler  filer.Filer
	templateName TemplateName

	// Internal state
	config   *config
	renderer *renderer
}

func constructOutputFiler(ctx context.Context, outputDir string) (filer.Filer, error) {
	outputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return nil, err
	}

	// If the CLI is running on DBR and we're writing to the workspace file system,
	// use the extension-aware workspace filesystem filer to instantiate the template.
	//
	// It is not possible to write notebooks through the workspace filesystem's FUSE mount.
	// Therefore this is the only way we can initialize templates that contain notebooks
	// when running the CLI on DBR and initializing a template to the workspace.
	//
	if strings.HasPrefix(outputDir, "/Workspace/") && dbr.RunsOnRuntime(ctx) {
		return filer.NewWorkspaceFilesExtensionsClient(root.WorkspaceClient(ctx), outputDir)
	}

	return filer.NewLocalClient(outputDir)
}

func (writer *defaultWriter) Configure(ctx context.Context, configPath, outputDir string) error {
	writer.configPath = configPath

	outputFiler, err := constructOutputFiler(ctx, outputDir)
	if err != nil {
		return err
	}

	writer.outputFiler = outputFiler
	return nil
}

func (writer *defaultWriter) promptForInput(ctx context.Context, reader Reader) error {
	readerFs, err := reader.FS(ctx)
	if err != nil {
		return err
	}
	if _, err := fs.Stat(readerFs, schemaFileName); errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("not a bundle template: expected to find a template schema file at %s", schemaFileName)
	}

	writer.config, err = newConfig(ctx, readerFs, schemaFileName)
	if err != nil {
		return err
	}

	// Read and assign config values from file
	if writer.configPath != "" {
		err = writer.config.assignValuesFromFile(writer.configPath)
		if err != nil {
			return err
		}
	}

	helpers := loadHelpers(ctx)
	writer.renderer, err = newRenderer(ctx, writer.config.values, helpers, readerFs, templateDirName, libraryDirName)
	if err != nil {
		return err
	}

	// Print welcome message
	welcome := writer.config.schema.WelcomeMessage
	if welcome != "" {
		welcome, err = writer.renderer.executeTemplate(welcome)
		if err != nil {
			return err
		}
		cmdio.LogString(ctx, welcome)
	}

	// Prompt user for any missing config values. Assign default values if
	// terminal is not TTY
	err = writer.config.promptOrAssignDefaultValues(writer.renderer)
	if err != nil {
		return err
	}
	return writer.config.validate()
}

func (writer *defaultWriter) printSuccessMessage(ctx context.Context) error {
	success := writer.config.schema.SuccessMessage
	if success == "" {
		cmdio.LogString(ctx, "✨ Successfully initialized template")
		return nil
	}

	success, err := writer.renderer.executeTemplate(success)
	if err != nil {
		return err
	}
	cmdio.LogString(ctx, success)
	return nil
}

func (writer *defaultWriter) Materialize(ctx context.Context, reader Reader) error {
	err := writer.promptForInput(ctx, reader)
	if err != nil {
		return err
	}

	// Walk the template file tree and compute in-memory representations of the
	// output files.
	err = writer.renderer.walk()
	if err != nil {
		return err
	}

	// Flush the output files to disk.
	err = writer.renderer.persistToDisk(ctx, writer.outputFiler)
	if err != nil {
		return err
	}

	return writer.printSuccessMessage(ctx)
}

func (writer *defaultWriter) LogTelemetry(ctx context.Context) {
	// We don't log template enum args in the default template writer.
	// This is because this writer can be used for customer templates, and we
	// don't want to log PII.
	event := telemetry.DatabricksCliLog{
		BundleInitEvent: &protos.BundleInitEvent{
			Uuid:         bundleUuid,
			TemplateName: string(writer.templateName),
		},
	}

	telemetry.Log(ctx, event)
}

type writerWithFullTelemetry struct {
	defaultWriter
}

func (writer *writerWithFullTelemetry) LogTelemetry(ctx context.Context) {
	templateEnumArgs := []protos.BundleInitTemplateEnumArg{}
	for k, v := range writer.config.enumValues() {
		templateEnumArgs = append(templateEnumArgs, protos.BundleInitTemplateEnumArg{
			Key:   k,
			Value: v,
		})
	}

	event := telemetry.DatabricksCliLog{
		BundleInitEvent: &protos.BundleInitEvent{
			Uuid:             bundleUuid,
			TemplateName:     string(writer.templateName),
			TemplateEnumArgs: templateEnumArgs,
		},
	}

	telemetry.Log(ctx, event)
}
