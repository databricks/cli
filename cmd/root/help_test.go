package root

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helpRoot returns a root with a group "grp" that has subcommands, and a
// group "other" that no skill covers.
func helpRoot(ctx context.Context) *cobra.Command {
	cmd := newRootCommand(ctx)
	for _, name := range []string{"grp", "other"} {
		group := &cobra.Command{Use: name, Short: name + " group"}
		group.AddCommand(&cobra.Command{Use: "child", Short: "child", Run: func(*cobra.Command, []string) {}})
		cmd.AddCommand(group)
	}
	return cmd
}

// helpOutput runs args on cmd and returns what it printed.
func helpOutput(t *testing.T, ctx context.Context, cmd *cobra.Command, args ...string) string {
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	require.NoError(t, cmd.ExecuteContext(ctx))
	return buf.String()
}

// skillsDir writes a skills release whose one skill covers the group "grp".
func skillsDir(t *testing.T) string {
	dir := t.TempDir()
	manifest := `{"version": "1", "skills": {"alpha": {"version": "1", "files": ["SKILL.md"], "commands": ["grp"]}}}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644))
	return dir
}

func TestHelpHintsOffIsUnchanged(t *testing.T) {
	ctx := env.Set(t.Context(), "DATABRICKS_SKILLS_DIR", skillsDir(t))
	ctx = env.Set(ctx, "DATABRICKS_SKILLS_PREVIEW", "1")

	for _, args := range [][]string{{"--help"}, {"grp", "--help"}} {
		out := helpOutput(t, ctx, helpRoot(ctx), args...)
		assert.NotContains(t, out, "databricks skills")
	}
}

func TestHelpHintsOnRootEndsWithGenericLine(t *testing.T) {
	ctx := env.Set(t.Context(), "DATABRICKS_SKILLS_PREVIEW", "1")
	ctx = env.Set(ctx, "DATABRICKS_SKILLS_HINTS", "1")

	out := helpOutput(t, ctx, helpRoot(ctx), "--help")
	assert.True(t, strings.HasSuffix(out, "\n\n"+rootSkillsHint+"\n"), out)
}

func TestHelpHintsOnGroupFooter(t *testing.T) {
	ctx := env.Set(t.Context(), "DATABRICKS_SKILLS_PREVIEW", "1")
	ctx = env.Set(ctx, "DATABRICKS_SKILLS_HINTS", "1")
	ctx = env.Set(ctx, "DATABRICKS_SKILLS_DIR", skillsDir(t))

	out := helpOutput(t, ctx, helpRoot(ctx), "grp", "--help")
	assert.True(t, strings.HasSuffix(out, "\n\nCoding agents: read the skills for this command group before you use it.\n  databricks skills read alpha\n"), out)

	out = helpOutput(t, ctx, helpRoot(ctx), "other", "--help")
	assert.NotContains(t, out, "databricks skills read")
	assert.NotContains(t, out, "Coding agents")
}

func TestHelpHintsOnWithoutSourceHasNoGroupFooter(t *testing.T) {
	ctx := env.Set(t.Context(), "DATABRICKS_SKILLS_PREVIEW", "1")
	ctx = env.Set(ctx, "DATABRICKS_SKILLS_HINTS", "1")
	ctx = env.Set(ctx, "DATABRICKS_SKILLS_DIR", "")

	out := helpOutput(t, ctx, helpRoot(ctx), "grp", "--help")
	assert.NotContains(t, out, "databricks skills read")
}

func TestHelpHintsOnLeafCommandHasNoFooter(t *testing.T) {
	ctx := env.Set(t.Context(), "DATABRICKS_SKILLS_PREVIEW", "1")
	ctx = env.Set(ctx, "DATABRICKS_SKILLS_HINTS", "1")
	ctx = env.Set(ctx, "DATABRICKS_SKILLS_DIR", skillsDir(t))

	out := helpOutput(t, ctx, helpRoot(ctx), "grp", "child", "--help")
	assert.NotContains(t, out, "databricks skills read")
}
