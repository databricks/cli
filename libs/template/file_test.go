package template

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateFileCommonPathForWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.SkipNow()
	}
	f := &fileCommon{
		root:    `c:\a\b\c`,
		relPath: "d/e",
	}
	assert.Equal(t, `c:\a\b\c\d\e`, f.Path())
	assert.Equal(t, `d/e`, f.RelPath())
}

func TestTemplateFileCommonPathForUnix(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.SkipNow()
	}
	f := &fileCommon{
		root:    `a/b/c`,
		relPath: "d/e",
	}
	assert.Equal(t, `a/b/c/d/e`, f.Path())
	assert.Equal(t, `d/e`, f.RelPath())
}

func TestTemplateFileInMemoryFilePersistToDisk(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.SkipNow()
	}
	tmpDir := t.TempDir()

	f := &inMemoryFile{
		fileCommon: &fileCommon{
			root:    tmpDir,
			relPath: "a/b/c",
			perm:    0755,
		},
		content: []byte("123"),
	}
	err := f.PersistToDisk()
	assert.NoError(t, err)

	assertFileContent(t, filepath.Join(tmpDir, "a/b/c"), "123")
	assertFilePermissions(t, filepath.Join(tmpDir, "a/b/c"), 0755)
}

func TestTemplateFileCopyFilePersistToDisk(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.SkipNow()
	}
	tmpDir := t.TempDir()

	templateFiler, err := filer.NewLocalClient(tmpDir)
	require.NoError(t, err)
	os.WriteFile(filepath.Join(tmpDir, "source"), []byte("qwerty"), 0644)

	f := &copyFile{
		ctx: context.Background(),
		fileCommon: &fileCommon{
			root:    tmpDir,
			relPath: "a/b/c",
			perm:    0644,
		},
		srcPath:  "source",
		srcFiler: templateFiler,
	}
	err = f.PersistToDisk()
	assert.NoError(t, err)

	assertFileContent(t, filepath.Join(tmpDir, "a/b/c"), "qwerty")
	assertFilePermissions(t, filepath.Join(tmpDir, "a/b/c"), 0644)
}

// we have separate tests for windows because of differences in valid
// fs.FileMode values we can use for different operating systems.
func TestTemplateFileInMemoryFilePersistToDiskForWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.SkipNow()
	}
	tmpDir := t.TempDir()

	f := &inMemoryFile{
		fileCommon: &fileCommon{
			root:    tmpDir,
			relPath: "a/b/c",
			perm:    0666,
		},
		content: []byte("123"),
	}
	err := f.PersistToDisk()
	assert.NoError(t, err)

	assertFileContent(t, filepath.Join(tmpDir, "a/b/c"), "123")
	assertFilePermissions(t, filepath.Join(tmpDir, "a/b/c"), 0666)
}

// we have separate tests for windows because of differences in valid
// fs.FileMode values we can use for different operating systems.
func TestTemplateFileCopyFilePersistToDiskForWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.SkipNow()
	}
	tmpDir := t.TempDir()

	templateFiler, err := filer.NewLocalClient(tmpDir)
	require.NoError(t, err)
	os.WriteFile(filepath.Join(tmpDir, "source"), []byte("qwerty"), 0666)

	f := &copyFile{
		ctx: context.Background(),
		fileCommon: &fileCommon{
			root:    tmpDir,
			relPath: "a/b/c",
			perm:    0666,
		},
		srcPath:  "source",
		srcFiler: templateFiler,
	}
	err = f.PersistToDisk()
	assert.NoError(t, err)

	assertFileContent(t, filepath.Join(tmpDir, "a/b/c"), "qwerty")
	assertFilePermissions(t, filepath.Join(tmpDir, "a/b/c"), 0666)
}
