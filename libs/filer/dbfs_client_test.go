package filer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveDbfsPath(t *testing.T) {
	path, err := ResolveDbfsPath("dbfs:/")
	assert.NoError(t, err)
	assert.Equal(t, "/", path)

	path, err = ResolveDbfsPath("dbfs:/abc")
	assert.NoError(t, err)
	assert.Equal(t, "/abc", path)

	path, err = ResolveDbfsPath("dbfs:/a/b/c")
	assert.NoError(t, err)
	assert.Equal(t, "/a/b/c", path)

	path, err = ResolveDbfsPath("dbfs:/a/b/.")
	assert.NoError(t, err)
	assert.Equal(t, "/a/b/.", path)

	path, err = ResolveDbfsPath("dbfs:/a/../c")
	assert.NoError(t, err)
	assert.Equal(t, "/a/../c", path)

	_, err = ResolveDbfsPath("dbf:/a/b/c")
	assert.ErrorContains(t, err, "expected dbfs path (with the dbfs:/ prefix): dbf:/a/b/c")

	_, err = ResolveDbfsPath("/a/b/c")
	assert.ErrorContains(t, err, "expected dbfs path (with the dbfs:/ prefix): /a/b/c")

	_, err = ResolveDbfsPath("dbfs:a/b/c")
	assert.ErrorContains(t, err, "expected dbfs path (with the dbfs:/ prefix): dbfs:a/b/c")
}
