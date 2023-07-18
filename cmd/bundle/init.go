package bundle

import (
	"github.com/databricks/cli/libs/template"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init TEMPLATE_PATH",
	Short: "Initialize Template",
	Long:  `Initialize template`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return template.Materialize(args[0], outputDir, configFile)
	},
}

var configFile string
var outputDir string

// TODO: move integration tests to be unit tests OR increase coverage

func init() {
	initCmd.Flags().StringVar(&configFile, "config-file", "", "Input parameters for template initialization")
	initCmd.Flags().StringVar(&outputDir, "output-dir", "", "Directory to output the generated project into")
	initCmd.MarkFlagRequired("config-file")
	// TODO: make this flag optional and initialize into current directory.
	// Should we initialize into current directory?
	initCmd.MarkFlagRequired("output-dir")
	AddCommand(initCmd)
}
