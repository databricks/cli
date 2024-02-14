package bundle

import (
	"github.com/spf13/cobra"
)

func initVariableFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().StringSlice("var", []string{}, `set values for variables defined in bundle config. Example: --var="foo=bar"`)
}
