package template

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testInMemoryFile(t *testing.T, ctx context.Context, perm fs.FileMode) {
	tmpDir := t.TempDir()

	f := &inMemoryFile{
		dstPath: &destinationPath{
			root:    tmpDir,
			relPath: "a/b/c",
		},
		perm:    perm,
		content: []byte("123"),
	}
	err := f.Write(ctx)
	assert.NoError(t, err)

	assertFileContent(t, filepath.Join(tmpDir, "a/b/c"), "123")
	assertFilePermissions(t, filepath.Join(tmpDir, "a/b/c"), perm)
}

func testCopyFile(t *testing.T, ctx context.Context, perm fs.FileMode) {
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "source"), []byte("qwerty"), perm)
	require.NoError(t, err)

	f := &copyFile{
		ctx: context.Background(),
		dstPath: &destinationPath{
			root:    tmpDir,
			relPath: "a/b/c",
		},
		perm:    perm,
		srcPath: "source",
		srcFS:   os.DirFS(tmpDir),
	}
	err = f.Write(ctx)
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
	ctx := context.Background()
	testInMemoryFile(t, ctx, 0755)
}

func TestTemplateInMemoryFilePersistToDiskForWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.SkipNow()
	}
	// we have separate tests for windows because of differences in valid
	// fs.FileMode values we can use for different operating systems.
	ctx := context.Background()
	testInMemoryFile(t, ctx, 0666)
}

func TestTemplateCopyFilePersistToDisk(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.SkipNow()
	}
	ctx := context.Background()
	testCopyFile(t, ctx, 0644)
}

func TestTemplateCopyFilePersistToDiskForWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.SkipNow()
	}
	// we have separate tests for windows because of differences in valid
	// fs.FileMode values we can use for different operating systems.
	ctx := context.Background()
	testCopyFile(t, ctx, 0666)
}
