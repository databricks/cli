package storage

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/credentials/u2m"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// stubStore is a test double for Store that records the source it was
// constructed from. It lets the tests confirm which factory ran.
type stubStore struct{ source string }

func (stubStore) Put(string, Entry) error      { return nil }
func (stubStore) Lookup(string) (Entry, error) { return Entry{}, ErrNotFound }
func (stubStore) Delete(string) error          { return nil }

// memStore is a functional in-memory Store for exercising the OAuth wrappers.
type memStore struct{ entries map[string]Entry }

func newMemStore() *memStore { return &memStore{entries: map[string]Entry{}} }

func (m *memStore) Put(key string, e Entry) error { m.entries[key] = e; return nil }

func (m *memStore) Lookup(key string) (Entry, error) {
	e, ok := m.entries[key]
	if !ok {
		return Entry{}, ErrNotFound
	}
	return e, nil
}

func (m *memStore) Delete(key string) error { delete(m.entries, key); return nil }

func fakeFactories(t *testing.T) storeFactories {
	t.Helper()
	return storeFactories{
		newFile:          func(context.Context) (Store, error) { return stubStore{source: "file"}, nil },
		newKeyring:       func() Store { return stubStore{source: "keyring"} },
		probeKeyring:     func() error { return nil },
		probeKeyringRead: func() error { return nil },
	}
}

// hermetic isolates the test from the caller's real env vars and
// .databrickscfg so ResolveStorageMode starts from a clean default.
func hermetic(t *testing.T) {
	t.Helper()
	t.Setenv(EnvVar, "")
	t.Setenv("DATABRICKS_CONFIG_FILE", filepath.Join(t.TempDir(), "databrickscfg"))
}

func TestResolveStore_DefaultsToSecureKeyring(t *testing.T) {
	hermetic(t)
	ctx := t.Context()

	got, mode, err := resolveStoreWith(ctx, "", fakeFactories(t))

	require.NoError(t, err)
	assert.Equal(t, StorageModeSecure, mode)
	assert.Equal(t, "keyring", got.(stubStore).source)
}

func TestResolveStore_OverrideSecureUsesKeyring(t *testing.T) {
	hermetic(t)
	ctx := t.Context()

	got, mode, err := resolveStoreWith(ctx, StorageModeSecure, fakeFactories(t))

	require.NoError(t, err)
	assert.Equal(t, StorageModeSecure, mode)
	assert.Equal(t, "keyring", got.(stubStore).source)
}

func TestResolveStore_EnvVarSelectsSecure(t *testing.T) {
	hermetic(t)
	ctx := env.Set(t.Context(), EnvVar, "secure")

	got, mode, err := resolveStoreWith(ctx, "", fakeFactories(t))

	require.NoError(t, err)
	assert.Equal(t, StorageModeSecure, mode)
	assert.Equal(t, "keyring", got.(stubStore).source)
}

func TestResolveStore_PlaintextOverrideUsesFile(t *testing.T) {
	hermetic(t)
	ctx := t.Context()

	got, mode, err := resolveStoreWith(ctx, StorageModePlaintext, fakeFactories(t))

	require.NoError(t, err)
	assert.Equal(t, StorageModePlaintext, mode)
	assert.Equal(t, "file", got.(stubStore).source)
}

func TestResolveStore_InvalidOverrideReturnsError(t *testing.T) {
	hermetic(t)
	ctx := t.Context()

	_, _, err := resolveStoreWith(ctx, StorageMode("bogus"), fakeFactories(t))

	require.Error(t, err)
	assert.Contains(t, err.Error(), `unsupported storage mode "bogus"`)
}

func TestResolveStore_InvalidEnvReturnsError(t *testing.T) {
	hermetic(t)
	ctx := env.Set(t.Context(), EnvVar, "bogus")

	_, _, err := resolveStoreWith(ctx, "", fakeFactories(t))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABRICKS_AUTH_STORAGE")
}

func TestResolveStore_FileFactoryErrorPropagates(t *testing.T) {
	hermetic(t)
	ctx := t.Context()
	boom := errors.New("disk full")
	factories := storeFactories{
		newFile:          func(context.Context) (Store, error) { return nil, boom },
		newKeyring:       func() Store { return stubStore{source: "keyring"} },
		probeKeyring:     func() error { return nil },
		probeKeyringRead: func() error { return nil },
	}

	_, _, err := resolveStoreWith(ctx, StorageModePlaintext, factories)

	require.Error(t, err)
	assert.ErrorIs(t, err, boom)
}

// applyReadFallback mirrors applyLoginFallback's logic on the read path:
// keyring is probed read-only, definitive failures fall through to the file
// cache so legacy plaintext tokens stay reachable, timeouts stay on the
// keyring, and explicit-secure is honored even when the probe fails.

func TestApplyReadFallback_PlaintextSkipsProbe(t *testing.T) {
	hermetic(t)
	ctx := t.Context()
	probed := false
	f := fakeFactories(t)
	f.probeKeyringRead = func() error {
		probed = true
		return nil
	}

	got, mode, err := applyReadFallback(ctx, StorageModePlaintext, false, f)

	require.NoError(t, err)
	assert.Equal(t, StorageModePlaintext, mode)
	assert.Equal(t, "file", got.(stubStore).source)
	assert.False(t, probed, "probe must not run when mode is already plaintext")
}

func TestApplyReadFallback_ExplicitSecureSkipsProbe(t *testing.T) {
	hermetic(t)
	ctx := t.Context()
	probed := false
	f := fakeFactories(t)
	f.probeKeyringRead = func() error {
		probed = true
		return errors.New("unreachable")
	}

	got, mode, err := applyReadFallback(ctx, StorageModeSecure, true, f)

	require.NoError(t, err)
	assert.Equal(t, StorageModeSecure, mode)
	assert.Equal(t, "keyring", got.(stubStore).source)
	assert.False(t, probed, "probe must not run when user is explicit about secure mode")
}

func TestApplyReadFallback_DefaultSecure_ProbeOK_UsesKeyring(t *testing.T) {
	hermetic(t)
	ctx := t.Context()

	got, mode, err := applyReadFallback(ctx, StorageModeSecure, false, fakeFactories(t))

	require.NoError(t, err)
	assert.Equal(t, StorageModeSecure, mode)
	assert.Equal(t, "keyring", got.(stubStore).source)
}

func TestApplyReadFallback_DefaultSecure_ProbeFail_FallsBack(t *testing.T) {
	hermetic(t)
	ctx := t.Context()
	configPath := env.Get(ctx, "DATABRICKS_CONFIG_FILE")

	f := fakeFactories(t)
	f.probeKeyringRead = func() error { return errors.New("no keyring") }

	got, mode, err := applyReadFallback(ctx, StorageModeSecure, false, f)

	require.NoError(t, err)
	assert.Equal(t, StorageModePlaintext, mode)
	assert.Equal(t, "file", got.(stubStore).source)

	// Read-path fallback must NOT pin: pinning is reserved for login,
	// where the write-probe gives stronger evidence of unavailability.
	persisted, gerr := databrickscfg.GetConfiguredAuthStorage(ctx, configPath)
	require.NoError(t, gerr)
	assert.Empty(t, persisted, "read-path fallback must not persist auth_storage")
}

// A timeout could mean a locked keyring that will work once the user unlocks
// it. Stay on the keyring so the actual Lookup surfaces the real outcome
// rather than silently routing reads to the file cache.
func TestApplyReadFallback_DefaultSecure_ProbeTimeout_StaysOnKeyring(t *testing.T) {
	cases := []struct {
		name     string
		probeErr error
	}{
		{"bare TimeoutError", &TimeoutError{Op: "get"}},
		{"wrapped TimeoutError", fmt.Errorf("get: %w", &TimeoutError{Op: "get"})},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hermetic(t)
			ctx := t.Context()
			configPath := env.Get(ctx, "DATABRICKS_CONFIG_FILE")

			f := fakeFactories(t)
			f.probeKeyringRead = func() error { return tc.probeErr }

			got, mode, err := applyReadFallback(ctx, StorageModeSecure, false, f)

			require.NoError(t, err)
			assert.Equal(t, StorageModeSecure, mode)
			assert.Equal(t, "keyring", got.(stubStore).source)

			persisted, gerr := databrickscfg.GetConfiguredAuthStorage(ctx, configPath)
			require.NoError(t, gerr)
			assert.Empty(t, persisted, "probe timeout must not persist anything")
		})
	}
}

func TestResolveStoreForLogin_PlaintextSkipsProbe(t *testing.T) {
	hermetic(t)
	ctx := t.Context()
	probed := false
	f := fakeFactories(t)
	f.probeKeyring = func() error {
		probed = true
		return nil
	}

	got, mode, err := resolveStoreForLoginWith(ctx, StorageModePlaintext, f)

	require.NoError(t, err)
	assert.Equal(t, StorageModePlaintext, mode)
	assert.Equal(t, "file", got.(stubStore).source)
	assert.False(t, probed, "probe must not run when mode is already plaintext")
}

func TestResolveStoreForLogin_SecureProbeOK(t *testing.T) {
	hermetic(t)
	ctx := env.Set(t.Context(), EnvVar, "secure")

	got, mode, err := resolveStoreForLoginWith(ctx, "", fakeFactories(t))

	require.NoError(t, err)
	assert.Equal(t, StorageModeSecure, mode)
	assert.Equal(t, "keyring", got.(stubStore).source)
}

func TestResolveStoreForLogin_ExplicitEnvSecure_ProbeFail_Errors(t *testing.T) {
	hermetic(t)
	ctx := env.Set(t.Context(), EnvVar, "secure")
	configPath := env.Get(ctx, "DATABRICKS_CONFIG_FILE")

	f := fakeFactories(t)
	f.probeKeyring = func() error { return errors.New("no keyring") }

	_, _, err := resolveStoreForLoginWith(ctx, "", f)
	require.Error(t, err)
	assert.ErrorContains(t, err, "secure storage was requested")

	persisted, gerr := databrickscfg.GetConfiguredAuthStorage(ctx, configPath)
	require.NoError(t, gerr)
	assert.Empty(t, persisted, "env-set secure must not be persisted as plaintext")
}

func TestResolveStoreForLogin_ExplicitConfigSecure_ProbeFail_Errors(t *testing.T) {
	hermetic(t)
	ctx := t.Context()
	configPath := env.Get(ctx, "DATABRICKS_CONFIG_FILE")
	require.NoError(t, os.WriteFile(configPath, []byte("[__settings__]\nauth_storage = secure\n"), 0o600))

	f := fakeFactories(t)
	f.probeKeyring = func() error { return errors.New("no keyring") }

	_, _, err := resolveStoreForLoginWith(ctx, "", f)
	require.Error(t, err)
	assert.ErrorContains(t, err, "secure storage was requested")

	persisted, gerr := databrickscfg.GetConfiguredAuthStorage(ctx, configPath)
	require.NoError(t, gerr)
	assert.Equal(t, "secure", persisted, "config-set secure must not be silently rewritten")
}

func TestResolveStoreForLogin_ExplicitOverrideSecure_ProbeFail_Errors(t *testing.T) {
	hermetic(t)
	ctx := t.Context()

	f := fakeFactories(t)
	f.probeKeyring = func() error { return errors.New("no keyring") }

	_, _, err := resolveStoreForLoginWith(ctx, StorageModeSecure, f)
	require.Error(t, err)
	assert.ErrorContains(t, err, "secure storage was requested")
}

func TestApplyLoginFallback_DefaultSecure_ProbeFail_FallsBackAndPersists(t *testing.T) {
	hermetic(t)
	ctx := t.Context()
	configPath := env.Get(ctx, "DATABRICKS_CONFIG_FILE")

	f := fakeFactories(t)
	f.probeKeyring = func() error { return errors.New("no keyring") }

	got, mode, err := applyLoginFallback(ctx, StorageModeSecure, false, f)

	require.NoError(t, err)
	assert.Equal(t, StorageModePlaintext, mode)
	assert.Equal(t, "file", got.(stubStore).source)

	persisted, err := databrickscfg.GetConfiguredAuthStorage(ctx, configPath)
	require.NoError(t, err)
	assert.Equal(t, "plaintext", persisted, "default-mode fallback must persist auth_storage = plaintext")
}

func TestApplyLoginFallback_ExplicitSecure_ProbeFail_Errors(t *testing.T) {
	hermetic(t)
	ctx := t.Context()
	configPath := env.Get(ctx, "DATABRICKS_CONFIG_FILE")

	f := fakeFactories(t)
	f.probeKeyring = func() error { return errors.New("no keyring") }

	_, _, err := applyLoginFallback(ctx, StorageModeSecure, true, f)
	require.Error(t, err)
	assert.ErrorContains(t, err, "secure storage was requested")

	persisted, gerr := databrickscfg.GetConfiguredAuthStorage(ctx, configPath)
	require.NoError(t, gerr)
	assert.Empty(t, persisted, "explicit-secure error must not write config")
}

// A locked keyring with a slow user surfaces as TimeoutError. We want login
// to stay on the keyring so the final Store lands there once the user has
// finished unlocking, regardless of whether secure was explicit. Cover both
// the bare TimeoutError (in case probe wraps thinner in the future) and the
// real wrapped form returned by probeWithBackend.
func TestApplyLoginFallback_ProbeTimeout_StaysOnKeyring(t *testing.T) {
	cases := []struct {
		name     string
		explicit bool
		probeErr error
	}{
		{"default-secure, bare TimeoutError", false, &TimeoutError{Op: "set"}},
		{"default-secure, wrapped TimeoutError", false, fmt.Errorf("write: %w", &TimeoutError{Op: "set"})},
		{"explicit-secure, bare TimeoutError", true, &TimeoutError{Op: "set"}},
		{"explicit-secure, wrapped TimeoutError", true, fmt.Errorf("write: %w", &TimeoutError{Op: "set"})},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hermetic(t)
			ctx := t.Context()
			configPath := env.Get(ctx, "DATABRICKS_CONFIG_FILE")

			f := fakeFactories(t)
			f.probeKeyring = func() error { return tc.probeErr }

			got, mode, err := applyLoginFallback(ctx, StorageModeSecure, tc.explicit, f)

			require.NoError(t, err)
			assert.Equal(t, StorageModeSecure, mode)
			assert.Equal(t, "keyring", got.(stubStore).source)

			persisted, gerr := databrickscfg.GetConfiguredAuthStorage(ctx, configPath)
			require.NoError(t, gerr)
			assert.Empty(t, persisted, "probe timeout must not persist plaintext fallback")
		})
	}
}

func TestPinSecureMode(t *testing.T) {
	cases := []struct {
		name        string
		mode        StorageMode
		override    StorageMode
		envValue    string
		configBody  string
		wantWritten string
	}{
		{
			name:        "secure from default persists secure",
			mode:        StorageModeSecure,
			wantWritten: "secure",
		},
		{
			name:        "plaintext mode is a no-op",
			mode:        StorageModePlaintext,
			wantWritten: "",
		},
		{
			name:        "secure from env is a no-op",
			mode:        StorageModeSecure,
			envValue:    "secure",
			wantWritten: "",
		},
		{
			name:        "secure from config is a no-op (already pinned)",
			mode:        StorageModeSecure,
			configBody:  "[__settings__]\nauth_storage = secure\n",
			wantWritten: "secure",
		},
		{
			// The override signal is per-invocation, so persisting it to
			// config would silently turn an ephemeral choice into a
			// persistent one. Honor the caller's explicit override by
			// no-op'ing the pin.
			name:        "secure from override is a no-op",
			mode:        StorageModeSecure,
			override:    StorageModeSecure,
			wantWritten: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hermetic(t)
			ctx := t.Context()
			configPath := env.Get(ctx, "DATABRICKS_CONFIG_FILE")
			if tc.configBody != "" {
				require.NoError(t, os.WriteFile(configPath, []byte(tc.configBody), 0o600))
			}
			if tc.envValue != "" {
				ctx = env.Set(ctx, EnvVar, tc.envValue)
			}

			PinSecureMode(ctx, tc.mode, tc.override)

			got, err := databrickscfg.GetConfiguredAuthStorage(ctx, configPath)
			require.NoError(t, err)
			assert.Equal(t, tc.wantWritten, got)
		})
	}
}

func TestPinSecureMode_IsIdempotent(t *testing.T) {
	hermetic(t)
	ctx := t.Context()
	configPath := env.Get(ctx, "DATABRICKS_CONFIG_FILE")

	PinSecureMode(ctx, StorageModeSecure, StorageModeUnknown)
	first, err := databrickscfg.GetConfiguredAuthStorage(ctx, configPath)
	require.NoError(t, err)
	require.Equal(t, "secure", first)

	// Second call should see source=Config and skip the write.
	PinSecureMode(ctx, StorageModeSecure, StorageModeUnknown)
	second, err := databrickscfg.GetConfiguredAuthStorage(ctx, configPath)
	require.NoError(t, err)
	assert.Equal(t, "secure", second)
}

func TestPinSecureMode_PersistFailureIsSwallowed(t *testing.T) {
	hermetic(t)
	ctx := t.Context()
	// Point DATABRICKS_CONFIG_FILE at a path whose parent does not exist.
	// loadOrCreateConfigFile does not mkdir, so the underlying os.OpenFile
	// fails and SetConfiguredAuthStorage returns an error.
	configPath := filepath.Join(t.TempDir(), "no-such-dir", ".databrickscfg")
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)

	// Must not panic or block; failure surfaces in the warn log.
	PinSecureMode(ctx, StorageModeSecure, StorageModeUnknown)

	// The persist failure must not have produced any file.
	_, err := os.Stat(configPath)
	assert.ErrorIs(t, err, fs.ErrNotExist, "no file should have been written")
}

func TestWrapForOAuthArgument(t *testing.T) {
	const (
		host       = "https://example.com"
		profileKey = "myprofile"
	)
	arg, err := u2m.NewProfileWorkspaceOAuthArgument(host, profileKey)
	require.NoError(t, err)

	cases := []struct {
		name        string
		mode        StorageMode
		wantWrap    bool
		wantHostKey bool
	}{
		{"plaintext wraps and mirrors under host key", StorageModePlaintext, true, true},
		{"secure returns inner unchanged; no host-key mirror", StorageModeSecure, false, false},
		{"unknown returns inner unchanged; no host-key mirror", StorageModeUnknown, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inner := newMemStore()
			got := WrapForOAuthArgument(t.Context(), inner, tc.mode, arg)

			_, wrapped := got.(*DualWritingTokenCache)
			assert.Equal(t, tc.wantWrap, wrapped, "wrapper presence")

			tok := &oauth2.Token{AccessToken: "abc"}
			require.NoError(t, got.Store(profileKey, tok))

			primary, err := inner.Lookup(profileKey)
			require.NoError(t, err, "primary key must always be written")
			assert.Equal(t, tok, primary.Token)

			_, err = inner.Lookup(host)
			if tc.wantHostKey {
				require.NoError(t, err, "host-key mirror expected in plaintext mode")
			} else {
				assert.ErrorIs(t, err, ErrNotFound, "no host-key mirror expected outside plaintext mode")
			}
		})
	}
}
