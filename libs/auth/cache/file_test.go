package cache

import (
	"encoding/json"
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
	c := &FileTokenCache{}
	err := c.Store("x", &oauth2.Token{
		AccessToken: "abc",
	})
	require.NoError(t, err)

	err = c.Store("y", &oauth2.Token{
		AccessToken: "bcd",
	})
	require.NoError(t, err)

	l := &FileTokenCache{}
	tok, err := l.Lookup("x")
	require.NoError(t, err)
	assert.Equal(t, "abc", tok.AccessToken)
	assert.Len(t, l.Tokens, 2)

	_, err = l.Lookup("z")
	assert.Equal(t, ErrNotConfigured, err)
}

func TestNoCacheFileReturnsErrNotConfigured(t *testing.T) {
	setup(t)
	l := &FileTokenCache{}
	_, err := l.Lookup("x")
	assert.Equal(t, ErrNotConfigured, err)
}

func TestLoadCorruptFile(t *testing.T) {
	home := setup(t)
	f := filepath.Join(home, tokenCacheFile)
	err := os.MkdirAll(filepath.Dir(f), ownerExecReadWrite)
	require.NoError(t, err)
	err = os.WriteFile(f, []byte("abc"), ownerExecReadWrite)
	require.NoError(t, err)

	l := &FileTokenCache{}
	_, err = l.Lookup("x")
	assert.EqualError(t, err, "load: parse: invalid character 'a' looking for beginning of value")
}

func TestLoadWrongVersion(t *testing.T) {
	home := setup(t)
	f := filepath.Join(home, tokenCacheFile)
	err := os.MkdirAll(filepath.Dir(f), ownerExecReadWrite)
	require.NoError(t, err)
	err = os.WriteFile(f, []byte(`{"version": 823, "things": []}`), ownerExecReadWrite)
	require.NoError(t, err)

	l := &FileTokenCache{}
	_, err = l.Lookup("x")
	assert.EqualError(t, err, "load: needs version 1, got version 823")
}

func TestDevNull(t *testing.T) {
	t.Setenv(homeEnvVar, "/dev/null")
	l := &FileTokenCache{}
	_, err := l.Lookup("x")
	// macOS/Linux: load: read: open /dev/null/.databricks/token-cache.json:
	// windows: databricks OAuth is not configured for this host
	assert.Error(t, err)
}

func TestStoreOnDev(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.SkipNow()
	}
	t.Setenv(homeEnvVar, "/dev")
	c := &FileTokenCache{}
	err := c.Store("x", &oauth2.Token{
		AccessToken: "abc",
	})
	// Linux: permission denied
	// macOS: read-only file system
	assert.Error(t, err)
}

func TestStoreAndDeleteKey(t *testing.T) {
	setup(t)
	c := &FileTokenCache{}
	err := c.Store("x", &oauth2.Token{
		AccessToken: "abc",
	})
	require.NoError(t, err)

	err = c.Store("y", &oauth2.Token{
		AccessToken: "bcd",
	})
	require.NoError(t, err)

	l := &FileTokenCache{}
	err = l.Delete("x")
	require.NoError(t, err)
	assert.Equal(t, 1, len(l.Tokens))

	_, err = l.Lookup("x")
	assert.Equal(t, ErrNotConfigured, err)

	tok, err := l.Lookup("y")
	require.NoError(t, err)
	assert.Equal(t, "bcd", tok.AccessToken)
}

func TestDeleteKeyNotExist(t *testing.T) {
	c := &FileTokenCache{
		Tokens: map[string]*oauth2.Token{},
	}
	err := c.Delete("x")
	assert.Equal(t, ErrNotConfigured, err)

	_, err = c.Lookup("x")
	assert.Equal(t, ErrNotConfigured, err)
}

func TestWrite(t *testing.T) {
	tempFile := filepath.Join(t.TempDir(), "token-cache.json")

	tokenMap := map[string]*oauth2.Token{}
	token := &oauth2.Token{
		AccessToken: "some-access-token",
	}
	tokenMap["test"] = token

	cache := &FileTokenCache{
		fileLocation: tempFile,
		Tokens:       tokenMap,
	}

	err := cache.write()
	assert.NoError(t, err)

	content, err := os.ReadFile(tempFile)
	require.NoError(t, err)

	expected, _ := json.MarshalIndent(&cache, "", "  ")
	assert.Equal(t, content, expected)
}
