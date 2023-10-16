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

var aliasedTemplates = map[string]string {
	"mlops-stack": "https://github.com/databricks/mlops-stack",
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
		Short: "Initialize Template",
		Args:  cobra.MaximumNArgs(1),
	}

	var configFile string
	var outputDir string
	var templateDir string
	cmd.Flags().StringVar(&configFile, "config-file", "", "File containing input parameters for template initialization.")
	cmd.Flags().StringVar(&templateDir, "template-dir", "", "Directory within repository that holds the template specification.")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Directory to write the initialized template to.")

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
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

		// Expand templatePath if it's an alias for a first party template
		if _, ok := aliasedTemplates[templatePath]; ok {
			templatePath = aliasedTemplates[templatePath]
		}

		if !isRepoUrl(templatePath) {
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
		err = git.Clone(ctx, templatePath, "", repoDir)
		if err != nil {
			return err
		}
		// Clean up downloaded repository once the template is materialized.
		defer os.RemoveAll(repoDir)
		return template.Materialize(ctx, configFile, filepath.Join(repoDir, templateDir), outputDir)
	}
	return cmd
}
