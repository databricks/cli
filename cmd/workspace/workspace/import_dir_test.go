package workspace

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordingFiler captures Mkdir and Write calls so a test can assert which
// paths were visited by the import-dir walker.
type recordingFiler struct {
	dirs  []string
	files []string
}

func (r *recordingFiler) Mkdir(ctx context.Context, p string) error {
	r.dirs = append(r.dirs, p)
	return nil
}

func (r *recordingFiler) Write(ctx context.Context, p string, reader io.Reader, mode ...filer.WriteMode) error {
	r.files = append(r.files, p)
	return nil
}

func (r *recordingFiler) Read(ctx context.Context, p string) (io.ReadCloser, error) {
	return nil, fs.ErrNotExist
}

func (r *recordingFiler) Delete(ctx context.Context, p string, mode ...filer.DeleteMode) error {
	return nil
}

func (r *recordingFiler) ReadDir(ctx context.Context, p string) ([]fs.DirEntry, error) {
	return nil, fs.ErrNotExist
}

func (r *recordingFiler) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	return nil, fs.ErrNotExist
}

func writeFile(t *testing.T, root, rel, contents string) {
	t.Helper()
	full := filepath.Join(root, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte(contents), 0o644))
}

func runWalk(t *testing.T, sourceDir string) *recordingFiler {
	t.Helper()
	rec := &recordingFiler{}
	ctx := cmdio.InContext(t.Context(),
		cmdio.NewIO(t.Context(), flags.OutputText, nil, io.Discard, io.Discard, "", ""))
	opts := importDirOptions{sourceDir: sourceDir, targetDir: "/Workspace/x", overwrite: true}
	cb := opts.callback(ctx, rec)
	require.NoError(t, filepath.WalkDir(sourceDir, cb))
	return rec
}

func TestImportDirSkipsGitDirectory(t *testing.T) {
	src := t.TempDir()
	writeFile(t, src, "app.py", "print('hi')")
	writeFile(t, src, ".git/config", "[remote]\n  url = git@github.com:org/template.git")
	writeFile(t, src, ".git/HEAD", "ref: refs/heads/main")
	writeFile(t, src, ".git/objects/abc123", "binary")

	rec := runWalk(t, src)

	slices.Sort(rec.files)
	assert.Equal(t, []string{"app.py"}, rec.files)
	for _, d := range rec.dirs {
		assert.NotContains(t, d, ".git", "no .git directory should be created in the workspace")
	}
}

func TestImportDirSkipsNestedGitDirectory(t *testing.T) {
	src := t.TempDir()
	writeFile(t, src, "app.py", "print('hi')")
	writeFile(t, src, "vendor/sub/.git/config", "[remote]\n  url = ...")
	writeFile(t, src, "vendor/sub/lib.py", "def f(): pass")

	rec := runWalk(t, src)

	slices.Sort(rec.files)
	assert.Equal(t, []string{"app.py", filepath.ToSlash("vendor/sub/lib.py")}, rec.files)
	for _, d := range rec.dirs {
		assert.NotContains(t, d, ".git")
	}
}

func TestImportDirSkipsDatabricksCacheDirectory(t *testing.T) {
	src := t.TempDir()
	writeFile(t, src, "databricks.yml", "bundle:\n  name: x")
	writeFile(t, src, ".databricks/bundle/state.json", "{}")

	rec := runWalk(t, src)

	slices.Sort(rec.files)
	assert.Equal(t, []string{"databricks.yml"}, rec.files)
	for _, d := range rec.dirs {
		assert.NotContains(t, d, ".databricks")
	}
}

func TestImportDirCopiesGitignoreFile(t *testing.T) {
	src := t.TempDir()
	writeFile(t, src, ".gitignore", "*.pyc\n")
	writeFile(t, src, "app.py", "print('hi')")

	rec := runWalk(t, src)

	slices.Sort(rec.files)
	assert.Equal(t, []string{".gitignore", "app.py"}, rec.files)
}

func TestImportDirAllowsExplicitGitRoot(t *testing.T) {
	// If a user explicitly passes a .git directory as the source root, copy
	// it: the skip rule applies to .git dirs encountered during the walk,
	// not to a deliberately-named root.
	src := t.TempDir()
	gitRoot := filepath.Join(src, ".git")
	require.NoError(t, os.MkdirAll(gitRoot, 0o755))
	writeFile(t, gitRoot, "HEAD", "ref: refs/heads/main")
	writeFile(t, gitRoot, "config", "[core]\n")

	rec := runWalk(t, gitRoot)

	slices.Sort(rec.files)
	assert.Equal(t, []string{"HEAD", "config"}, rec.files)
}
