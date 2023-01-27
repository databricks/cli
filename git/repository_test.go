package git

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository(t *testing.T) {
	// Load this repository as test.
	repo, err := NewRepository("..")
	require.NoError(t, err)

	// Load all .gitignore files in this repository.
	err = repo.includeIgnoreFilesForPath(".")
	require.NoError(t, err)

	// Check that top level ignores work.
	assert.True(t, repo.Ignore(".DS_Store"))
	assert.True(t, repo.Ignore("foo.pyc"))
	assert.False(t, repo.Ignore("vendor"))
	assert.True(t, repo.Ignore("vendor/"))

	// Check that ignores under testdata work.
	assert.True(t, repo.Ignore(filepath.Join("git", "testdata", "root.ignoreme")))
}
