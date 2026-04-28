package ucm

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestUcmSubtree_NoPersistentPreRunE walks the entire ucm cobra subtree and
// asserts no command sets PersistentPreRunE. The auth chain is intentionally
// routed through ProcessUcm; setting PersistentPreRunE on a verb would cause
// the cmdio TUI panic seen in #41/#45 (interactive prompts firing before
// stdout is buffered).
func TestUcmSubtree_NoPersistentPreRunE(t *testing.T) {
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		if c.PersistentPreRunE != nil || c.PersistentPreRun != nil {
			t.Errorf("command %q sets PersistentPreRun(E); ucm verbs route auth through ProcessUcm — see #41/#45", c.CommandPath())
		}
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(New())
}
