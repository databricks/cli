//go:build darwin || windows

package storage

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"uuid"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

// TestKeyringSet_RejectsLargeSecret pins that the go-keyring
// release in go.mod rejects a 3250-byte secret on macOS and Windows,
// which is the behavior fakeBackend's caps in keyring_size_limit_test.go
// transcribe. It is the only test here that observes the real library reject
// a secret, so it is what fails when a bumped or replaced keyring library
// accepts larger secrets and the modeled caps go stale.
//
// It calls go-keyring directly: the store splits a token of this size, and
// zalandoBackend checks the size itself on macOS.
//
// It calls the real backend and therefore runs on CI only (GitHub Actions
// sets CI=true). The library's size check fires before the OS keyring is
// contacted, so on the expected path nothing is written. Only once the cap is
// gone does the secret land in the runner's keyring, and then the cleanup
// removes it.
func TestKeyringSet_RejectsLargeSecret(t *testing.T) {
	if os.Getenv("CI") == "" {
		t.Skip("touches the real OS keyring; runs on CI only")
	}

	account := keyringProbeAccountPrefix + uuid.New().String()

	// The secret alone exceeds both caps (macOS after base64, Windows raw).
	secret := strings.Repeat("a", issue7707LargeAccessTokenBytes)

	err := keyring.Set(keyringServiceName, account, secret)
	if err == nil {
		t.Cleanup(func() { _ = keyring.Delete(keyringServiceName, account) })
	}
	require.ErrorIs(t, err, keyring.ErrSetDataTooBig,
		"go-keyring accepted a %d-byte secret; the OS caps modeled in fakeBackend are stale, re-verify them against the new backend",
		issue7707LargeAccessTokenBytes)
}

// newRealKeyringStore returns a store on the real OS keyring and a key that no
// other test or profile uses. The store has the test timeout rather than the
// 3-second production one, so that a slow runner cannot fail these tests. The
// cleanup removes the entry and its parts through the backend, so it does not
// depend on the store's Delete and also runs when the test fails.
//
// The tests that use it write to the real OS keyring and therefore run on CI
// only (GitHub Actions sets CI=true).
func newRealKeyringStore(t *testing.T) (*keyringStore, keyringBackend, string) {
	t.Helper()
	if os.Getenv("CI") == "" {
		t.Skip("touches the real OS keyring; runs on CI only")
	}

	backend := zalandoBackend{}
	key := keyringProbeAccountPrefix + uuid.New().String()
	t.Cleanup(func() {
		for _, account := range itemsOf(key, maxTestParts) {
			_ = backend.Delete(keyringServiceName, account)
		}
	})
	return newTestStore(backend), backend, key
}

func TestKeyringStore_RealBackend_SplitTokenRoundTrips(t *testing.T) {
	cases := []struct {
		accessTokenSize int
		wantParts       int
	}{
		// https://github.com/databricks/cli/issues/7707
		{accessTokenSize: issue7707LargeAccessTokenBytes, wantParts: 2},
		{accessTokenSize: 5000, wantParts: 3},
	}

	for _, tc := range cases {
		t.Run(strconv.Itoa(tc.accessTokenSize)+" bytes", func(t *testing.T) {
			c, backend, key := newRealKeyringStore(t)
			want := testToken(tc.accessTokenSize)

			require.NoError(t, c.Put(key, Entry{Token: want}))
			assert.Equal(t, itemsOf(key, tc.wantParts), presentItems(t, backend, key))

			got, err := c.Lookup(key)
			require.NoError(t, err)
			assert.Equal(t, want.AccessToken, got.Token.AccessToken)
			assert.Equal(t, want.TokenType, got.Token.TokenType)
			assert.Equal(t, want.RefreshToken, got.Token.RefreshToken)
			assert.True(t, want.Expiry.Equal(got.Token.Expiry))

			require.NoError(t, c.Delete(key))
			assert.Empty(t, presentItems(t, backend, key))
		})
	}
}

// Released CLI versions on the same machine read a token that fits, so it
// must stay one item in their format.
func TestKeyringStore_RealBackend_TokenThatFitsIsOneItem(t *testing.T) {
	c, backend, key := newRealKeyringStore(t)
	tok := testToken(issue7707SmallAccessTokenBytes)

	require.NoError(t, c.Put(key, Entry{Token: tok}))

	got, err := backend.Get(keyringServiceName, key)
	require.NoError(t, err)
	assert.Equal(t, releasedEntry(t, tok), got)
	assert.Equal(t, itemsOf(key, 0), presentItems(t, backend, key))
}

func TestKeyringStore_RealBackend_TokenSizeTransitions(t *testing.T) {
	c, backend, key := newRealKeyringStore(t)
	assertTokenSizeTransitions(t, c, backend, key)
}

// The cap that fakeBackend models for this OS must match the real backend at
// its edge: the largest token that fits is stored and read back whole in one
// item, and one byte more is split. On macOS this is where go-keyring's check
// and the 4096-byte line of `security -i` meet.
func TestKeyringStore_RealBackend_SplitsOnlyTokenThatDoesNotFit(t *testing.T) {
	c, backend, key := newRealKeyringStore(t)
	secretTooBig := darwinSecurityCommandTooBig
	if runtime.GOOS == "windows" {
		secretTooBig = windowsCredentialBlobTooBig
	}
	n := largestAccessTokenInOneItem(t, secretTooBig, key)

	fits := testToken(n)
	require.NoError(t, c.Put(key, Entry{Token: fits}))
	raw, err := backend.Get(keyringServiceName, key)
	require.NoError(t, err)
	assert.Equal(t, releasedEntry(t, fits), raw)
	assert.Equal(t, itemsOf(key, 0), presentItems(t, backend, key))

	tooBig := testToken(n + 1)
	require.NoError(t, c.Put(key, Entry{Token: tooBig}))
	assert.Equal(t, itemsOf(key, 2), presentItems(t, backend, key))
	got, err := c.Lookup(key)
	require.NoError(t, err)
	assert.Equal(t, tooBig.AccessToken, got.Token.AccessToken)
}
