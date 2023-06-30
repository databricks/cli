package internal

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/git"
	"github.com/stretchr/testify/assert"
)

func TestAccGitClonePublicRepository(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	tmpDir := t.TempDir()
	ctx := context.Background()
	var err error

	// We unset PATH to ensure that git.Clone cannot rely on the git CLI
	t.Setenv("PATH", "")

	err = git.Clone(ctx, git.CloneOptions{
		Provider:       "github",
		Organization:   "databricks",
		RepositoryName: "cli",
		Reference:      "main",
		TargetDir:      tmpDir,
	})

	assert.NoError(t, err)
	assert.DirExists(t, filepath.Join(tmpDir, "cli-main"))

	b, err := os.ReadFile(filepath.Join(tmpDir, "cli-main/NOTICE"))
	assert.NoError(t, err)
	assert.Contains(t, string(b), "Copyright (2023) Databricks, Inc.")
}

func TestAccGitClonePublicRepositoryForTagReference(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	tmpDir := t.TempDir()
	ctx := context.Background()
	var err error

	// We unset PATH to ensure that git.Clone cannot rely on the git CLI
	t.Setenv("PATH", "")

	err = git.Clone(ctx, git.CloneOptions{
		Provider:       "github",
		Organization:   "databricks",
		RepositoryName: "cli",
		Reference:      "snapshot",
		TargetDir:      tmpDir,
	})

	assert.NoError(t, err)
	assert.DirExists(t, filepath.Join(tmpDir, "cli-snapshot"))

	b, err := os.ReadFile(filepath.Join(tmpDir, "cli-snapshot/NOTICE"))
	assert.NoError(t, err)
	assert.Contains(t, string(b), "Copyright (2023) Databricks, Inc.")
}
