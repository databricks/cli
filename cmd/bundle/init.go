package bundle

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/git"
	"github.com/databricks/cli/libs/template"
	"github.com/spf13/cobra"
)

var gitUrlPrefixes = []string{
	"https://",
	"git@",
}

type nativeTemplate struct {
	name        string
	gitUrl      string
	description string
	aliases     []string
	hidden      bool
}

const customTemplate = "custom..."

var nativeTemplates = []nativeTemplate{
	{
		name:        "default-python",
		description: "The default Python template for Notebooks / Delta Live Tables / Workflows",
	},
	{
		name:        "default-sql",
		description: "The default SQL template for .sql files that run with Databricks SQL",
	},
	{
		name:        "dbt-sql",
		description: "The dbt SQL template (databricks.com/blog/delivering-cost-effective-data-real-time-dbt-and-databricks)",
	},
	{
		name:        "mlops-stacks",
		gitUrl:      "https://github.com/databricks/mlops-stacks",
		description: "The Databricks MLOps Stacks template (github.com/databricks/mlops-stacks)",
		aliases:     []string{"mlops-stack"},
	},
	{
		name:        "default-pydabs",
		gitUrl:      "https://databricks.github.io/workflows-authoring-toolkit/pydabs-template.git",
		hidden:      true,
		description: "The default PyDABs template",
	},
	{
		name:        "experimental-jobs-as-code",
		hidden:      true,
		description: "Jobs as code template (experimental)",
	},
	{
		name:        customTemplate,
		description: "Bring your own template",
	},
}

// Return template descriptions for command-line help
func nativeTemplateHelpDescriptions() string {
	var lines []string
	for _, template := range nativeTemplates {
		if template.name != customTemplate && !template.hidden {
			lines = append(lines, fmt.Sprintf("- %s: %s", template.name, template.description))
		}
	}
	return strings.Join(lines, "\n")
}

// Return template options for an interactive prompt
func nativeTemplateOptions() []cmdio.Tuple {
	names := make([]cmdio.Tuple, 0, len(nativeTemplates))
	for _, template := range nativeTemplates {
		if template.hidden {
			continue
		}
		tuple := cmdio.Tuple{
			Name: template.name,
			Id:   template.description,
		}
		names = append(names, tuple)
	}
	return names
}

func getNativeTemplateByDescription(description string) string {
	for _, template := range nativeTemplates {
		if template.description == description {
			return template.name
		}
	}
	return ""
}

func getUrlForNativeTemplate(name string) string {
	for _, template := range nativeTemplates {
		if template.name == name {
			return template.gitUrl
		}
		if slices.Contains(template.aliases, name) {
			return template.gitUrl
		}
	}
	return ""
}

func getFsForNativeTemplate(name string) (fs.FS, error) {
	builtin, err := template.Builtin()
	if err != nil {
		return nil, err
	}

	// If this is a built-in template, the return value will be non-nil.
	var templateFS fs.FS
	for _, entry := range builtin {
		if entry.Name == name {
			templateFS = entry.FS
			break
		}
	}

	return templateFS, nil
}

func isRepoUrl(url string) bool {
	result := false
	for _, prefix := range gitUrlPrefixes {
		if strings.HasPrefix(url, prefix) {
			result = true
			break
		}
	}
	return result
}

// Computes the repo name from the repo URL. Treats the last non empty word
// when splitting at '/' as the repo name. For example: for url git@github.com:databricks/cli.git
// the name would be "cli.git"
func repoName(url string) string {
	parts := strings.Split(strings.TrimRight(url, "/"), "/")
	return parts[len(parts)-1]
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

See https://docs.databricks.com/en/dev-tools/bundles/templates.html for more information on templates.`, nativeTemplateHelpDescriptions()),
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

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if tag != "" && branch != "" {
			return errors.New("only one of --tag or --branch can be specified")
		}

		// Git ref to use for template initialization
		ref := branch
		if tag != "" {
			ref = tag
		}

		ctx := cmd.Context()
		var templatePath string
		if len(args) > 0 {
			templatePath = args[0]
		} else {
			var err error
			if !cmdio.IsPromptSupported(ctx) {
				return errors.New("please specify a template")
			}
			description, err := cmdio.SelectOrdered(ctx, nativeTemplateOptions(), "Template to use")
			if err != nil {
				return err
			}
			templatePath = getNativeTemplateByDescription(description)
		}

		outputFiler, err := constructOutputFiler(ctx, outputDir)
		if err != nil {
			return err
		}

		if templatePath == customTemplate {
			cmdio.LogString(ctx, "Please specify a path or Git repository to use a custom template.")
			cmdio.LogString(ctx, "See https://docs.databricks.com/en/dev-tools/bundles/templates.html to learn more about custom templates.")
			return nil
		}

		// Expand templatePath to a git URL if it's an alias for a known native template
		// and we know it's git URL.
		if gitUrl := getUrlForNativeTemplate(templatePath); gitUrl != "" {
			templatePath = gitUrl
		}

		if !isRepoUrl(templatePath) {
			if templateDir != "" {
				return errors.New("--template-dir can only be used with a Git repository URL")
			}

			templateFS, err := getFsForNativeTemplate(templatePath)
			if err != nil {
				return err
			}

			// If this is not a built-in template, then it must be a local file system path.
			if templateFS == nil {
				templateFS = os.DirFS(templatePath)
			}

			// skip downloading the repo because input arg is not a URL. We assume
			// it's a path on the local file system in that case
			return template.Materialize(ctx, configFile, templateFS, outputFiler)
		}

		// Create a temporary directory with the name of the repository.  The '*'
		// character is replaced by a random string in the generated temporary directory.
		repoDir, err := os.MkdirTemp("", repoName(templatePath)+"-*")
		if err != nil {
			return err
		}

		// start the spinner
		promptSpinner := cmdio.Spinner(ctx)
		promptSpinner <- "Downloading the template\n"

		// TODO: Add automated test that the downloaded git repo is cleaned up.
		// Clone the repository in the temporary directory
		err = git.Clone(ctx, templatePath, ref, repoDir)
		close(promptSpinner)
		if err != nil {
			return err
		}

		// Clean up downloaded repository once the template is materialized.
		defer os.RemoveAll(repoDir)
		templateFS := os.DirFS(filepath.Join(repoDir, templateDir))
		return template.Materialize(ctx, configFile, templateFS, outputFiler)
	}
	return cmd
}
