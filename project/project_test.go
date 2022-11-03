package project

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectInitialize(t *testing.T) {
	ctx, err := Initialize(context.Background(), "./testdata", DefaultEnvironment)
	require.NoError(t, err)
	assert.Equal(t, Get(ctx).config.Name, "dev")
}

func TestProjectInitializationCreatesGitIgnoreIfAbsent(t *testing.T) {
	// create project root with databricks.yml
	projectDir := t.TempDir()
	f1, err := os.Create(filepath.Join(projectDir, "databricks.yml"))
	assert.NoError(t, err)
	defer f1.Close()

	ctx, err := Initialize(context.Background(), projectDir, DefaultEnvironment)
	assert.NoError(t, err)

	gitIgnorePath := filepath.Join(projectDir, ".gitignore")
	assert.FileExists(t, gitIgnorePath)
	fileBytes, err := os.ReadFile(gitIgnorePath)
	assert.NoError(t, err)
	assert.Contains(t, string(fileBytes), ".databricks")

	prj := Get(ctx)
	_, err = prj.CacheDir()
	assert.NoError(t, err)
}

func TestProjectInitializationAddsCacheDirToGitIgnore(t *testing.T) {
	// create project root with databricks.yml
	projectDir := t.TempDir()
	f1, err := os.Create(filepath.Join(projectDir, "databricks.yml"))
	assert.NoError(t, err)
	defer f1.Close()

	gitIgnorePath := filepath.Join(projectDir, ".gitignore")
	f2, err := os.Create(gitIgnorePath)
	assert.NoError(t, err)
	defer f2.Close()

	ctx, err := Initialize(context.Background(), projectDir, DefaultEnvironment)
	assert.NoError(t, err)

	fileBytes, err := os.ReadFile(gitIgnorePath)
	assert.NoError(t, err)
	assert.Contains(t, string(fileBytes), ".databricks")

	prj := Get(ctx)
	_, err = prj.CacheDir()
	assert.NoError(t, err)
}

func TestProjectInitializationDoesNotAddCacheDirToGitIgnoreIfAlreadyPresent(t *testing.T) {
	// create project root with databricks.yml
	projectDir := t.TempDir()
	f1, err := os.Create(filepath.Join(projectDir, "databricks.yml"))
	assert.NoError(t, err)
	defer f1.Close()

	gitIgnorePath := filepath.Join(projectDir, ".gitignore")

	err = os.WriteFile(gitIgnorePath, []byte("/.databricks/"), 0o644)
	assert.NoError(t, err)

	_, err = Initialize(context.Background(), projectDir, DefaultEnvironment)
	assert.NoError(t, err)

	fileBytes, err := os.ReadFile(gitIgnorePath)
	assert.NoError(t, err)

	assert.Equal(t, 1, strings.Count(string(fileBytes), ".databricks"))
}

func TestProjectCacheDir(t *testing.T) {
	// create project root with databricks.yml
	projectDir := t.TempDir()
	f1, err := os.Create(filepath.Join(projectDir, "databricks.yml"))
	assert.NoError(t, err)
	defer f1.Close()

	// create .gitignore with the .databricks dir in it
	f2, err := os.Create(filepath.Join(projectDir, ".gitignore"))
	assert.NoError(t, err)
	defer f2.Close()
	content := []byte("/.databricks/")
	_, err = f2.Write(content)
	assert.NoError(t, err)

	ctx, err := Initialize(context.Background(), projectDir, DefaultEnvironment)
	assert.NoError(t, err)

	prj := Get(ctx)
	cacheDir, err := prj.CacheDir()
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join(projectDir, ".databricks"), cacheDir)
}
