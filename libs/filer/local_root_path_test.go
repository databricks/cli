package filer

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testUnixLocalRootPath(t *testing.T, uncleanRoot string) {
	cleanRoot := filepath.Clean(uncleanRoot)
	rp := NewLocalRootPath(uncleanRoot)

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

func TestUnixLocalRootPath(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.SkipNow()
	}

	testUnixLocalRootPath(t, "/some/root/path")
	testUnixLocalRootPath(t, "/some/root/path/")
	testUnixLocalRootPath(t, "/some/root/path/.")
	testUnixLocalRootPath(t, "/some/root/../path/")
}

func testWindowsLocalRootPath(t *testing.T, uncleanRoot string) {
	cleanRoot := filepath.Clean(uncleanRoot)
	rp := NewLocalRootPath(uncleanRoot)

	remotePath, err := rp.Join(`a\b\c`)
	assert.NoError(t, err)
	assert.Equal(t, cleanRoot+`\a\b\c`, remotePath)

	remotePath, err = rp.Join(`a\b\..\d`)
	assert.NoError(t, err)
	assert.Equal(t, cleanRoot+`\a\d`, remotePath)

	remotePath, err = rp.Join(`a\..\c`)
	assert.NoError(t, err)
	assert.Equal(t, cleanRoot+`\c`, remotePath)

	remotePath, err = rp.Join(`a\b\c\.`)
	assert.NoError(t, err)
	assert.Equal(t, cleanRoot+`\a\b\c`, remotePath)

	remotePath, err = rp.Join(`a\b\..\..`)
	assert.NoError(t, err)
	assert.Equal(t, cleanRoot, remotePath)

	remotePath, err = rp.Join("")
	assert.NoError(t, err)
	assert.Equal(t, cleanRoot, remotePath)

	remotePath, err = rp.Join(".")
	assert.NoError(t, err)
	assert.Equal(t, cleanRoot, remotePath)

	_, err = rp.Join("..")
	assert.ErrorContains(t, err, `relative path escapes root`)

	_, err = rp.Join(`a\..\..`)
	assert.ErrorContains(t, err, `relative path escapes root`)
}

func TestWindowsLocalRootPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.SkipNow()
	}

	testWindowsLocalRootPath(t, `c:\some\root\path`)
	testWindowsLocalRootPath(t, `c:\some\root\path\`)
	testWindowsLocalRootPath(t, `c:\some\root\path\.`)
	testWindowsLocalRootPath(t, `C:\some\root\..\path\`)
}
