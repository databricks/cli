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

	err = git.Clone(ctx, "https://github.com/databricks/cli.git", tmpDir)
	assert.NoError(t, err)

	// assert on repo content
	b, err := os.ReadFile(filepath.Join(tmpDir, "NOTICE"))
	assert.NoError(t, err)
	assert.Contains(t, string(b), "Copyright (2023) Databricks, Inc.")

	// assert current branch is main
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

	err = git.Clone(ctx, "databricks/cli", tmpDir)

	assert.NoError(t, err)
	b, err := os.ReadFile(filepath.Join(tmpDir, "NOTICE"))
	assert.NoError(t, err)
	assert.Contains(t, string(b), "Copyright (2023) Databricks, Inc.")
}

func TestAccGitCloneWithOnlyRepoName(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	tmpDir := t.TempDir()
	cmdIO := cmdio.NewIO("text", os.Stdin, os.Stdout, os.Stderr, "")
	ctx := cmdio.InContext(context.Background(), cmdIO)
	var err error

	err = git.Clone(ctx, "cli", tmpDir)

	assert.NoError(t, err)
	b, err := os.ReadFile(filepath.Join(tmpDir, "NOTICE"))
	assert.NoError(t, err)
	assert.Contains(t, string(b), "Copyright (2023) Databricks, Inc.")

}
