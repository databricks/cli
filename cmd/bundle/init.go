package bundle

import (
	"github.com/databricks/cli/libs/template"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init TEMPLATE_PATH INSTANCE_PATH",
	Short: "Initialize Template",
	Long:  `Initialize template`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return template.Materialize(args[0], args[1], configFile)
	},
}

var configFile string

func init() {
	initCmd.Flags().StringVar(&configFile, "config-file", "", "input parameters for template initialization")
	initCmd.MarkFlagRequired("config-file")
	AddCommand(initCmd)
}
