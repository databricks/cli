package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReferenceReferencePathForObjectID(t *testing.T) {
	ref := &Reference{
		Type:    ReferenceTypeSHA1,
		Content: strings.Repeat("a", 40),
	}
	_, err := ref.ResolvePath()
	assert.ErrorIs(t, err, ErrNotAReferencePointer)
}

func TestReferenceCurrentBranchForObjectID(t *testing.T) {
	ref := &Reference{
		Type:    ReferenceTypeSHA1,
		Content: strings.Repeat("a", 40),
	}
	_, err := ref.CurrentBranch()
	assert.ErrorIs(t, err, ErrNotABranch)
}

func TestReferenceCurrentBranchForReference(t *testing.T) {
	ref := &Reference{
		Type:    ReferenceTypePointer,
		Content: `ref: refs/heads/my-branch`,
	}
	branch, err := ref.CurrentBranch()
	assert.NoError(t, err)
	assert.Equal(t, "my-branch", branch)
}

func TestReferenceReferencePathForReference(t *testing.T) {
	ref := &Reference{
		Type:    ReferenceTypePointer,
		Content: `ref: refs/heads/my-branch`,
	}
	path, err := ref.ResolvePath()
	assert.NoError(t, err)
	assert.Equal(t, "refs/heads/my-branch", path)
}

func TestReferenceLoadingForObjectID(t *testing.T) {
	tmp := t.TempDir()
	f, err := os.Create(filepath.Join(tmp, "HEAD"))
	require.NoError(t, err)
	defer f.Close()
	_, err = f.WriteString(strings.Repeat("e", 40) + "\r\n")
	require.NoError(t, err)

	ref, err := LoadReferenceFile(vfs.MustNew(tmp), "HEAD")
	assert.NoError(t, err)
	assert.Equal(t, ReferenceTypeSHA1, ref.Type)
	assert.Equal(t, strings.Repeat("e", 40), ref.Content)
}

func TestReferenceLoadingForReference(t *testing.T) {
	tmp := t.TempDir()
	f, err := os.OpenFile(filepath.Join(tmp, "HEAD"), os.O_CREATE|os.O_WRONLY, os.ModePerm)
	require.NoError(t, err)
	defer f.Close()
	_, err = f.WriteString("ref: refs/heads/foo\n")
	require.NoError(t, err)

	ref, err := LoadReferenceFile(vfs.MustNew(tmp), "HEAD")
	assert.NoError(t, err)
	assert.Equal(t, ReferenceTypePointer, ref.Type)
	assert.Equal(t, "ref: refs/heads/foo", ref.Content)
}

func TestReferenceLoadingFailsForInvalidContent(t *testing.T) {
	tmp := t.TempDir()
	f, err := os.OpenFile(filepath.Join(tmp, "HEAD"), os.O_CREATE|os.O_WRONLY, os.ModePerm)
	require.NoError(t, err)
	defer f.Close()
	_, err = f.WriteString("abc")
	require.NoError(t, err)

	_, err = LoadReferenceFile(vfs.MustNew(tmp), "HEAD")
	assert.ErrorContains(t, err, "unknown format for git HEAD")
}

func TestReferenceIsSha1(t *testing.T) {
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
