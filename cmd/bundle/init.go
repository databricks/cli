package bundle

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/git"
	"github.com/databricks/cli/libs/template"
	"github.com/spf13/cobra"
)

var gitUrlPrefixes = []string{
	"https://",
	"git@",
}

var aliasedTemplates = map[string]string{
	"mlops-stack":  "https://github.com/databricks/mlops-stacks",
	"mlops-stacks": "https://github.com/databricks/mlops-stacks",
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

func newInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [TEMPLATE_PATH]",
		Short: "Initialize using a bundle template",
		Args:  cobra.MaximumNArgs(1),
		Long: `Initialize using a bundle template.

TEMPLATE_PATH optionally specifies which template to use. It can be one of the following:
- 'default-python' for the default Python template
- a local file system path with a template directory
- a Git repository URL, e.g. https://github.com/my/repository

See https://docs.databricks.com/en/dev-tools/bundles/templates.html for more information on templates.`,
	}

	var configFile string
	var outputDir string
	var templateDir string
	var tag string
	var branch string
	cmd.Flags().StringVar(&configFile, "config-file", "", "File containing input parameters for template initialization.")
	cmd.Flags().StringVar(&templateDir, "template-dir", "", "Directory path within a Git repository containing the template.")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Directory to write the initialized template to.")
	cmd.Flags().StringVar(&branch, "tag", "", "Git tag to use for template initialization from remote repository")
	cmd.Flags().StringVar(&tag, "branch", "", "Git branch to use for template initialization from remote repository")

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
			if !cmdio.IsOutTTY(ctx) || !cmdio.IsInTTY(ctx) {
				return errors.New("please specify a template")
			}
			templatePath, err = cmdio.Ask(ctx, "Template to use", "default-python")
			if err != nil {
				return err
			}
		}

		// Expand templatePath if it's an alias for a known template
		if _, ok := aliasedTemplates[templatePath]; ok {
			templatePath = aliasedTemplates[templatePath]
		}

		if !isRepoUrl(templatePath) {
			if templateDir != "" {
				return errors.New("--template-dir can only be used with a Git repository URL")
			}
			// skip downloading the repo because input arg is not a URL. We assume
			// it's a path on the local file system in that case
			return template.Materialize(ctx, configFile, templatePath, outputDir)
		}

		// Create a temporary directory with the name of the repository.  The '*'
		// character is replaced by a random string in the generated temporary directory.
		repoDir, err := os.MkdirTemp("", repoName(templatePath)+"-*")
		if err != nil {
			return err
		}
		// TODO: Add automated test that the downloaded git repo is cleaned up.
		// Clone the repository in the temporary directory
		err = git.Clone(ctx, templatePath, ref, repoDir)
		if err != nil {
			return err
		}
		// Clean up downloaded repository once the template is materialized.
		defer os.RemoveAll(repoDir)
		return template.Materialize(ctx, configFile, filepath.Join(repoDir, templateDir), outputDir)
	}
	return cmd
}
