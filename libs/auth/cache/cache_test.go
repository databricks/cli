package cache

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

var homeEnvVar = "HOME"

func init() {
	if runtime.GOOS == "windows" {
		homeEnvVar = "USERPROFILE"
	}
}

func setup(t *testing.T) string {
	tempHomeDir := t.TempDir()
	t.Setenv(homeEnvVar, tempHomeDir)
	return tempHomeDir
}

func TestStoreAndLookup(t *testing.T) {
	setup(t)
	c := &TokenCache{}
	err := c.Store("x", &oauth2.Token{
		AccessToken: "abc",
	})
	require.NoError(t, err)

	err = c.Store("y", &oauth2.Token{
		AccessToken: "bcd",
	})
	require.NoError(t, err)

	l := &TokenCache{}
	tok, err := l.Lookup("x")
	require.NoError(t, err)
	assert.Equal(t, "abc", tok.AccessToken)
	assert.Equal(t, 2, len(l.Tokens))

	_, err = l.Lookup("z")
	assert.Equal(t, ErrNotConfigured, err)
}

func TestNoCacheFileReturnsErrNotConfigured(t *testing.T) {
	setup(t)
	l := &TokenCache{}
	_, err := l.Lookup("x")
	assert.Equal(t, ErrNotConfigured, err)
}

func TestLoadCorruptFile(t *testing.T) {
	home := setup(t)
	f := filepath.Join(home, tokenCacheFile)
	err := os.MkdirAll(filepath.Dir(f), tokenCacheChmod)
	require.NoError(t, err)
	err = os.WriteFile(f, []byte("abc"), tokenCacheChmod)
	require.NoError(t, err)

	l := &TokenCache{}
	_, err = l.Lookup("x")
	assert.EqualError(t, err, "load: parse: invalid character 'a' looking for beginning of value")
}

func TestLoadWrongVersion(t *testing.T) {
	home := setup(t)
	f := filepath.Join(home, tokenCacheFile)
	err := os.MkdirAll(filepath.Dir(f), tokenCacheChmod)
	require.NoError(t, err)
	err = os.WriteFile(f, []byte(`{"version": 823, "things": []}`), tokenCacheChmod)
	require.NoError(t, err)

	l := &TokenCache{}
	_, err = l.Lookup("x")
	assert.EqualError(t, err, "load: needs version 1, got version 823")
}

func TestDevNull(t *testing.T) {
	t.Setenv(homeEnvVar, "/dev/null")
	l := &TokenCache{}
	_, err := l.Lookup("x")
	assert.EqualError(t, err, "load: read: open /dev/null/.databricks/token-cache.json: not a directory")
}

func TestStoreOnRoot(t *testing.T) {
	t.Setenv(homeEnvVar, "/")
	c := &TokenCache{}
	err := c.Store("x", &oauth2.Token{
		AccessToken: "abc",
	})
	assert.EqualError(t, err, "mkdir: mkdir /.databricks: read-only file system")
}
