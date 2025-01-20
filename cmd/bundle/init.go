package bundle

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/template"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/spf13/cobra"
)

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
		var templatePathOrUrl string
		if len(args) > 0 {
			templatePathOrUrl = args[0]
		}
		r := template.Resolver{
			TemplatePathOrUrl: templatePathOrUrl,
			ConfigFile:        configFile,
			OutputDir:         outputDir,
			TemplateDir:       templateDir,
			Tag:               tag,
			Branch:            branch,
		}

		ctx := cmd.Context()
		tmpl, err := r.Resolve(ctx)
		if errors.Is(err, template.ErrCustomSelected) {
			cmdio.LogString(ctx, "Please specify a path or Git repository to use a custom template.")
			cmdio.LogString(ctx, "See https://docs.databricks.com/en/dev-tools/bundles/templates.html to learn more about custom templates.")
			return nil
		}
		if err != nil {
			return err
		}
		defer tmpl.Reader.Cleanup(ctx)

		err = tmpl.Writer.Materialize(ctx, tmpl.Reader)
		if err != nil {
			return err
		}
		defer tmpl.Reader.Cleanup(ctx)

		tmpl.Writer.LogTelemetry(ctx)
		return nil
	}
	return cmd
}
