package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Wrap a Repository and expose a panicking version of [Repository.Ignore].
type testRepository struct {
	t *testing.T
	r *Repository
}

func newTestRepository(t *testing.T) *testRepository {
	tmp := t.TempDir()
	err := os.Mkdir(filepath.Join(tmp, ".git"), os.ModePerm)
	require.NoError(t, err)

	f1, err := os.Create(filepath.Join(tmp, ".git", "config"))
	require.NoError(t, err)
	defer f1.Close()

	f1.WriteString(
		`[core]
	repositoryformatversion = 0
	filemode = true
	bare = false
	logallrefupdates = true
	ignorecase = true
	precomposeunicode = true
`)

	f2, err := os.Create(filepath.Join(tmp, ".git", "HEAD"))
	require.NoError(t, err)
	defer f2.Close()

	_, err = f2.WriteString(`ref: refs/heads/main`)
	require.NoError(t, err)

	repo, err := NewRepository(tmp)
	require.NoError(t, err)

	return &testRepository{
		t: t,
		r: repo,
	}
}

func (testRepo *testRepository) checkoutCommit(commitId string) {
	f, err := os.OpenFile(filepath.Join(testRepo.r.rootPath, ".git", "HEAD"), os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	require.NoError(testRepo.t, err)
	defer f.Close()

	_, err = f.WriteString(commitId)
	require.NoError(testRepo.t, err)
}

func (testRepo *testRepository) addBranch(name string, latestCommit string) {
	// create dir for branch head reference
	branchDir := filepath.Join(testRepo.r.rootPath, ".git", "refs", "heads")
	err := os.MkdirAll(branchDir, os.ModePerm)
	require.NoError(testRepo.t, err)

	// create branch head reference file
	f, err := os.OpenFile(filepath.Join(branchDir, name), os.O_CREATE|os.O_WRONLY, os.ModePerm)
	require.NoError(testRepo.t, err)
	defer f.Close()

	// enter the latest commit in the branch reference file
	_, err = f.WriteString(latestCommit)
	require.NoError(testRepo.t, err)
}

func (testRepo *testRepository) checkoutBranch(name string) {
	f, err := os.OpenFile(filepath.Join(testRepo.r.rootPath, ".git", "HEAD"), os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	require.NoError(testRepo.t, err)
	defer f.Close()

	_, err = f.WriteString("ref: refs/heads/" + name)
	require.NoError(testRepo.t, err)
}

// add remote origin url to test repo
func (testRepo *testRepository) addOriginUrl(url string) {
	// open config in append mode
	f, err := os.OpenFile(filepath.Join(testRepo.r.rootPath, ".git", "config"), os.O_WRONLY|os.O_APPEND, os.ModePerm)
	require.NoError(testRepo.t, err)
	defer f.Close()

	_, err = f.WriteString(
		fmt.Sprintf(`[remote "origin"]
	url = %s`, url))
	require.NoError(testRepo.t, err)

	// reload config to reflect the remote url
	err = testRepo.r.loadConfig()
	require.NoError(testRepo.t, err)
}

func (testRepo *testRepository) Ignore(relPath string) bool {
	ign, err := testRepo.r.Ignore(relPath)
	require.NoError(testRepo.t, err)
	return ign
}

func (testRepo *testRepository) assertBranch(expected string) {
	branch, err := testRepo.r.CurrentBranch()
	assert.NoError(testRepo.t, err)
	assert.Equal(testRepo.t, expected, branch)
}

func (testRepo *testRepository) assertCommit(expected string) {
	commit, err := testRepo.r.LatestCommit()
	assert.NoError(testRepo.t, err)
	assert.Equal(testRepo.t, expected, commit)
}

func (testRepo *testRepository) assertOriginUrl(expected string) {
	originUrl, err := testRepo.r.OriginUrl()
	assert.NoError(testRepo.t, err)
	assert.Equal(testRepo.t, expected, originUrl)
}

func TestRepository(t *testing.T) {
	// Load this repository as test.
	repo, err := NewRepository("../..")
	tr := testRepository{t, repo}
	require.NoError(t, err)

	// Check that the root path is real.
	assert.True(t, filepath.IsAbs(repo.Root()))

	// Check that top level ignores work.
	assert.True(t, tr.Ignore(".DS_Store"))
	assert.True(t, tr.Ignore("foo.pyc"))
	assert.False(t, tr.Ignore("vendor"))
	assert.True(t, tr.Ignore("vendor/"))

	// Check that ignores under testdata work.
	assert.True(t, tr.Ignore(filepath.Join("libs", "git", "testdata", "root.ignoreme")))
}

func TestRepositoryGitConfigForEmptyRepo(t *testing.T) {
	repo := newTestRepository(t)
	repo.assertBranch("main")
	repo.assertCommit("")
	repo.assertOriginUrl("")
}

func TestRepositoryGitConfig(t *testing.T) {
	repo := newTestRepository(t)
	repo.addBranch("foo", strings.Repeat("1", 40))
	repo.addBranch("bar", strings.Repeat("2", 40))
	repo.assertBranch("main")
	repo.assertCommit("")
	repo.assertOriginUrl("")

	repo.checkoutBranch("foo")
	repo.assertBranch("foo")
	repo.assertCommit(strings.Repeat("1", 40))
	repo.assertOriginUrl("")

	repo.addOriginUrl("https://www.foo.com/bar")
	repo.assertBranch("foo")
	repo.assertCommit(strings.Repeat("1", 40))
	repo.assertOriginUrl("https://www.foo.com/bar")

	repo.checkoutBranch("bar")
	repo.assertBranch("bar")
	repo.assertCommit(strings.Repeat("2", 40))
	repo.assertOriginUrl("https://www.foo.com/bar")

	repo.checkoutCommit(strings.Repeat("3", 40))
	repo.assertBranch("")
	repo.assertCommit(strings.Repeat("3", 40))
	repo.assertOriginUrl("https://www.foo.com/bar")
}

func TestRepositoryGitConfigForSshUrl(t *testing.T) {
	repo := newTestRepository(t)
	repo.addOriginUrl(`git@foo.com:databricks/bar.git`)

	repo.assertBranch("main")
	repo.assertCommit("")
	repo.assertOriginUrl("git@foo.com:databricks/bar.git")
}

func TestRepositoryGitConfigWhenNotARepo(t *testing.T) {
	tmp := t.TempDir()
	repo, err := NewRepository(tmp)
	require.NoError(t, err)

	branch, err := repo.CurrentBranch()
	assert.NoError(t, err)
	assert.Equal(t, "", branch)

	commit, err := repo.LatestCommit()
	assert.NoError(t, err)
	assert.Equal(t, "", commit)

	originUrl, err := repo.OriginUrl()
	assert.NoError(t, err)
	assert.Equal(t, "", originUrl)
}
