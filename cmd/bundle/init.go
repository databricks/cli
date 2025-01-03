package bundle

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/template"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/spf13/cobra"
)

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

func newInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [TEMPLATE_PATH]",
		Short: "Initialize using a bundle template",
		Args:  root.MaximumNArgs(1),
		Long: fmt.Sprintf(`Initialize using a bundle template.

TEMPLATE_PATH optionally specifies which template to use. It can be one of the following:
%s
- a local file system path with a template directory
- a Git repository URL, e.g. https://github.com/my/repository

See https://docs.databricks.com/en/dev-tools/bundles/templates.html for more information on templates.`, template.HelpDescriptions()),
	}

	var configFile string
	var outputDir string
	var templateDir string
	var tag string
	var branch string
	cmd.Flags().StringVar(&configFile, "config-file", "", "JSON file containing key value pairs of input parameters required for template initialization.")
	cmd.Flags().StringVar(&templateDir, "template-dir", "", "Directory path within a Git repository containing the template.")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Directory to write the initialized template to.")
	cmd.Flags().StringVar(&branch, "tag", "", "Git tag to use for template initialization")
	cmd.Flags().StringVar(&tag, "branch", "", "Git branch to use for template initialization")

	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		// Configure the logger to send telemetry to Databricks.
		ctx := telemetry.WithDefaultLogger(cmd.Context())
		cmd.SetContext(ctx)

		return root.MustWorkspaceClient(cmd, args)
	}

	cmd.PostRun = func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		w := root.WorkspaceClient(ctx)
		apiClient, err := client.New(w.Config)
		if err != nil {
			// Uploading telemetry is best effort. Do not error.
			log.Debugf(ctx, "Could not create API client to send telemetry using: %v", err)
			return
		}

		telemetry.Flush(cmd.Context(), apiClient)
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if tag != "" && branch != "" {
			return errors.New("only one of --tag or --branch can be specified")
		}

		// Git ref to use for template initialization
		ref := branch
		if tag != "" {
			ref = tag
		}

		var tmpl *template.Template
		var err error
		ctx := cmd.Context()

		if len(args) > 0 {
			// User already specified a template local path or a Git URL. Use that
			// information to configure a reader for the template
			tmpl = template.Get(template.Custom)
			// TODO: Get rid of the name arg.
			if template.IsGitRepoUrl(args[0]) {
				tmpl.SetReader(template.NewGitReader("", args[0], ref, templateDir))
			} else {
				tmpl.SetReader(template.NewLocalReader("", args[0]))
			}
		} else {
			tmplId, err := template.PromptForTemplateId(cmd.Context(), ref, templateDir)
			if tmplId == template.Custom {
				// If a user selects custom during the prompt, ask them to provide a path or Git URL
				// as a positional argument.
				cmdio.LogString(ctx, "Please specify a path or Git repository to use a custom template.")
				cmdio.LogString(ctx, "See https://docs.databricks.com/en/dev-tools/bundles/templates.html to learn more about custom templates.")
				return nil
			}
			if err != nil {
				return err
			}

			tmpl = template.Get(tmplId)
		}

		defer tmpl.Reader.Close()

		outputFiler, err := constructOutputFiler(ctx, outputDir)
		if err != nil {
			return err
		}

		tmpl.Writer.Initialize(tmpl.Reader, configFile, outputFiler)

		err = tmpl.Writer.Materialize(ctx)
		if err != nil {
			return err
		}

		return tmpl.Writer.LogTelemetry(ctx)
	}
	return cmd
}
