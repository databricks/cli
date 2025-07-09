package git

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func copyTestdata(t *testing.T, name string) string {
	tempDir := t.TempDir()

	// Copy everything under testdata ${name} to temporary directory.
	err := filepath.WalkDir(name, func(path string, d fs.DirEntry, err error) error {
		require.NoError(t, err)

		if d.IsDir() {
			err := os.MkdirAll(filepath.Join(tempDir, path), 0o755)
			require.NoError(t, err)
			return nil
		}

		fin, err := os.Open(path)
		require.NoError(t, err)
		defer fin.Close()

		fout, err := os.Create(filepath.Join(tempDir, path))
		require.NoError(t, err)
		defer fout.Close()

		_, err = io.Copy(fout, fin)
		require.NoError(t, err)
		return nil
	})

	require.NoError(t, err)
	return filepath.Join(tempDir, name)
}

func createFakeRepo(t *testing.T, testdataName string) string {
	absPath := copyTestdata(t, testdataName)

	// Add .git directory to make it look like a Git repository.
	err := os.Mkdir(filepath.Join(absPath, ".git"), 0o755)
	require.NoError(t, err)
	return absPath
}

// Wrap a View and expose a panicking version of [View.Ignore].
type testView struct {
	t *testing.T
	v *View
}

func (v *testView) Ignore(relPath string) bool {
	ign, err := v.v.Ignore(relPath)
	require.NoError(v.t, err)
	return ign
}

func testViewAtRoot(t *testing.T, tv testView) {
	// Check .gitignore at root.
	assert.True(t, tv.Ignore("root.sh"))
	assert.True(t, tv.Ignore("root/foo"))
	assert.True(t, tv.Ignore("root_double"))
	assert.False(t, tv.Ignore("newfile"))
	assert.True(t, tv.Ignore(".gitignore"))
	assert.False(t, tv.Ignore("newfile.py"))
	assert.True(t, tv.Ignore("ignoredirectory/"))

	// Never ignore the root directory.
	// This is the only path that may be checked as `.`,
	// and would match the `.*` ignore pattern if specified.
	assert.False(t, tv.Ignore("."))

	// Nested .gitignores should not affect root.
	assert.False(t, tv.Ignore("a.sh"))

	// Nested .gitignores should apply in their path.
	assert.True(t, tv.Ignore("a/a.sh"))
	assert.True(t, tv.Ignore("a/whatever/a.sh"))

	// .git must always be ignored.
	assert.True(t, tv.Ignore(".git"))
}

func TestViewRootInBricksRepo(t *testing.T) {
	v, err := NewViewAtRoot(vfs.MustNew("./testdata"))
	require.NoError(t, err)
	testViewAtRoot(t, testView{t, v})
}

func TestViewRootInTempRepo(t *testing.T) {
	v, err := NewViewAtRoot(vfs.MustNew(createFakeRepo(t, "testdata")))
	require.NoError(t, err)
	testViewAtRoot(t, testView{t, v})
}

func TestViewRootInTempDir(t *testing.T) {
	v, err := NewViewAtRoot(vfs.MustNew(copyTestdata(t, "testdata")))
	require.NoError(t, err)
	testViewAtRoot(t, testView{t, v})
}

func testViewAtA(t *testing.T, tv testView) {
	// Inherit .gitignore from root.
	assert.True(t, tv.Ignore("root.sh"))
	assert.False(t, tv.Ignore("root/foo"))
	assert.True(t, tv.Ignore("root_double"))
	assert.True(t, tv.Ignore("ignoredirectory/"))

	// Check current .gitignore
	assert.True(t, tv.Ignore("a.sh"))
	assert.True(t, tv.Ignore("a_double"))
	assert.False(t, tv.Ignore("newfile"))

	// Nested .gitignores should apply in their path.
	assert.True(t, tv.Ignore("b/b.sh"))
	assert.True(t, tv.Ignore("b/whatever/b.sh"))
}

func TestViewAInBricksRepo(t *testing.T) {
	v, err := NewView(vfs.MustNew("."), vfs.MustNew("./testdata/a"))
	require.NoError(t, err)
	testViewAtA(t, testView{t, v})
}

func TestViewAInTempRepo(t *testing.T) {
	repo := createFakeRepo(t, "testdata")
	v, err := NewView(vfs.MustNew(repo), vfs.MustNew(filepath.Join(repo, "a")))
	require.NoError(t, err)
	testViewAtA(t, testView{t, v})
}

func TestViewAInTempDir(t *testing.T) {
	// Since this is not a fake repo it should not traverse up the tree.
	v, err := NewViewAtRoot(vfs.MustNew(filepath.Join(copyTestdata(t, "testdata"), "a")))
	require.NoError(t, err)
	tv := testView{t, v}

	// Check that this doesn't inherit .gitignore from root.
	assert.False(t, tv.Ignore("root.sh"))
	assert.False(t, tv.Ignore("root/foo"))
	assert.False(t, tv.Ignore("root_double"))

	// Check current .gitignore
	assert.True(t, tv.Ignore("a.sh"))
	assert.True(t, tv.Ignore("a_double"))
	assert.False(t, tv.Ignore("newfile"))

	// Nested .gitignores should apply in their path.
	assert.True(t, tv.Ignore("b/b.sh"))
	assert.True(t, tv.Ignore("b/whatever/b.sh"))
}

func testViewAtAB(t *testing.T, tv testView) {
	// Inherit .gitignore from root.
	assert.True(t, tv.Ignore("root.sh"))
	assert.False(t, tv.Ignore("root/foo"))
	assert.True(t, tv.Ignore("root_double"))
	assert.True(t, tv.Ignore("ignoredirectory/"))

	// Inherit .gitignore from root/a.
	assert.True(t, tv.Ignore("a.sh"))
	assert.True(t, tv.Ignore("a_double"))

	// Check current .gitignore
	assert.True(t, tv.Ignore("b.sh"))
	assert.True(t, tv.Ignore("b_double"))
	assert.False(t, tv.Ignore("newfile"))
}

func TestViewABInBricksRepo(t *testing.T) {
	v, err := NewView(vfs.MustNew("."), vfs.MustNew("./testdata/a/b"))
	require.NoError(t, err)
	testViewAtAB(t, testView{t, v})
}

func TestViewABInTempRepo(t *testing.T) {
	repo := createFakeRepo(t, "testdata")
	v, err := NewView(vfs.MustNew(repo), vfs.MustNew(filepath.Join(repo, "a", "b")))
	require.NoError(t, err)
	testViewAtAB(t, testView{t, v})
}

func TestViewABInTempDir(t *testing.T) {
	// Since this is not a fake repo it should not traverse up the tree.
	v, err := NewViewAtRoot(vfs.MustNew(filepath.Join(copyTestdata(t, "testdata"), "a", "b")))
	tv := testView{t, v}
	require.NoError(t, err)

	// Check that this doesn't inherit .gitignore from root.
	assert.False(t, tv.Ignore("root.sh"))
	assert.False(t, tv.Ignore("root/foo"))
	assert.False(t, tv.Ignore("root_double"))

	// Check that this doesn't inherit .gitignore from root/a.
	assert.False(t, tv.Ignore("a.sh"))
	assert.False(t, tv.Ignore("a_double"))

	// Check current .gitignore
	assert.True(t, tv.Ignore("b.sh"))
	assert.True(t, tv.Ignore("b_double"))
	assert.False(t, tv.Ignore("newfile"))
}

func TestViewAlwaysIgnoresLocalStateDir(t *testing.T) {
	repoPath := createFakeRepo(t, "testdata")

	v, err := NewViewAtRoot(vfs.MustNew(repoPath))
	require.NoError(t, err)

	// assert .databricks is still being ignored
	ign1, err := v.IgnoreDirectory(".databricks")
	require.NoError(t, err)
	assert.True(t, ign1)

	ign2, err := v.IgnoreDirectory("a/.databricks")
	require.NoError(t, err)
	assert.True(t, ign2)
}
