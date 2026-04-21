package ucm

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// TestAuthVerbs_UsePreRunE guards against re-introducing the cmdIO panic caused
// by overriding PersistentPreRunE. Auth-requiring verbs must wire
// root.MustWorkspaceClient as PreRunE so the root's PersistentPreRunE (which
// installs cmdio) still executes. See https://github.com/micheledaddetta-databricks/cli/issues/TBD.
func TestAuthVerbs_UsePreRunE(t *testing.T) {
	root := New()
	authVerbs := map[string]bool{
		"plan":    true,
		"deploy":  true,
		"destroy": true,
		"summary": true,
	}

	seen := map[string]bool{}
	for _, sub := range root.Commands() {
		name := sub.Name()
		if !authVerbs[name] {
			continue
		}
		seen[name] = true
		assert.Nil(t, sub.PersistentPreRunE, "%s must not set PersistentPreRunE (would bypass root cmdio install)", name)
		assert.NotNil(t, sub.PreRunE, "%s must set PreRunE for workspace-client auth", name)
	}

	for verb := range authVerbs {
		assert.True(t, seen[verb], "verb %q not found under ucm root; test needs updating", verb)
	}
}

// assertNoSubtreeHasPersistentPreRunE walks the tree and fails if any descendant
// of `root` (except root itself) declares PersistentPreRunE. Kept exported for
// symmetry with stripAuthHooks.
func assertNoSubtreeHasPersistentPreRunE(t *testing.T, c *cobra.Command) {
	t.Helper()
	for _, sub := range c.Commands() {
		assert.Nil(t, sub.PersistentPreRunE, "%s has PersistentPreRunE which would shadow the root's cmdio setup", sub.CommandPath())
		assertNoSubtreeHasPersistentPreRunE(t, sub)
	}
}

func TestUcmSubtree_NoPersistentPreRunE(t *testing.T) {
	assertNoSubtreeHasPersistentPreRunE(t, New())
}
