package filer

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testUnixRootPath(t *testing.T, uncleanRoot string) {
	cleanRoot := path.Clean(uncleanRoot)
	rp := NewUnixRootPath(uncleanRoot)

	remotePath, err := rp.Join("a/b/c")
	assert.NoError(t, err)
	assert.Equal(t, cleanRoot+"/a/b/c", remotePath)

	remotePath, err = rp.Join("a/b/../d")
	assert.NoError(t, err)
	assert.Equal(t, cleanRoot+"/a/d", remotePath)

	remotePath, err = rp.Join("a/../c")
	assert.NoError(t, err)
	assert.Equal(t, cleanRoot+"/c", remotePath)

	remotePath, err = rp.Join("a/b/c/.")
	assert.NoError(t, err)
	assert.Equal(t, cleanRoot+"/a/b/c", remotePath)

	remotePath, err = rp.Join("a/b/c/d/./../../f/g")
	assert.NoError(t, err)
	assert.Equal(t, cleanRoot+"/a/b/f/g", remotePath)

	remotePath, err = rp.Join(".//a/..//./b/..")
	assert.NoError(t, err)
	assert.Equal(t, cleanRoot, remotePath)

	remotePath, err = rp.Join("a/b/../..")
	assert.NoError(t, err)
	assert.Equal(t, cleanRoot, remotePath)

	remotePath, err = rp.Join("")
	assert.NoError(t, err)
	assert.Equal(t, cleanRoot, remotePath)

	remotePath, err = rp.Join(".")
	assert.NoError(t, err)
	assert.Equal(t, cleanRoot, remotePath)

	remotePath, err = rp.Join("/")
	assert.NoError(t, err)
	assert.Equal(t, cleanRoot, remotePath)

	_, err = rp.Join("..")
	assert.ErrorContains(t, err, `relative path escapes root: ..`)

	_, err = rp.Join("a/../..")
	assert.ErrorContains(t, err, `relative path escapes root: a/../..`)

	_, err = rp.Join("./../.")
	assert.ErrorContains(t, err, `relative path escapes root: ./../.`)

	_, err = rp.Join("/./.././..")
	assert.ErrorContains(t, err, `relative path escapes root: /./.././..`)

	_, err = rp.Join("./../.")
	assert.ErrorContains(t, err, `relative path escapes root: ./../.`)

	_, err = rp.Join("./..")
	assert.ErrorContains(t, err, `relative path escapes root: ./..`)

	_, err = rp.Join("./../../..")
	assert.ErrorContains(t, err, `relative path escapes root: ./../../..`)

	_, err = rp.Join("./../a/./b../../..")
	assert.ErrorContains(t, err, `relative path escapes root: ./../a/./b../../..`)

	_, err = rp.Join("../..")
	assert.ErrorContains(t, err, `relative path escapes root: ../..`)
}

func TestUnixRootPathClean(t *testing.T) {
	testUnixRootPath(t, "/some/root/path")
}

func TestUnixRootPathUnclean(t *testing.T) {
	testUnixRootPath(t, "/some/root/path/")
	testUnixRootPath(t, "/some/root/path/.")
	testUnixRootPath(t, "/some/root/../path/")
}
