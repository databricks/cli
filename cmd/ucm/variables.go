package ucm

import (
	"github.com/spf13/cobra"
)

// InitVariableFlag wires the persistent --var flag onto cmd. Mirrors
// cmd/bundle/variables.go::initVariableFlag for the ucm subtree.
func InitVariableFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().StringSlice("var", []string{}, `set values for variables defined in ucm config. Example: --var="foo=bar"`)
}
