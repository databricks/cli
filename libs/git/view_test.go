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

func copyTestdata(t *testing.T, name string) string {
	tempDir := t.TempDir()

	// Copy everything under testdata ${name} to temporary directory.
	err := filepath.WalkDir(name, func(path string, d fs.DirEntry, err error) error {
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

func createFakeRepo(t *testing.T, testdataName string) string {
	absPath := copyTestdata(t, testdataName)

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
	v, err := NewView(createFakeRepo(t, "testdata"))
	require.NoError(t, err)
	testViewAtRoot(t, testView{t, v})
}

func TestViewRootInTempDir(t *testing.T) {
	v, err := NewView(copyTestdata(t, "testdata"))
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
	v, err := NewView(filepath.Join(createFakeRepo(t, "testdata"), "a"))
	require.NoError(t, err)
	testViewAtA(t, testView{t, v})
}

func TestViewAInTempDir(t *testing.T) {
	// Since this is not a fake repo it should not traverse up the tree.
	v, err := NewView(filepath.Join(copyTestdata(t, "testdata"), "a"))
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
	v, err := NewView(filepath.Join(createFakeRepo(t, "testdata"), "a", "b"))
	require.NoError(t, err)
	testViewAtAB(t, testView{t, v})
}

func TestViewABInTempDir(t *testing.T) {
	// Since this is not a fake repo it should not traverse up the tree.
	v, err := NewView(filepath.Join(copyTestdata(t, "testdata"), "a", "b"))
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

func TestViewDoesNotChangeGitignoreIfCacheDirAlreadyIgnoredAtRoot(t *testing.T) {
	expected, err := os.ReadFile("./testdata/.gitignore")
	require.NoError(t, err)

	repoPath := createFakeRepo(t, "testdata")

	// Since root .gitignore already has .databricks, there should be no edits
	// to root .gitignore
	_, err = NewView(repoPath)
	require.NoError(t, err)

	actual, err := os.ReadFile(filepath.Join(repoPath, ".gitignore"))
	require.NoError(t, err)

	assert.Equal(t, string(expected), string(actual))
}

func TestViewDoesNotChangeGitignoreIfCacheDirAlreadyIgnoredInSubdir(t *testing.T) {
	expected, err := os.ReadFile("./testdata/a/.gitignore")
	require.NoError(t, err)

	repoPath := createFakeRepo(t, "testdata")

	// Since root .gitignore already has .databricks, there should be no edits
	// to a/.gitignore
	v, err := NewView(filepath.Join(repoPath, "a"))
	require.NoError(t, err)

	actual, err := os.ReadFile(filepath.Join(repoPath, v.targetPath, ".gitignore"))
	require.NoError(t, err)

	assert.Equal(t, string(expected), string(actual))
}

func TestViewAddsGitignoreWithCacheDir(t *testing.T) {
	repoPath := createFakeRepo(t, "testdata")
	err := os.Remove(filepath.Join(repoPath, ".gitignore"))
	assert.NoError(t, err)

	// Since root .gitignore was deleted, new view adds .databricks to root .gitignore
	v, err := NewView(repoPath)
	require.NoError(t, err)

	err = v.EnsureValidGitIgnoreExists()
	require.NoError(t, err)

	actual, err := os.ReadFile(filepath.Join(repoPath, ".gitignore"))
	require.NoError(t, err)

	assert.Contains(t, string(actual), "\n.databricks\n")
}

func TestViewAddsGitignoreWithCacheDirAtSubdir(t *testing.T) {
	repoPath := createFakeRepo(t, "testdata")
	err := os.Remove(filepath.Join(repoPath, ".gitignore"))
	require.NoError(t, err)

	// Since root .gitignore was deleted, new view adds .databricks to a/.gitignore
	v, err := NewView(filepath.Join(repoPath, "a"))
	require.NoError(t, err)

	err = v.EnsureValidGitIgnoreExists()
	require.NoError(t, err)

	actual, err := os.ReadFile(filepath.Join(repoPath, v.targetPath, ".gitignore"))
	require.NoError(t, err)

	// created .gitignore has cache dir listed
	assert.Contains(t, string(actual), "\n.databricks\n")
	assert.NoFileExists(t, filepath.Join(repoPath, ".gitignore"))
}

func TestViewAlwaysIgnoresCacheDir(t *testing.T) {
	repoPath := createFakeRepo(t, "testdata")

	v, err := NewView(repoPath)
	require.NoError(t, err)

	err = v.EnsureValidGitIgnoreExists()
	require.NoError(t, err)

	// Delete root .gitignore which contains .databricks entry
	err = os.Remove(filepath.Join(repoPath, ".gitignore"))
	require.NoError(t, err)

	// taint rules to reload .gitignore
	v.repo.taintIgnoreRules()

	// assert .databricks is still being ignored
	ign1, err := v.IgnoreDirectory(".databricks")
	require.NoError(t, err)
	assert.True(t, ign1)

	ign2, err := v.IgnoreDirectory("a/.databricks")
	require.NoError(t, err)
	assert.True(t, ign2)
}
