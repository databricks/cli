package fs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveDbfsPath(t *testing.T) {
	path, err := resolveDbfsPath("dbfs:/")
	assert.NoError(t, err)
	assert.Equal(t, "/", path)

	path, err = resolveDbfsPath("dbfs:/abc")
	assert.NoError(t, err)
	assert.Equal(t, "/abc", path)

	path, err = resolveDbfsPath("dbfs:/a/b/c")
	assert.NoError(t, err)
	assert.Equal(t, "/a/b/c", path)

	path, err = resolveDbfsPath("dbfs:/a/b/.")
	assert.NoError(t, err)
	assert.Equal(t, "/a/b/.", path)

	path, err = resolveDbfsPath("dbfs:/a/../c")
	assert.NoError(t, err)
	assert.Equal(t, "/a/../c", path)

	_, err = resolveDbfsPath("dbf:/a/b/c")
	assert.ErrorContains(t, err, "expected dbfs path (with the dbfs:/ prefix): dbf:/a/b/c")

	_, err = resolveDbfsPath("/a/b/c")
	assert.ErrorContains(t, err, "expected dbfs path (with the dbfs:/ prefix): /a/b/c")

	_, err = resolveDbfsPath("dbfs:a/b/c")
	assert.ErrorContains(t, err, "expected dbfs path (with the dbfs:/ prefix): dbfs:a/b/c")
}
