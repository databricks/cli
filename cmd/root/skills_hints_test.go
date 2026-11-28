package root

import (
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/clicompat"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/skills"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A cache refresh during a command, such as the auto-install after it, must not
// change skills hints for that run: execute resolves them once.
func TestExecuteSkillHintsOncePerRun(t *testing.T) {
	testutil.CleanupEnvironment(t)
	ctx := env.Set(t.Context(), "DATABRICKS_SKILLS_PREVIEW", "true")
	ctx = env.Set(ctx, "DATABRICKS_CACHE_DIR", t.TempDir())
	path := clicompat.LocalManifestPath(ctx)
	// One entry applies to every CLI version.
	testutil.WriteFile(t, path, `{"0.1.0": {"appkit": "0.84.0", "skills": "0.2.30", "skills_hints": true}}`)

	var before, after bool
	cmd := &cobra.Command{
		Use:           "test",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			before = skills.HintsEnabled(cmd.Context())
			testutil.WriteFile(t, path, `{"0.1.0": {"appkit": "0.84.0", "skills": "0.2.30"}}`)
			after = skills.HintsEnabled(cmd.Context())
			return nil
		},
	}
	require.NoError(t, execute(ctx, cmd, nil))
	require.True(t, before)
	assert.True(t, after, "a cache refresh during the command changed skills hints for that run")
}
