package gitignore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestView(t *testing.T) {
	v, err := NewView("./testdata")
	require.NoError(t, err)

	// Check .gitignore at root.
	assert.True(t, v.Ignore("root.sh"))
	assert.True(t, v.Ignore("root/foo"))
	assert.True(t, v.Ignore("root_double"))
	assert.False(t, v.Ignore("newfile"))

	// Nested .gitignores should not affect root.
	assert.False(t, v.Ignore("a.sh"))

	// Nested .gitignores should apply in their path.
	assert.True(t, v.Ignore("a/a.sh"))
	assert.True(t, v.Ignore("a/whatever/a.sh"))
}

func TestViewA(t *testing.T) {
	v, err := NewView("./testdata/a")
	require.NoError(t, err)

	// Inherit .gitignore from root.
	assert.True(t, v.Ignore("root.sh"))
	assert.False(t, v.Ignore("root/foo"))
	assert.True(t, v.Ignore("root_double"))

	// Check current .gitignore
	assert.True(t, v.Ignore("a.sh"))
	assert.True(t, v.Ignore("a_double"))
	assert.False(t, v.Ignore("newfile"))

	// Nested .gitignores should apply in their path.
	assert.True(t, v.Ignore("b/b.sh"))
	assert.True(t, v.Ignore("b/whatever/b.sh"))
}

func TestViewAB(t *testing.T) {
	v, err := NewView("./testdata/a/b")
	require.NoError(t, err)

	// Inherit .gitignore from root.
	assert.True(t, v.Ignore("root.sh"))
	assert.False(t, v.Ignore("root/foo"))
	assert.True(t, v.Ignore("root_double"))

	// Inherit .gitignore from root/a.
	assert.True(t, v.Ignore("a.sh"))
	assert.True(t, v.Ignore("a_double"))

	// Check current .gitignore
	assert.True(t, v.Ignore("b.sh"))
	assert.True(t, v.Ignore("b_double"))
	assert.False(t, v.Ignore("newfile"))

	// Nested .gitignores should apply in their path.
	assert.True(t, v.Ignore("c/c.sh"))
	assert.True(t, v.Ignore("c/whatever/c.sh"))
}

func TestViewABC(t *testing.T) {
	v, err := NewView("./testdata/a/b/c")
	require.NoError(t, err)

	// Inherit .gitignore from root.
	assert.True(t, v.Ignore("root.sh"))
	assert.False(t, v.Ignore("root/foo"))
	assert.True(t, v.Ignore("root_double"))

	// Inherit .gitignore from root/a.
	assert.True(t, v.Ignore("a.sh"))
	assert.True(t, v.Ignore("a_double"))

	// Inherit .gitignore from root/b.
	assert.True(t, v.Ignore("b.sh"))
	assert.True(t, v.Ignore("b_double"))

	// Check current .gitignore
	assert.True(t, v.Ignore("c.sh"))
	assert.True(t, v.Ignore("c_double"))
	assert.False(t, v.Ignore("newfile"))
}
