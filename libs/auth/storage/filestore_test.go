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
	c, err := NewFileStore(t.Context(), WithFileLocation(setup(t)))
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

func TestPendingLoginIsApartFromTokens(t *testing.T) {
	f := setup(t)
	c, err := NewFileStore(t.Context(), WithFileLocation(f))
	require.NoError(t, err)
	pendingLogins := c.(PendingLoginStore)
	pending := PendingLogin{
		DeviceCode: "device-code",
		UserCode:   "WDJBMJHT",
		Host:       "https://workspace.test",
		Scopes:     []string{"sql"},
	}
	require.NoError(t, c.Put("x", Entry{Token: &oauth2.Token{AccessToken: "abc"}}))
	require.NoError(t, pendingLogins.PutPendingLogin("x", pending))

	got, err := c.Lookup("x")
	require.NoError(t, err)
	assert.Equal(t, "abc", got.Token.AccessToken)
	gotPending, err := pendingLogins.LookupPendingLogin("x")
	require.NoError(t, err)
	assert.Equal(t, pending, gotPending)
	_, err = pendingLogins.LookupPendingLogin("y")
	assert.ErrorIs(t, err, ErrNotFound)

	raw, err := os.ReadFile(f)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"version": 1,
		"tokens": {
			"x": {"access_token": "abc", "expiry": "0001-01-01T00:00:00Z"}
		},
		"pending_logins": {
			"x": {"device_code": "device-code", "user_code": "WDJBMJHT", "host": "https://workspace.test", "scopes": ["sql"]}
		}
	}`, string(raw))

	require.NoError(t, pendingLogins.DeletePendingLogin("x"))
	require.NoError(t, pendingLogins.DeletePendingLogin("y"))
	_, err = pendingLogins.LookupPendingLogin("x")
	assert.ErrorIs(t, err, ErrNotFound)
	got, err = c.Lookup("x")
	require.NoError(t, err)
	assert.Equal(t, "abc", got.Token.AccessToken)
}

func TestNoStoreFileReturnsErrNotConfigured(t *testing.T) {
	l, err := NewFileStore(t.Context(), WithFileLocation(setup(t)))
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

	_, err = NewFileStore(t.Context(), WithFileLocation(f))
	assert.EqualError(t, err, "load: parse: invalid character 'a' looking for beginning of value")
}

func TestLoadWrongVersion(t *testing.T) {
	f := setup(t)
	err := os.MkdirAll(filepath.Dir(f), ownerExecReadWrite)
	require.NoError(t, err)
	err = os.WriteFile(f, []byte(`{"version": 823, "things": []}`), ownerExecReadWrite)
	require.NoError(t, err)

	_, err = NewFileStore(t.Context(), WithFileLocation(f))
	assert.EqualError(t, err, "load: needs version 1, got version 823")
}
