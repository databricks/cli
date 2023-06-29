package internal

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/git"
	"github.com/stretchr/testify/assert"
)

// TODO: add assertion for error if git CLI is not found
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
		Reference:      "snapshot",
		TargetDir:      tmpDir,
	})

	assert.NoError(t, err)
	assert.DirExists(t, filepath.Join(tmpDir, "cli-snapshot"))

	b, err := os.ReadFile(filepath.Join(tmpDir, "cli-snapshot/NOTICE"))
	assert.NoError(t, err)
	assert.Contains(t, string(b), "Copyright (2023) Databricks, Inc.")
}

// TODO(file an issue before merge): create a dedicated databricks private repository
// and test this for branches and tags
func TestAccGitClonePrivateRepository(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	tmpDir := t.TempDir()
	cmdIO := cmdio.NewIO("text", os.Stdin, os.Stdout, os.Stderr, "")
	ctx := cmdio.InContext(context.Background(), cmdIO)

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
