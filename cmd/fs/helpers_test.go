package fs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTrimDbfsScheme(t *testing.T) {
	path, err := trimDbfsScheme("dbfs:/")
	assert.NoError(t, err)
	assert.Equal(t, "/", path)

	path, err = trimDbfsScheme("dbfs:/abc")
	assert.NoError(t, err)
	assert.Equal(t, "/abc", path)

	path, err = trimDbfsScheme("dbfs:/a/b/c")
	assert.NoError(t, err)
	assert.Equal(t, "/a/b/c", path)

	path, err = trimDbfsScheme("dbfs:/a/b/.")
	assert.NoError(t, err)
	assert.Equal(t, "/a/b/.", path)

	path, err = trimDbfsScheme("dbfs:/a/../c")
	assert.NoError(t, err)
	assert.Equal(t, "/a/../c", path)

	_, err = trimDbfsScheme("dbf:/a/b/c")
	assert.ErrorContains(t, err, "expected dbfs path (with the dbfs:/ prefix): dbf:/a/b/c")

	_, err = trimDbfsScheme("/a/b/c")
	assert.ErrorContains(t, err, "expected dbfs path (with the dbfs:/ prefix): /a/b/c")

	// check we error on relative paths
	_, err = trimDbfsScheme("dbfs:a/b/c")
	assert.ErrorContains(t, err, "expected dbfs path (with the dbfs:/ prefix): dbfs:a/b/c")
}

func TestTrimLocalScheme(t *testing.T) {
	path, err := trimLocalScheme("file:/")
	assert.NoError(t, err)
	assert.Equal(t, "/", path)

	path, err = trimLocalScheme("file:/abc")
	assert.NoError(t, err)
	assert.Equal(t, "/abc", path)

	path, err = trimLocalScheme("file:/a/b/c")
	assert.NoError(t, err)
	assert.Equal(t, "/a/b/c", path)

	path, err = trimLocalScheme("file:/a/b/.")
	assert.NoError(t, err)
	assert.Equal(t, "/a/b/.", path)

	path, err = trimLocalScheme("file:/a/../c")
	assert.NoError(t, err)
	assert.Equal(t, "/a/../c", path)

	// check relative path are also supported
	path, err = trimLocalScheme("file:a/b/c")
	assert.NoError(t, err)
	assert.Equal(t, "a/b/c", path)

	path, err = trimLocalScheme("file:./a/b")
	assert.NoError(t, err)
	assert.Equal(t, "./a/b", path)

	_, err = trimLocalScheme("fil:/a/b/c")
	assert.ErrorContains(t, err, "expected file path (with the file: prefix): fil:/a/b/c")

	_, err = trimLocalScheme("/a/b/c")
	assert.ErrorContains(t, err, "expected file path (with the file: prefix): /a/b/c")

}
