package bundle

import (
	"github.com/spf13/cobra"
)

func initVariableFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().StringSlice("var", []string{}, `set values for variables defined in bundle config. Example: --var="foo=bar"`)
	cmd.PersistentFlags().String("var-file", "", `file path to a JSON file containing variables. Example: --var-file="/path/to/vars.json"`)
}
