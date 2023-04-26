package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeadReferencePathForObjectID(t *testing.T) {
	head := &Reference{
		Type:    ReferenceTypeSHA1,
		Content: strings.Repeat("a", 40),
	}
	_, err := head.ReferencePath()
	assert.ErrorIs(t, err, ErrNotAReference)
}

func TestHeadCurrentBranchForObjectID(t *testing.T) {
	head := &Reference{
		Type:    ReferenceTypeSHA1,
		Content: strings.Repeat("a", 40),
	}
	_, err := head.CurrentBranch()
	assert.ErrorIs(t, err, ErrNotABranch)
}

func TestHeadCurrentBranchForReference(t *testing.T) {
	head := &Reference{
		Type:    ReferenceTypePointer,
		Content: `ref: refs/heads/my-branch`,
	}
	branch, err := head.CurrentBranch()
	assert.NoError(t, err)
	assert.Equal(t, "my-branch", branch)
}

func TestHeadReferencePathForReference(t *testing.T) {
	head := &Reference{
		Type:    ReferenceTypePointer,
		Content: `ref: refs/heads/my-branch`,
	}
	path, err := head.ReferencePath()
	assert.NoError(t, err)
	assert.Equal(t, filepath.FromSlash("refs/heads/my-branch"), path)
}

func TestHeadLoadingForObjectID(t *testing.T) {
	tmp := t.TempDir()
	f, err := os.Create(filepath.Join(tmp, "HEAD"))
	require.NoError(t, err)
	defer f.Close()
	f.WriteString(strings.Repeat("e", 40) + "\r\n")

	head, err := LoadReferenceFile(filepath.Join(tmp, "HEAD"))
	assert.NoError(t, err)
	assert.Equal(t, ReferenceTypeSHA1, head.Type)
	assert.Equal(t, strings.Repeat("e", 40), head.Content)
}

func TestHeadLoadingForReference(t *testing.T) {
	tmp := t.TempDir()
	f, err := os.OpenFile(filepath.Join(tmp, "HEAD"), os.O_CREATE|os.O_WRONLY, os.ModePerm)
	require.NoError(t, err)
	defer f.Close()
	f.WriteString("ref: refs/heads/foo\n")

	head, err := LoadReferenceFile(filepath.Join(tmp, "HEAD"))
	assert.NoError(t, err)
	assert.Equal(t, ReferenceTypePointer, head.Type)
	assert.Equal(t, "ref: refs/heads/foo", head.Content)
}

func TestHeadLoadingFailsForInvalidContent(t *testing.T) {
	tmp := t.TempDir()
	f, err := os.OpenFile(filepath.Join(tmp, "HEAD"), os.O_CREATE|os.O_WRONLY, os.ModePerm)
	require.NoError(t, err)
	defer f.Close()
	f.WriteString("abc")

	_, err = LoadReferenceFile(filepath.Join(tmp, "HEAD"))
	assert.ErrorContains(t, err, "unknown format for git HEAD")
}

func TestIsSha1(t *testing.T) {
	a := strings.Repeat("0", 40)
	b := strings.Repeat("f", 40)
	c := strings.Repeat("0", 39)
	d := strings.Repeat("F", 40)
	e := strings.Repeat("0", 41)

	assert.True(t, isSHA1(a))
	assert.True(t, isSHA1(b))
	assert.False(t, isSHA1(c))
	assert.False(t, isSHA1(d))
	assert.False(t, isSHA1(e))
}
