package storage

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// Access token sizes reported in https://github.com/databricks/cli/issues/7707:
// U2M tokens of about 3250 bytes fail to store on macOS, while tokens of
// 800-950 bytes from the same account store fine.
const (
	issue7707LargeAccessTokenBytes = 3250
	issue7707SmallAccessTokenBytes = 950
)

// darwinSecurityCommandTooBig mirrors the size check in zalando/go-keyring
// v0.2.8 keyring_darwin.go. The backend base64-encodes the secret, builds one
// `add-generic-password` line for `security -i`, and rejects the write when
// that line exceeds 4096 bytes: the tool reads at most 4095 bytes per line and
// would otherwise store a truncated secret.
func darwinSecurityCommandTooBig(service, account, secret string) bool {
	password := "go-keyring-base64:" + base64.StdEncoding.EncodeToString([]byte(secret))
	command := fmt.Sprintf("add-generic-password -U -s %s -a %s -w %s\n", shellQuote(service), shellQuote(account), shellQuote(password))
	return len(command) > 4096
}

// windowsCredentialBlobTooBig mirrors zalando/go-keyring v0.2.8
// keyring_windows.go, which stores the raw secret and rejects blobs over 2560
// bytes.
func windowsCredentialBlobTooBig(_, _, secret string) bool {
	return len(secret) > 2560
}

// shellQuote mirrors go-keyring's internal shellescape.Quote: a string made of
// [A-Za-z0-9_@%+=:,./-] passes through unchanged, anything else is
// single-quoted.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if strings.ContainsFunc(s, func(r rune) bool { return !shellSafe(r) }) {
		return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
	}
	return s
}

func shellSafe(r rune) bool {
	if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
		return true
	}
	return strings.ContainsRune("_@%+=:,./-", r)
}

// osSecretCaps are go-keyring's per-OS secret caps as fakeBackend models them.
var osSecretCaps = []struct {
	name         string
	secretTooBig func(service, account, secret string) bool
}{
	{name: "darwin security -i line", secretTooBig: darwinSecurityCommandTooBig},
	{name: "windows credential blob", secretTooBig: windowsCredentialBlobTooBig},
	{name: "linux secret service has no cap", secretTooBig: nil},
}

// TestKeyringStore_Put_OSSecretSizeCaps pins how the CLI's keyring envelope
// fares against go-keyring's per-OS secret caps. The caps are modeled in
// fakeBackend so the test is keychain-agnostic and runs on every OS;
// TestKeyringSet_RejectsLargeSecret in
// keyring_size_limit_os_test.go checks the real library on CI and fails when
// the modeled caps go stale. A fix for #7707 that changes what Put stores
// must update this test on purpose.
func TestKeyringStore_Put_OSSecretSizeCaps(t *testing.T) {
	for _, tc := range osSecretCaps {
		t.Run(tc.name, func(t *testing.T) {
			backend := newFakeBackend()
			backend.secretTooBig = tc.secretTooBig
			store := newTestStore(backend)

			small := &oauth2.Token{AccessToken: strings.Repeat("a", issue7707SmallAccessTokenBytes)}
			require.NoError(t, store.Put("my-profile", Entry{Token: small}))
			got, err := store.Lookup("my-profile")
			require.NoError(t, err)
			assert.Equal(t, small.AccessToken, got.Token.AccessToken)

			// The access token alone exceeds both caps (macOS after base64,
			// Windows raw), so no other token field is needed.
			large := &oauth2.Token{AccessToken: strings.Repeat("a", issue7707LargeAccessTokenBytes)}
			require.NoError(t, store.Put("my-profile", Entry{Token: large}))
			got, err = store.Lookup("my-profile")
			require.NoError(t, err)
			assert.Equal(t, large.AccessToken, got.Token.AccessToken)
		})
	}
}

// TestDarwinCommandTooBig pins that the macOS size check, which lets the store
// skip go-keyring, only rejects a secret that go-keyring rejects too. The check
// leaves out the shell quotes that go-keyring adds, so it is exact for an
// account that needs none.
func TestDarwinCommandTooBig(t *testing.T) {
	accounts := []string{
		"my-profile",
		"https://accounts.cloud.databricks.test/oidc/accounts/968367da-7edd-44f7-9dea-3e0b20b0ec97",
		// go-keyring quotes these.
		"my profile",
		"my-profile#access-token-0",
	}
	for _, account := range accounts {
		quoted := shellQuote(account) != account
		for n := 2800; n <= 3100; n++ {
			secret := strings.Repeat("a", n)
			got := darwinCommandTooBig(keyringServiceName, account, secret)
			want := darwinSecurityCommandTooBig(keyringServiceName, account, secret)
			require.False(t, got && !want, "account %q, %d-byte secret: rejected, but go-keyring stores it", account, n)
			if !quoted {
				require.Equal(t, want, got, "account %q, %d-byte secret", account, n)
			}
		}
	}
}
