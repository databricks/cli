package template

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/filer"
)

// TODO: Retain coverage for the missing schema test case
// func TestMaterializeForNonTemplateDirectory(t *testing.T) {
// 	tmpDir := t.TempDir()
// 	w, err := databricks.NewWorkspaceClient(&databricks.Config{})
// 	require.NoError(t, err)
// 	ctx := root.SetWorkspaceClient(context.Background(), w)

// 	tmpl := TemplateX{
// 		TemplateOpts: TemplateOpts{
// 			ConfigFilePath: "",
// 			TemplateFS:     os.DirFS(tmpDir),
// 			OutputFiler:    nil,
// 		},
// 	}

// 	// Try to materialize a non-template directory.
// 	err = tmpl.Materialize(ctx)
// 	assert.EqualError(t, err, fmt.Sprintf("not a bundle template: expected to find a template schema file at %s", schemaFileName))
// }


// TODO: Add tests for these writers, mocking the cmdio library
// at the same time.
const (
	libraryDirName  = "library"
	templateDirName = "template"
	schemaFileName  = "databricks_template_schema.json"
)

type Writer interface {
	Initialize(reader Reader, configPath string, outputFiler filer.Filer)
	Materialize(ctx context.Context) error
	LogTelemetry(ctx context.Context) error
}

type defaultWriter struct {
	reader      Reader
	configPath  string
	outputFiler filer.Filer

	// Internal state
	config   *config
	renderer *renderer
}

func (tmpl *defaultWriter) Initialize(reader Reader, configPath string, outputFiler filer.Filer) {
	tmpl.configPath = configPath
	tmpl.outputFiler = outputFiler
}

func (tmpl *defaultWriter) promptForInput(ctx context.Context) error {
	readerFs, err := tmpl.reader.FS(ctx)
	if err != nil {
		return err
	}
	if _, err := fs.Stat(readerFs, schemaFileName); errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("not a bundle template: expected to find a template schema file at %s", schemaFileName)
	}

	tmpl.config, err = newConfig(ctx, readerFs, schemaFileName)
	if err != nil {
		return err
	}

	// Read and assign config values from file
	if tmpl.configPath != "" {
		err = tmpl.config.assignValuesFromFile(tmpl.configPath)
		if err != nil {
			return err
		}
	}

	helpers := loadHelpers(ctx)
	tmpl.renderer, err = newRenderer(ctx, tmpl.config.values, helpers, readerFs, templateDirName, libraryDirName)
	if err != nil {
		return err
	}

	// Print welcome message
	welcome := tmpl.config.schema.WelcomeMessage
	if welcome != "" {
		welcome, err = tmpl.renderer.executeTemplate(welcome)
		if err != nil {
			return err
		}
		cmdio.LogString(ctx, welcome)
	}

	// Prompt user for any missing config values. Assign default values if
	// terminal is not TTY
	err = tmpl.config.promptOrAssignDefaultValues(tmpl.renderer)
	if err != nil {
		return err
	}
	return tmpl.config.validate()
}

func (tmpl *defaultWriter) printSuccessMessage(ctx context.Context) error {
	success := tmpl.config.schema.SuccessMessage
	if success == "" {
		cmdio.LogString(ctx, "✨ Successfully initialized template")
		return nil
	}

	success, err := tmpl.renderer.executeTemplate(success)
	if err != nil {
		return err
	}
	cmdio.LogString(ctx, success)
	return nil
}

func (tmpl *defaultWriter) Materialize(ctx context.Context) error {
	err := tmpl.promptForInput(ctx)
	if err != nil {
		return err
	}

	// Walk the template file tree and compute in-memory representations of the
	// output files.
	err = tmpl.renderer.walk()
	if err != nil {
		return err
	}

	// Flush the output files to disk.
	err = tmpl.renderer.persistToDisk(ctx, tmpl.outputFiler)
	if err != nil {
		return err
	}

	return tmpl.printSuccessMessage(ctx)
}

func (tmpl *defaultWriter) LogTelemetry(ctx context.Context) error {
	// no-op
	return nil
}

type writerWithTelemetry struct {
	defaultWriter
}

func (tmpl *writerWithTelemetry) LogTelemetry(ctx context.Context) error {
	// Log telemetry. TODO.
	return nil
}

func NewWriterWithTelemetry(reader Reader, configPath string, outputFiler filer.Filer) Writer {
	return &writerWithTelemetry{
		defaultWriter: defaultWriter{
			reader:      reader,
			configPath:  configPath,
			outputFiler: outputFiler,
		},
	}
}
