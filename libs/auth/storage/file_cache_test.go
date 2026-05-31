package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func setup(t *testing.T) string {
	tempHomeDir := t.TempDir()
	return filepath.Join(tempHomeDir, "token-cache.json")
}

func TestStoreAndLookup(t *testing.T) {
	c, err := NewFileTokenCache(t.Context(), WithFileLocation(setup(t)))
	require.NoError(t, err)
	err = c.Put("x", Entry{Token: &oauth2.Token{
		AccessToken: "abc",
	}})
	require.NoError(t, err)

	err = c.Put("y", Entry{Token: &oauth2.Token{
		AccessToken: "bcd",
	}})
	require.NoError(t, err)

	got, err := c.Lookup("x")
	require.NoError(t, err)
	assert.Equal(t, "abc", got.Token.AccessToken)

	_, err = c.Lookup("z")
	assert.Equal(t, ErrNotFound, err)
}

func TestNoCacheFileReturnsErrNotConfigured(t *testing.T) {
	l, err := NewFileTokenCache(t.Context(), WithFileLocation(setup(t)))
	require.NoError(t, err)
	_, err = l.Lookup("x")
	assert.Equal(t, ErrNotFound, err)
}

func TestLoadCorruptFile(t *testing.T) {
	f := setup(t)
	err := os.MkdirAll(filepath.Dir(f), ownerExecReadWrite)
	require.NoError(t, err)
	err = os.WriteFile(f, []byte("abc"), ownerExecReadWrite)
	require.NoError(t, err)

	_, err = NewFileTokenCache(t.Context(), WithFileLocation(f))
	assert.EqualError(t, err, "load: parse: invalid character 'a' looking for beginning of value")
}

func TestLoadWrongVersion(t *testing.T) {
	f := setup(t)
	err := os.MkdirAll(filepath.Dir(f), ownerExecReadWrite)
	require.NoError(t, err)
	err = os.WriteFile(f, []byte(`{"version": 823, "things": []}`), ownerExecReadWrite)
	require.NoError(t, err)

	_, err = NewFileTokenCache(t.Context(), WithFileLocation(f))
	assert.EqualError(t, err, "load: needs version 1, got version 823")
}
