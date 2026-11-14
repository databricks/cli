package aircmd

import (
	"bytes"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// airUnderRoot mounts the real air tree under a root whose pre-run hook only
// records that it ran, mirroring where cmd/root calls agents.MaybeHint.
func airUnderRoot(preRunCalled *bool) (*cobra.Command, *cobra.Command) {
	root := &cobra.Command{Use: "databricks"}
	root.AddGroup(&cobra.Group{ID: "development", Title: "Developer Tools"})
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		*preRunCalled = true
		return nil
	}
	air := New()
	root.AddCommand(air)
	return root, air
}

// TestBareAirAndHelpShowFooterWithoutPreRun pins why the footer, not the
// dynamic hint, is the recommendation surface for `air` and `air --help`.
func TestBareAirAndHelpShowFooterWithoutPreRun(t *testing.T) {
	for _, args := range [][]string{{"air"}, {"air", "--help"}} {
		t.Run(args[len(args)-1], func(t *testing.T) {
			preRunCalled := false
			root, _ := airUnderRoot(&preRunCalled)
			var out bytes.Buffer
			root.SetOut(&out)
			root.SetErr(&out)
			root.SetArgs(args)
			require.NoError(t, root.Execute())
			assert.False(t, preRunCalled, "help must not run the pre-run hook")
			assert.Contains(t, out.String(), "Coding agent skill       databricks aitools install --skills-only --skills databricks-ai-runtime")
			assert.Contains(t, out.String(), "Preview the skill        databricks aitools install --path <dir> --skills databricks-ai-runtime")
		})
	}
}

// TestRunnableAirCommandsGetSkillHint walks the real air tree, so a new
// subcommand is covered without listing it here.
func TestRunnableAirCommandsGetSkillHint(t *testing.T) {
	testutil.SetAgentEnv(t, "CLAUDECODE", "1")
	preRunCalled := false
	_, air := airUnderRoot(&preRunCalled)
	assert.False(t, air.Runnable(), "bare air renders help; the footer covers it")

	var runnable []string
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			if sub.Runnable() {
				runnable = append(runnable, sub.CommandPath())
			}
			walk(sub)
		}
	}
	walk(air)
	for _, name := range []string{"run", "get", "list", "logs", "cancel", "convert-to-dabs"} {
		assert.Contains(t, runnable, "databricks air "+name)
	}

	var visit func(c *cobra.Command)
	visit = func(c *cobra.Command) {
		for _, sub := range c.Commands() {
			if sub.Runnable() {
				t.Run(sub.CommandPath(), func(t *testing.T) {
					t.Chdir(t.TempDir())
					ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
					ctx = env.WithUserHomeDir(ctx, t.TempDir())
					ctx = env.Set(ctx, "DATABRICKS_CACHE_DIR", t.TempDir())
					agents.MaybeHint(ctx, sub)
					assert.Contains(t, stderr.String(), "the databricks-ai-runtime skill is not installed")
				})
			}
			visit(sub)
		}
	}
	visit(air)
}
