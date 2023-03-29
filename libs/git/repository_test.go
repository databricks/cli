package git

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Wrap a Repository and expose a panicking version of [Repository.Ignore].
type testRepository struct {
	t *testing.T
	r *Repository
}

func (r *testRepository) Ignore(relPath string) bool {
	ign, err := r.r.Ignore(relPath)
	require.NoError(r.t, err)
	return ign
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
