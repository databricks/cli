//go:build darwin || windows

package storage

import (
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

// issue7707AccessTokenBytes is the size of the U2M access tokens reported in
// https://github.com/databricks/cli/issues/7707. Tokens of 800-950 bytes from
// the same account store fine.
const issue7707AccessTokenBytes = 3250

// The OS backends behind zalando/go-keyring cap the secret size and return
// ErrSetDataTooBig before touching the keyring, so this test does not need an
// unlocked keychain or Credential Manager:
//   - macOS: the base64-encoded `security -i add-generic-password` line must be
//     at most 4096 bytes (https://github.com/zalando/go-keyring/blob/v0.2.8/keyring_darwin.go),
//     which leaves ~3000 bytes for the JSON envelope.
//   - Windows: the credential blob must be at most 2560 bytes
//     (https://github.com/zalando/go-keyring/blob/v0.2.8/keyring_windows.go).
//
// Linux Secret Service has no such cap, hence the build constraint. This test
// pins the current behavior so a fix for #7707 has to flip it on purpose.
func TestKeyringStore_Put_RealBackendRejectsLargeToken(t *testing.T) {
	store := NewKeyringStore()
	account := keyringProbeAccountPrefix + uuid.New().String()
	t.Cleanup(func() {
		// Nothing is written when Put fails as expected; this only matters if
		// the limit disappears and the token lands in the real keyring.
		_ = store.Delete(account)
	})

	// The access token alone already exceeds both caps once base64-encoded,
	// so the exact refresh token size is irrelevant.
	tok := &oauth2.Token{
		AccessToken:  strings.Repeat("a", issue7707AccessTokenBytes),
		TokenType:    "Bearer",
		RefreshToken: strings.Repeat("r", 64),
		Expiry:       time.Now().Add(time.Hour),
	}

	err := store.Put(account, Entry{Token: tok})
	require.ErrorIs(t, err, keyring.ErrSetDataTooBig)
}
