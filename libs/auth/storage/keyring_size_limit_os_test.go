//go:build darwin || windows

package storage

import (
	"os"
	"strings"
	"testing"
	"uuid"

	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

// TestKeyringStore_Put_RealBackendRejectsLargeToken pins that the go-keyring
// release in go.mod rejects a 3250-byte access token on macOS and Windows,
// which is the behavior fakeBackend's caps in keyring_size_limit_test.go
// transcribe. It is the only test here that observes the real library, so it
// is what fails when a bumped or replaced keyring library accepts larger
// secrets and the modeled caps go stale.
//
// It calls the real backend and therefore runs on CI only (GitHub Actions
// sets CI=true). The library's size check fires before the OS keyring is
// contacted, so on the expected path nothing is written. Only once the cap is
// gone does the token land in the runner's keyring, and then the cleanup
// removes it.
func TestKeyringStore_Put_RealBackendRejectsLargeToken(t *testing.T) {
	if os.Getenv("CI") == "" {
		t.Skip("touches the real OS keyring; runs on CI only")
	}

	store := NewKeyringStore()
	account := keyringProbeAccountPrefix + uuid.New().String()

	// The access token alone exceeds both caps (macOS after base64, Windows
	// raw), so no other token field is needed.
	tok := &oauth2.Token{AccessToken: strings.Repeat("a", issue7707LargeAccessTokenBytes)}

	err := store.Put(account, Entry{Token: tok})
	if err == nil {
		t.Cleanup(func() { _ = store.Delete(account) })
	}
	require.ErrorIs(t, err, keyring.ErrSetDataTooBig,
		"go-keyring accepted a %d-byte access token; the OS caps modeled in fakeBackend are stale, re-verify them against the new backend",
		issue7707LargeAccessTokenBytes)
}
