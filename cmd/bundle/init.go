package bundle

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/git"
	"github.com/databricks/cli/libs/template"
	"github.com/spf13/cobra"
)

var gitUrlPrefixes = []string{
	"https://",
	"git@",
}

func newInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init TEMPLATE_PATH",
		Short: "Initialize Template",
		Args:  cobra.ExactArgs(1),
	}

	var configFile string
	var projectDir string
	cmd.Flags().StringVar(&configFile, "config-file", "", "Input parameters for template initialization.")
	cmd.Flags().StringVar(&projectDir, "project-dir", "", "The project will be initialized in this directory.")
	cmd.MarkFlagRequired("output-dir")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		templateURL := args[0]
		ctx := cmd.Context()

		isRepo := false
		for _, prefix := range gitUrlPrefixes {
			if strings.HasPrefix(templateURL, prefix) {
				isRepo = true
				break
			}
		}
		if !isRepo {
			// skip downloading the repo because input arg is not a URL. We assume
			// it's a path on the local file system in that case
			return template.Materialize(ctx, configFile, templateURL, projectDir)
		}

		// Download the template in a temporary directory
		tmpDir := os.TempDir()
		templateDir := filepath.Join(tmpDir, templateURL)
		err := os.MkdirAll(templateDir, 0755)
		if err != nil {
			return err
		}
		err = git.Clone(ctx, templateURL, "", templateDir)
		if err != nil {
			return err
		}

		return template.Materialize(ctx, configFile, templateDir, projectDir)
	}

	return cmd
}
