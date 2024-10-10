package template

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/mock"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testInMemoryFile(t *testing.T, perm fs.FileMode) {
	tmpDir := t.TempDir()

	f := &inMemoryFile{
		ctx: context.Background(),
		dstPath: &destinationPath{
			root:    tmpDir,
			relPath: "a/b/c",
		},
		perm:    perm,
		content: []byte("123"),
	}
	err := f.PersistToDisk()
	assert.NoError(t, err)

	assertFileContent(t, filepath.Join(tmpDir, "a/b/c"), "123")
	assertFilePermissions(t, filepath.Join(tmpDir, "a/b/c"), perm)
}

func testCopyFile(t *testing.T, perm fs.FileMode) {
	tmpDir := t.TempDir()

	templateFiler, err := filer.NewLocalClient(tmpDir)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "source"), []byte("qwerty"), perm)
	require.NoError(t, err)

	f := &copyFile{
		ctx: context.Background(),
		dstPath: &destinationPath{
			root:    tmpDir,
			relPath: "a/b/c",
		},
		perm:     perm,
		srcPath:  "source",
		srcFiler: templateFiler,
	}
	err = f.PersistToDisk()
	assert.NoError(t, err)

	assertFileContent(t, filepath.Join(tmpDir, "a/b/c"), "qwerty")
	assertFilePermissions(t, filepath.Join(tmpDir, "a/b/c"), perm)
}

func TestTemplateFileDestinationPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.SkipNow()
	}
	f := &destinationPath{
		root:    `a/b/c`,
		relPath: "d/e",
	}
	assert.Equal(t, `a/b/c/d/e`, f.absPath())
}

func TestTemplateFileDestinationPathForWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.SkipNow()
	}
	f := &destinationPath{
		root:    `c:\a\b\c`,
		relPath: "d/e",
	}
	assert.Equal(t, `c:\a\b\c\d\e`, f.absPath())
}

func TestTemplateInMemoryFilePersistToDisk(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.SkipNow()
	}
	testInMemoryFile(t, 0755)
}

func TestTemplateInMemoryFilePersistToDiskForWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.SkipNow()
	}
	// we have separate tests for windows because of differences in valid
	// fs.FileMode values we can use for different operating systems.
	testInMemoryFile(t, 0666)
}

func TestTemplateCopyFilePersistToDisk(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.SkipNow()
	}
	testCopyFile(t, 0644)
}

func TestTemplateCopyFilePersistToDiskForWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.SkipNow()
	}
	// we have separate tests for windows because of differences in valid
	// fs.FileMode values we can use for different operating systems.
	testCopyFile(t, 0666)
}

func TestShouldUseImportNotebook(t *testing.T) {
	ctx := context.Background()
	data := []byte("# Databricks notebook source\n print('hello')")

	assert.False(t, shouldUseImportNotebook(ctx, "./foo/bar", data))
	assert.False(t, shouldUseImportNotebook(ctx, "./foo/bar.ipynb", data))
	assert.False(t, shouldUseImportNotebook(ctx, "/Workspace/foo/bar", data))
	assert.False(t, shouldUseImportNotebook(ctx, "/Workspace/foo/bar.ipynb", data))

	t.Setenv("DATABRICKS_RUNTIME_VERSION", "14.3")
	assert.False(t, shouldUseImportNotebook(ctx, "./foo/bar", data))
	assert.False(t, shouldUseImportNotebook(ctx, "./foo/bar.ipynb", data))
	assert.False(t, shouldUseImportNotebook(ctx, "/Workspace/foo/bar", data))
	assert.True(t, shouldUseImportNotebook(ctx, "/Workspace/foo/bar.py", data))
}

func TestImportNotebook(t *testing.T) {
	ctx := context.Background()

	m := mocks.NewMockWorkspaceClient(t)
	ctx = root.SetWorkspaceClient(ctx, m.WorkspaceClient)

	workspaceApi := m.GetMockWorkspaceAPI()
	workspaceApi.EXPECT().Import(mock.Anything, workspace.Import{
		Content:   "cXdlcnR5", // base64 of "qwerty"
		Format:    "AUTO",
		Overwrite: false,
		Path:      "/Workspace/foo/bar.ipynb",
	}).Return(nil)

	err := importNotebook(ctx, "/Workspace/foo/bar.ipynb", []byte("qwerty"))
	assert.NoError(t, err)
}
