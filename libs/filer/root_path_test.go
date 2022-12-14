package filer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRootPath(t *testing.T) {
	root := RootPath("/some/root/path")

	remotePath, err := root.Join("a/b/c")
	assert.NoError(t, err)
	assert.Equal(t, root+"/a/b/c", remotePath)

	remotePath, err = root.Join("a/b/../d")
	assert.NoError(t, err)
	assert.Equal(t, root+"/a/d", remotePath)

	remotePath, err = root.Join("a/../c")
	assert.NoError(t, err)
	assert.Equal(t, root+"/c", remotePath)

	remotePath, err = root.Join("a/b/c/.")
	assert.NoError(t, err)
	assert.Equal(t, root+"/a/b/c", remotePath)

	remotePath, err = root.Join("a/b/c/d/./../../f/g")
	assert.NoError(t, err)
	assert.Equal(t, root+"/a/b/f/g", remotePath)

	_, err = root.Join("..")
	assert.ErrorContains(t, err, `relative path escapes root: ..`)

	_, err = root.Join("a/../..")
	assert.ErrorContains(t, err, `relative path escapes root: a/../..`)

	_, err = root.Join("./../.")
	assert.ErrorContains(t, err, `relative path escapes root: ./../.`)

	_, err = root.Join("/./.././..")
	assert.ErrorContains(t, err, `relative path escapes root: /./.././..`)

	_, err = root.Join("./../.")
	assert.ErrorContains(t, err, `relative path escapes root: ./../.`)

	_, err = root.Join("./..")
	assert.ErrorContains(t, err, `relative path escapes root: ./..`)

	_, err = root.Join("./../../..")
	assert.ErrorContains(t, err, `relative path escapes root: ./../../..`)

	_, err = root.Join("./../a/./b../../..")
	assert.ErrorContains(t, err, `relative path escapes root: ./../a/./b../../..`)

	_, err = root.Join("../..")
	assert.ErrorContains(t, err, `relative path escapes root: ../..`)

	_, err = root.Join(".//a/..//./b/..")
	assert.ErrorContains(t, err, `relative path resolves to root: .//a/..//./b/..`)

	_, err = root.Join("a/b/../..")
	assert.ErrorContains(t, err, "relative path resolves to root: a/b/../..")

	_, err = root.Join("")
	assert.ErrorContains(t, err, "relative path resolves to root: ")

	_, err = root.Join(".")
	assert.ErrorContains(t, err, "relative path resolves to root: .")

	_, err = root.Join("/")
	assert.ErrorContains(t, err, "relative path resolves to root: /")
}
