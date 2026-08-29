package installer

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDumpSkillsToPathWritesFilesNoState(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	dest := filepath.Join(t.TempDir(), "out")
	src := &mockManifestSource{manifest: testManifest()}

	n, err := DumpSkillsToPath(ctx, src, dest, InstallOptions{})
	require.NoError(t, err)
	assert.Equal(t, 2, n)

	for _, name := range []string{"databricks-sql", "databricks-jobs"} {
		_, err := os.Stat(filepath.Join(dest, name, "SKILL.md"))
		assert.NoError(t, err)
	}

	// A dumb dump writes no state and no agent dirs.
	_, err = os.Stat(filepath.Join(dest, stateFileName))
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestDumpSkillsToPathCherryPick(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	setupFetchMock(t)
	t.Setenv("DATABRICKS_SKILLS_REF", testSkillsRef)

	dest := filepath.Join(t.TempDir(), "out")
	src := &mockManifestSource{manifest: testManifest()}

	n, err := DumpSkillsToPath(ctx, src, dest, InstallOptions{SpecificSkills: []string{"databricks-sql"}})
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	_, err = os.Stat(filepath.Join(dest, "databricks-sql", "SKILL.md"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(dest, "databricks-jobs"))
	assert.ErrorIs(t, err, fs.ErrNotExist)
}
