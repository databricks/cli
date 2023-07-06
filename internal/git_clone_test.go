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

func TestAccGitClone(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	tmpDir := t.TempDir()
	cmdIO := cmdio.NewIO("text", os.Stdin, os.Stdout, os.Stderr, "")
	ctx := cmdio.InContext(context.Background(), cmdIO)
	var err error

	err = git.Clone(ctx, "https://github.com/ShreyasGoenka/empty-databricks-cli-repo.git", tmpDir)
	assert.NoError(t, err)

	// assert on repo content
	b, err := os.ReadFile(filepath.Join(tmpDir, "README.md"))
	assert.NoError(t, err)
	assert.Contains(t, string(b), "empty-databricks-cli-repo")

	// assert current branch is main, ie default for the repo
	b, err = os.ReadFile(filepath.Join(tmpDir, ".git/HEAD"))
	assert.NoError(t, err)
	assert.Contains(t, string(b), "main")
}

func TestAccGitCloneWithOrgAndRepoName(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	tmpDir := t.TempDir()
	cmdIO := cmdio.NewIO("text", os.Stdin, os.Stdout, os.Stderr, "")
	ctx := cmdio.InContext(context.Background(), cmdIO)
	var err error

	err = git.Clone(ctx, "ShreyasGoenka/empty-databricks-cli-repo@cli", tmpDir)

	assert.NoError(t, err)
	b, err := os.ReadFile(filepath.Join(tmpDir, "README.md"))
	assert.NoError(t, err)
	assert.Contains(t, string(b), "empty-databricks-cli-repo")

	// assert current branch is "cli"
	b, err = os.ReadFile(filepath.Join(tmpDir, ".git/HEAD"))
	assert.NoError(t, err)
	assert.Contains(t, string(b), "cli")
}

func TestAccGitCloneWithOnlyRepoName(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	tmpDir := t.TempDir()
	cmdIO := cmdio.NewIO("text", os.Stdin, os.Stdout, os.Stderr, "")
	ctx := cmdio.InContext(context.Background(), cmdIO)
	var err error

	err = git.Clone(ctx, "databricks-empty-ide-project", tmpDir)

	assert.NoError(t, err)
	b, err := os.ReadFile(filepath.Join(tmpDir, "README-IDE.md"))
	assert.NoError(t, err)
	assert.Contains(t, string(b), "This folder contains a project that was synchronized from an IDE.")
}
