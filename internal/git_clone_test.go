package internal

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/git"
	"github.com/stretchr/testify/assert"
)

// TODO: add assertion that the git tool is not called, maybe by voiding PATH
// TODO: add assertion for error if git CLI is not found
func TestGitClonePublicRepository(t *testing.T) {
	// t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

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

func TestAccGitClonePrivateRepository(t *testing.T) {
	// t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	tmpDir := t.TempDir()
	ctx := context.Background()

	// This is a private repository only accessible to databricks employees
	err := git.Clone(ctx, git.CloneOptions{
		Provider:       "github",
		Organization:   "databricks",
		RepositoryName: "bundle-samples-internal",
		Reference:      "main",
		TargetDir:      tmpDir,
	})
	assert.NoError(t, err)

	// assert examples from the private repository
	assert.DirExists(t, filepath.Join(tmpDir, "bundle-samples-internal-main", "shark_sightings"))
	assert.DirExists(t, filepath.Join(tmpDir, "bundle-samples-internal-main", "wikipedia_clickstream"))
}
