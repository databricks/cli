package filer

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testRootPath(t *testing.T, uncleanRoot string) {
	cleanRoot := path.Clean(uncleanRoot)
	rp := NewWorkspaceRootPath(uncleanRoot)

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

	// All roots in this test end in the path element "path".
	remotePath, err = rp.Join("../path/x")
	assert.NoError(t, err)
	assert.Equal(t, cleanRoot+"/x", remotePath)

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

	// Siblings that share the root as a string prefix must be rejected.
	_, err = rp.Join("../path-evil")
	assert.ErrorContains(t, err, `relative path escapes root: ../path-evil`)

	_, err = rp.Join("../path-evil/x")
	assert.ErrorContains(t, err, `relative path escapes root: ../path-evil/x`)

	_, err = rp.Join("../pathx")
	assert.ErrorContains(t, err, `relative path escapes root: ../pathx`)
}

func TestRootPathClean(t *testing.T) {
	testRootPath(t, "/some/root/path")
}

func TestRootPathUnclean(t *testing.T) {
	testRootPath(t, "/some/root/path/")
	testRootPath(t, "/some/root/path/.")
	testRootPath(t, "/some/root/../path/")
}
