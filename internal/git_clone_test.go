package internal

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/git"
	"github.com/stretchr/testify/assert"
)

func TestAccGitClone(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	tmpDir := t.TempDir()
	ctx := context.Background()
	var err error

	err = git.Clone(ctx, "https://github.com/databricks/databricks-empty-ide-project.git", "", tmpDir)
	assert.NoError(t, err)

	// assert repo content
	assert.NoError(t, err)
	b, err := os.ReadFile(filepath.Join(tmpDir, "README-IDE.md"))
	assert.NoError(t, err)
	assert.Contains(t, string(b), "This folder contains a project that was synchronized from an IDE.")

	// assert current branch is ide, ie default for the repo
	b, err = os.ReadFile(filepath.Join(tmpDir, ".git/HEAD"))
	assert.NoError(t, err)
	assert.Contains(t, string(b), "ide")
}

func TestAccGitCloneWithOnlyRepoNameOnAlternateBranch(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	tmpDir := t.TempDir()
	ctx := context.Background()
	var err error

	err = git.Clone(ctx, "notebook-best-practices", "dais-2022", tmpDir)

	// assert on repo content
	assert.NoError(t, err)
	b, err := os.ReadFile(filepath.Join(tmpDir, "README.md"))
	assert.NoError(t, err)
	assert.Contains(t, string(b), "Software engineering best practices for Databricks notebooks")

	// assert current branch is main, ie default for the repo
	b, err = os.ReadFile(filepath.Join(tmpDir, ".git/HEAD"))
	assert.NoError(t, err)
	assert.Contains(t, string(b), "dais-2022")
}

func TestAccGitCloneRepositoryDoesNotExist(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	tmpDir := t.TempDir()

	err := git.Clone(context.Background(), "doesnot-exist", "", tmpDir)
	assert.Contains(t, err.Error(), `repository 'https://github.com/databricks/doesnot-exist/' not found`)
}
