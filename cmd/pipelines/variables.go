package pipelines

import (
	"github.com/spf13/cobra"
)

// Copied from cmd/bundle/variables.go
func initVariableFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().StringSlice("var", []string{}, `set values for variables defined in project config. Example: --var="foo=bar"`)
}
