package repofiles

import (
	"os"
	"path/filepath"
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
	assert.ErrorContains(t, err, `contains forbidden pattern ".."`)

	_, err = cleanPath("a/../..")
	assert.ErrorContains(t, err, `contains forbidden pattern ".."`)

	_, err = cleanPath("./../.")
	assert.ErrorContains(t, err, `contains forbidden pattern ".."`)

	_, err = cleanPath("/./.././..")
	assert.ErrorContains(t, err, `file path relative to repo root cannot be empty`)

	_, err = cleanPath("./..")
	assert.ErrorContains(t, err, `contains forbidden pattern ".."`)

	_, err = cleanPath("./../../..")
	assert.ErrorContains(t, err, `contains forbidden pattern ".."`)

	_, err = cleanPath("./../a/./b../../..")
	assert.ErrorContains(t, err, `contains forbidden pattern ".."`)

	_, err = cleanPath(".//a/..//./b/..")
	assert.ErrorContains(t, err, `file path relative to repo root cannot be empty`)
}

func TestRepoFilesRemotePath(t *testing.T) {
	repoFiles := Create("/Repos/doraemon/bar", "/doraemon/foo/bar", nil)

	remotePath, err := repoFiles.remotePath("a/b/c")
	assert.NoError(t, err)
	assert.Equal(t, "/Repos/doraemon/bar/a/b/c", remotePath)

	remotePath, err = repoFiles.remotePath("a/b/../d")
	assert.NoError(t, err)
	assert.Equal(t, "/Repos/doraemon/bar/a/d", remotePath)

	_, err = repoFiles.remotePath("a/b/../..")
	assert.ErrorContains(t, err, "file path relative to repo root cannot be empty")

	_, err = repoFiles.remotePath("")
	assert.ErrorContains(t, err, "file path relative to repo root cannot be empty")

	_, err = repoFiles.remotePath(".")
	assert.ErrorContains(t, err, "file path relative to repo root cannot be empty")

	_, err = repoFiles.remotePath("../..")
	assert.ErrorContains(t, err, `contains forbidden pattern ".."`)
}

func TestRepoFilesLocalPath(t *testing.T) {
	repoFiles := Create("/Repos/doraemon/bar", "/doraemon/foo/bar", nil)

	remotePath, err := repoFiles.localPath("a/b/c")
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join("/doraemon/foo/bar/a/b", "c"), remotePath)

	remotePath, err = repoFiles.localPath("a/b/../d")
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join("/doraemon/foo/bar/a", "d"), remotePath)

	_, err = repoFiles.localPath("a/b/../..")
	assert.ErrorContains(t, err, "file path relative to repo root cannot be empty")

	_, err = repoFiles.localPath("")
	assert.ErrorContains(t, err, "file path relative to repo root cannot be empty")

	_, err = repoFiles.localPath(".")
	assert.ErrorContains(t, err, "file path relative to repo root cannot be empty")

	_, err = repoFiles.localPath("../..")
	assert.ErrorContains(t, err, `contains forbidden pattern ".."`)
}

func TestRepoReadLocal(t *testing.T) {
	tempDir := t.TempDir()
	helloPath := filepath.Join(tempDir, "hello.txt")
	err := os.WriteFile(helloPath, []byte("my name is doraemon :P"), os.ModePerm)
	assert.NoError(t, err)

	repoFiles := Create("/Repos/doraemon/bar", tempDir, nil)
	bytes, err := repoFiles.readLocal("./a/../hello.txt")
	assert.NoError(t, err)
	assert.Equal(t, "my name is doraemon :P", string(bytes))
}

// TODO: setup mocking of api calls for bricks with gomock
