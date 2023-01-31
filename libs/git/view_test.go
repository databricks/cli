package git

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func copyTestdata(t *testing.T) string {
	tempDir := t.TempDir()

	// Copy everything under "testdata" to temporary directory.
	err := filepath.WalkDir("testdata", func(path string, d fs.DirEntry, err error) error {
		require.NoError(t, err)

		if d.IsDir() {
			err := os.MkdirAll(filepath.Join(tempDir, path), 0755)
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
	return filepath.Join(tempDir, "testdata")
}

func createFakeRepo(t *testing.T) string {
	absPath := copyTestdata(t)

	// Add .git directory to make it look like a Git repository.
	err := os.Mkdir(filepath.Join(absPath, ".git"), 0755)
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
	assert.True(t, tv.Ignore("ignoredirectory/"))

	// Nested .gitignores should not affect root.
	assert.False(t, tv.Ignore("a.sh"))

	// Nested .gitignores should apply in their path.
	assert.True(t, tv.Ignore("a/a.sh"))
	assert.True(t, tv.Ignore("a/whatever/a.sh"))

	// .git must always be ignored.
	assert.True(t, tv.Ignore(".git"))
}

func TestViewRootInBricksRepo(t *testing.T) {
	v, err := NewView("./testdata")
	require.NoError(t, err)
	testViewAtRoot(t, testView{t, v})
}

func TestViewRootInTempRepo(t *testing.T) {
	v, err := NewView(createFakeRepo(t))
	require.NoError(t, err)
	testViewAtRoot(t, testView{t, v})
}

func TestViewRootInTempDir(t *testing.T) {
	v, err := NewView(copyTestdata(t))
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
	v, err := NewView("./testdata/a")
	require.NoError(t, err)
	testViewAtA(t, testView{t, v})
}

func TestViewAInTempRepo(t *testing.T) {
	v, err := NewView(filepath.Join(createFakeRepo(t), "a"))
	require.NoError(t, err)
	testViewAtA(t, testView{t, v})
}

func TestViewAInTempDir(t *testing.T) {
	// Since this is not a fake repo it should not traverse up the tree.
	v, err := NewView(filepath.Join(copyTestdata(t), "a"))
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
	v, err := NewView("./testdata/a/b")
	require.NoError(t, err)
	testViewAtAB(t, testView{t, v})
}

func TestViewABInTempRepo(t *testing.T) {
	v, err := NewView(filepath.Join(createFakeRepo(t), "a", "b"))
	require.NoError(t, err)
	testViewAtAB(t, testView{t, v})
}

func TestViewABInTempDir(t *testing.T) {
	// Since this is not a fake repo it should not traverse up the tree.
	v, err := NewView(filepath.Join(copyTestdata(t), "a", "b"))
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
