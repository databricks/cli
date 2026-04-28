package ucm

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/template"
	ucmtemplates "github.com/databricks/cli/ucm/templates"
	"github.com/spf13/cobra"
)

func newInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [TEMPLATE_PATH]",
		Short: "Scaffold a new ucm.yml project from a starter template.",
		Long: fmt.Sprintf(`Scaffold a new ucm.yml project from a starter template.

TEMPLATE_PATH is optional. It can be one of:
%s
- a local filesystem path to a ucm template directory
- a Git repository URL (https:// or git@)

Examples:
  databricks ucm init                   # Prompt for a built-in template
  databricks ucm init default           # Minimal catalog + schema + grant
  databricks ucm init brownfield        # Stub for 'ucm generate' follow-up
  databricks ucm init multienv          # dev/staging/prod targets
  databricks ucm init ./my-template     # Initialize from a local directory
  databricks ucm init --output-dir ./my-project default

See https://docs.databricks.com/en/dev-tools/ucm/index.html for more
information on ucm templates.`, ucmtemplates.HelpDescriptions()),
		Args: root.MaximumNArgs(1),
	}

	var configFile string
	var outputDir string
	var templateDir string
	var tag string
	var branch string
	cmd.Flags().StringVar(&configFile, "config-file", "", "JSON file containing key value pairs of input parameters required for template initialization.")
	cmd.Flags().StringVar(&templateDir, "template-dir", "", "Directory path within a Git repository containing the template.")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Directory to write the initialized template to.")
	cmd.Flags().StringVar(&branch, "branch", "", "Git branch to use for template initialization.")
	cmd.Flags().StringVar(&tag, "tag", "", "Git tag to use for template initialization.")

	// The template renderer exposes helpers (workspace_host, short_name, etc.)
	// that need a live workspace client even when a given template doesn't call
	// them. Mirror DAB's init and eagerly acquire one so any template — custom
	// or built-in — can reference the full helper surface.
	cmd.PreRunE = root.MustWorkspaceClient

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if tag != "" && branch != "" {
			return errors.New("only one of --tag or --branch can be specified")
		}

		var templatePathOrUrl string
		if len(args) > 0 {
			templatePathOrUrl = args[0]
		}

		ctx := cmd.Context()

		// If the user selected a built-in ucm template, stage it to a temp dir
		// and hand the path to the shared resolver so all custom-template
		// plumbing (local filer, schema prompt, rendering) is reused as-is.
		pathOrUrl, cleanup, err := ucmtemplates.ResolveBuiltinOrPassthrough(ctx, templatePathOrUrl)
		if err != nil {
			return err
		}
		defer cleanup()

		r := template.Resolver{
			TemplatePathOrUrl: pathOrUrl,
			ConfigFile:        configFile,
			OutputDir:         outputDir,
			TemplateDir:       templateDir,
			Tag:               tag,
			Branch:            branch,
		}

		tmpl, err := r.Resolve(ctx)
		if errors.Is(err, template.ErrCustomSelected) {
			cmdio.LogString(ctx, "Please specify a path or Git repository to use a custom template.")
			cmdio.LogString(ctx, "See https://docs.databricks.com/en/dev-tools/ucm/index.html to learn more about ucm templates.")
			return nil
		}
		if err != nil {
			return err
		}
		defer tmpl.Reader.Cleanup(ctx)

		if err := tmpl.Writer.Materialize(ctx, tmpl.Reader); err != nil {
			return err
		}
		tmpl.Writer.LogTelemetry(ctx)
		return nil
	}

	return cmd
}
