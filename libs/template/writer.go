package template

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/jsonschema"
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

	// Log telemetry for the template initialization event.
	LogTelemetry(ctx context.Context)
}

type defaultWriter struct {
	name        TemplateName
	configPath  string
	outputFiler filer.Filer

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
		return filer.NewWorkspaceFilesExtensionsClient(cmdctx.WorkspaceClient(ctx), outputDir)
	}

	return filer.NewLocalClient(outputDir)
}

func (tmpl *defaultWriter) Configure(ctx context.Context, configPath, outputDir string) error {
	tmpl.configPath = configPath

	outputFiler, err := constructOutputFiler(ctx, outputDir)
	if err != nil {
		return err
	}

	tmpl.outputFiler = outputFiler
	return nil
}

func (tmpl *defaultWriter) promptForInput(ctx context.Context, reader Reader) error {
	readerFs, err := reader.FS(ctx)
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

func (tmpl *defaultWriter) Materialize(ctx context.Context, reader Reader) error {
	err := tmpl.promptForInput(ctx, reader)
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

func (tmpl *defaultWriter) LogTelemetry(ctx context.Context) {
	telemetry.Log(ctx, protos.DatabricksCliLog{
		BundleInitEvent: &protos.BundleInitEvent{
			BundleUuid:   bundleUuid,
			TemplateName: string(tmpl.name),
		},
	})
}

type writerWithFullTelemetry struct {
	defaultWriter
}

func (tmpl *writerWithFullTelemetry) LogTelemetry(ctx context.Context) {
	var args []protos.BundleInitTemplateEnumArg
	for k, v := range tmpl.config.values {
		s := tmpl.config.schema.Properties[k]

		switch {
		case s.Type == jsonschema.BooleanType:
			args = append(args, protos.BundleInitTemplateEnumArg{
				Key:   k,
				Value: strconv.FormatBool(v.(bool)),
			})

		case len(s.Enum) > 0:
			args = append(args, protos.BundleInitTemplateEnumArg{
				Key:   k,
				Value: v.(string),
			})

		default:
			// Do nothing
			// We only log enum or boolean values

		}
	}

	// Sort the arguments by key for deterministic telemetry logging
	sort.Slice(args, func(i, j int) bool {
		return args[i].Key < args[j].Key
	})

	telemetry.Log(ctx, protos.DatabricksCliLog{
		BundleInitEvent: &protos.BundleInitEvent{
			BundleUuid:       bundleUuid,
			TemplateName:     string(tmpl.name),
			TemplateEnumArgs: args,
		},
	})
}
