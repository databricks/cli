package template

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testInMemoryFile(t *testing.T, ctx context.Context, executable bool) {
	tmpDir := t.TempDir()

	perm := fs.FileMode(0o644)
	if executable {
		perm = 0o755
	}
	f := &inMemoryFile{
		perm:    perm,
		relPath: "a/b/c",
		content: []byte("123"),
	}

	out, err := filer.NewLocalClient(tmpDir)
	require.NoError(t, err)
	err = f.Write(ctx, out)
	assert.NoError(t, err)

	testutil.AssertFileContents(t, filepath.Join(tmpDir, "a/b/c"), "123")
	testutil.AssertFileOwnerExec(t, filepath.Join(tmpDir, "a/b/c"), executable)
}

func testCopyFile(t *testing.T, ctx context.Context, executable bool) {
	tmpDir := t.TempDir()

	perm := fs.FileMode(0o644)
	if executable {
		perm = 0o755
	}
	err := os.WriteFile(filepath.Join(tmpDir, "source"), []byte("qwerty"), perm)
	require.NoError(t, err)

	f := &copyFile{
		perm:    perm,
		relPath: "a/b/c",
		srcFS:   os.DirFS(tmpDir),
		srcPath: "source",
	}

	out, err := filer.NewLocalClient(tmpDir)
	require.NoError(t, err)
	err = f.Write(ctx, out)
	assert.NoError(t, err)

	testutil.AssertFileContents(t, filepath.Join(tmpDir, "source"), "qwerty")
	testutil.AssertFileOwnerExec(t, filepath.Join(tmpDir, "source"), executable)
}

func TestTemplateInMemoryFilePersistToDisk(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.SkipNow()
	}
	ctx := t.Context()
	testInMemoryFile(t, ctx, true)
}

func TestTemplateInMemoryFilePersistToDiskForWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.SkipNow()
	}
	// we have separate tests for windows because of differences in valid
	// fs.FileMode values we can use for different operating systems.
	ctx := t.Context()
	testInMemoryFile(t, ctx, false)
}

func TestTemplateCopyFilePersistToDisk(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.SkipNow()
	}
	ctx := t.Context()
	testCopyFile(t, ctx, false)
}

func TestTemplateCopyFilePersistToDiskForWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.SkipNow()
	}
	// we have separate tests for windows because of differences in valid
	// fs.FileMode values we can use for different operating systems.
	ctx := t.Context()
	testCopyFile(t, ctx, false)
}
