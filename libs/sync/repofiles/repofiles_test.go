package repofiles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepoFilesRemotePath(t *testing.T) {
	repoRoot := "/Repos/doraemon/bar"
	repoFiles, err := Create(repoRoot, "/doraemon/foo/bar", nil, nil)
	require.NoError(t, err)

	remotePath, err := repoFiles.RemotePath("a/b/c")
	assert.NoError(t, err)
	assert.Equal(t, repoRoot+"/a/b/c", remotePath)

	remotePath, err = repoFiles.RemotePath("a/b/../d")
	assert.NoError(t, err)
	assert.Equal(t, repoRoot+"/a/d", remotePath)

	remotePath, err = repoFiles.RemotePath("a/../c")
	assert.NoError(t, err)
	assert.Equal(t, repoRoot+"/c", remotePath)

	remotePath, err = repoFiles.RemotePath("a/b/c/.")
	assert.NoError(t, err)
	assert.Equal(t, repoRoot+"/a/b/c", remotePath)

	remotePath, err = repoFiles.RemotePath("a/b/c/d/./../../f/g")
	assert.NoError(t, err)
	assert.Equal(t, repoRoot+"/a/b/f/g", remotePath)

	_, err = repoFiles.RemotePath("..")
	assert.ErrorContains(t, err, `relative file path is not inside repo root: ..`)

	_, err = repoFiles.RemotePath("a/../..")
	assert.ErrorContains(t, err, `relative file path is not inside repo root: a/../..`)

	_, err = repoFiles.RemotePath("./../.")
	assert.ErrorContains(t, err, `relative file path is not inside repo root: ./../.`)

	_, err = repoFiles.RemotePath("/./.././..")
	assert.ErrorContains(t, err, `relative file path is not inside repo root: /./.././..`)

	_, err = repoFiles.RemotePath("./../.")
	assert.ErrorContains(t, err, `relative file path is not inside repo root: ./../.`)

	_, err = repoFiles.RemotePath("./..")
	assert.ErrorContains(t, err, `relative file path is not inside repo root: ./..`)

	_, err = repoFiles.RemotePath("./../../..")
	assert.ErrorContains(t, err, `relative file path is not inside repo root: ./../../..`)

	_, err = repoFiles.RemotePath("./../a/./b../../..")
	assert.ErrorContains(t, err, `relative file path is not inside repo root: ./../a/./b../../..`)

	_, err = repoFiles.RemotePath("../..")
	assert.ErrorContains(t, err, `relative file path is not inside repo root: ../..`)

	_, err = repoFiles.RemotePath(".//a/..//./b/..")
	assert.ErrorContains(t, err, `file path relative to repo root cannot be empty`)

	_, err = repoFiles.RemotePath("a/b/../..")
	assert.ErrorContains(t, err, "file path relative to repo root cannot be empty")

	_, err = repoFiles.RemotePath("")
	assert.ErrorContains(t, err, "file path relative to repo root cannot be empty")

	_, err = repoFiles.RemotePath(".")
	assert.ErrorContains(t, err, "file path relative to repo root cannot be empty")

	_, err = repoFiles.RemotePath("/")
	assert.ErrorContains(t, err, "file path relative to repo root cannot be empty")
}

func TestRepoReadLocal(t *testing.T) {
	tempDir := t.TempDir()
	helloPath := filepath.Join(tempDir, "hello.txt")
	err := os.WriteFile(helloPath, []byte("my name is doraemon :P"), os.ModePerm)
	assert.NoError(t, err)

	repoFiles, err := Create("/Repos/doraemon/bar", tempDir, nil, nil)
	require.NoError(t, err)
	bytes, err := repoFiles.readLocal("./a/../hello.txt")
	assert.NoError(t, err)
	assert.Equal(t, "my name is doraemon :P", string(bytes))
}
