package testutil

import (
	"strings"

	"github.com/spf13/cobra"
)

// CommandAt builds a command tree under a "databricks" root and returns the
// command at the space-separated path, e.g. "current-user me". Use it to test
// code that inspects a command's position in the tree.
func CommandAt(path string) *cobra.Command {
	cmd := &cobra.Command{Use: "databricks"}
	for name := range strings.FieldsSeq(path) {
		child := &cobra.Command{Use: name}
		cmd.AddCommand(child)
		cmd = child
	}
	return cmd
}
