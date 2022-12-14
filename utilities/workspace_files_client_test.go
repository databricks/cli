package utilities

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkspaceFilesClientPaths(t *testing.T) {
	root := "/some/root/path"
	f := WorkspaceFilesClient{root: root}

	remotePath, err := f.absPath("a/b/c")
	assert.NoError(t, err)
	assert.Equal(t, root+"/a/b/c", remotePath)

	remotePath, err = f.absPath("a/b/../d")
	assert.NoError(t, err)
	assert.Equal(t, root+"/a/d", remotePath)

	remotePath, err = f.absPath("a/../c")
	assert.NoError(t, err)
	assert.Equal(t, root+"/c", remotePath)

	remotePath, err = f.absPath("a/b/c/.")
	assert.NoError(t, err)
	assert.Equal(t, root+"/a/b/c", remotePath)

	remotePath, err = f.absPath("a/b/c/d/./../../f/g")
	assert.NoError(t, err)
	assert.Equal(t, root+"/a/b/f/g", remotePath)

	_, err = f.absPath("..")
	assert.ErrorContains(t, err, `relative path escapes root: ..`)

	_, err = f.absPath("a/../..")
	assert.ErrorContains(t, err, `relative path escapes root: a/../..`)

	_, err = f.absPath("./../.")
	assert.ErrorContains(t, err, `relative path escapes root: ./../.`)

	_, err = f.absPath("/./.././..")
	assert.ErrorContains(t, err, `relative path escapes root: /./.././..`)

	_, err = f.absPath("./../.")
	assert.ErrorContains(t, err, `relative path escapes root: ./../.`)

	_, err = f.absPath("./..")
	assert.ErrorContains(t, err, `relative path escapes root: ./..`)

	_, err = f.absPath("./../../..")
	assert.ErrorContains(t, err, `relative path escapes root: ./../../..`)

	_, err = f.absPath("./../a/./b../../..")
	assert.ErrorContains(t, err, `relative path escapes root: ./../a/./b../../..`)

	_, err = f.absPath("../..")
	assert.ErrorContains(t, err, `relative path escapes root: ../..`)

	_, err = f.absPath(".//a/..//./b/..")
	assert.ErrorContains(t, err, `relative path resolves to root: .//a/..//./b/..`)

	_, err = f.absPath("a/b/../..")
	assert.ErrorContains(t, err, "relative path resolves to root: a/b/../..")

	_, err = f.absPath("")
	assert.ErrorContains(t, err, "relative path resolves to root: ")

	_, err = f.absPath(".")
	assert.ErrorContains(t, err, "relative path resolves to root: .")

	_, err = f.absPath("/")
	assert.ErrorContains(t, err, "relative path resolves to root: /")
}
