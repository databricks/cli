package repofiles

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanPath(t *testing.T) {
	val, err := cleanPath("a/b/c")
	assert.NoError(t, err)
	assert.Equal(t, "a/b/c", val)

	val, err = cleanPath("a/../c")
	assert.NoError(t, err)
	assert.Equal(t, "c", val)

	val, err = cleanPath("a/b/c/.")
	assert.NoError(t, err)
	assert.Equal(t, "a/b/c", val)

	val, err = cleanPath("a/b/c/d/./../../f/g")
	assert.NoError(t, err)
	assert.Equal(t, "a/b/f/g", val)

	_, err = cleanPath("..")
	assert.Error(t, err, `contains forbidden pattern ".."`)

	_, err = cleanPath("a/../..")
	assert.Error(t, err, `contains forbidden pattern ".."`)

	_, err = cleanPath("./../.")
	assert.Error(t, err, `contains forbidden pattern ".."`)

	_, err = cleanPath("/./.././..")
	assert.Error(t, err, `file path relative to repo root cannot be empty`)

	_, err = cleanPath("./..")
	assert.Error(t, err, `file path relative to repo root cannot be empty`)

	_, err = cleanPath("./../../..")
	assert.Error(t, err, `file path relative to repo root cannot be empty`)

	_, err = cleanPath("./../a/./b../../..")
	assert.Error(t, err, `file path relative to repo root cannot be empty`)
}
