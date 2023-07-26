package bundle

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/git"
	"github.com/databricks/cli/libs/template"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init TEMPLATE_PATH",
	Short: "Initialize Template",
	Long:  `Initialize template`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		templateURL := args[0]
		tmpDir := os.TempDir()
		templateDir := filepath.Join(tmpDir, templateURL)
		ctx := cmd.Context()

		err := os.MkdirAll(templateDir, 0755)
		if err != nil {
			return err
		}

		// TODO: should we delete this directory once we are done with it?
		// It's a destructive action that can be risky
		err = git.Clone(ctx, templateURL, "", templateDir)
		if err != nil {
			return err
		}

		// TODO: substitute to a read config method that respects the schema
		// and prompts for input variables
		b, err := os.ReadFile(configFile)
		if err != nil {
			return err
		}
		config := make(map[string]any)
		err = json.Unmarshal(b, &config)
		if err != nil {
			return err
		}
		return template.Materialize(ctx, config, templateDir, projectDir)
	},
}

var configFile string
var projectDir string

func init() {
	initCmd.Flags().StringVar(&configFile, "config-file", "", "Input parameters for template initialization.")
	initCmd.Flags().StringVar(&projectDir, "project-dir", "", "The project will be initialized in this directory.")
	initCmd.MarkFlagRequired("config-file")
	initCmd.MarkFlagRequired("output-dir")
	AddCommand(initCmd)
}
