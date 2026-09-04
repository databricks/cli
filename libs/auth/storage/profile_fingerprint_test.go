package storage

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// TestProfileFingerprintStoreLookup verifies that matching entries can be read while
// changed and legacy entries are rejected with the appropriate mismatch details.
func TestProfileFingerprintStoreLookup(t *testing.T) {
	currentFingerprint := "current"

	tests := []struct {
		name              string
		storedFingerprint string
		wantMissing       bool
		wantErr           bool
	}{
		{
			name:              "matching fingerprint",
			storedFingerprint: currentFingerprint,
		},
		{
			name:              "changed fingerprint",
			storedFingerprint: "old",
			wantErr:           true,
		},
		{
			name:        "missing legacy fingerprint",
			wantMissing: true,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inner := newMemStore()
			store := NewProfileFingerprintStore(inner, "TEST", currentFingerprint)
			require.NoError(t, inner.Put("TEST", Entry{
				Token:              &oauth2.Token{AccessToken: "token"},
				ProfileFingerprint: tt.storedFingerprint,
			}))

			got, err := store.Lookup("TEST")
			if !tt.wantErr {
				require.NoError(t, err)
				assert.Equal(t, "token", got.Token.AccessToken)
				return
			}

			assert.ErrorIs(t, err, ErrProfileChanged)
			changedErr, ok := errors.AsType[*ProfileFingerprintError](err)
			require.True(t, ok)
			assert.Equal(t, tt.wantMissing, changedErr.Missing)
		})
	}
}

// TestProfileFingerprintStoreStampsWrites verifies that replacement token writes retain
// the fingerprint binding, as required when OAuth refresh replaces a cache entry.
func TestProfileFingerprintStoreStampsWrites(t *testing.T) {
	inner := newMemStore()
	store := NewProfileFingerprintStore(inner, "TEST", "current")

	require.NoError(t, store.Put("TEST", Entry{Token: &oauth2.Token{AccessToken: "token"}}))
	entry, err := inner.Lookup("TEST")
	require.NoError(t, err)
	assert.Equal(t, "current", entry.ProfileFingerprint)
}

// TestSetProfileFingerprintOnlyUpdatesProfileKey verifies that login binds the
// profile-keyed token without binding the shared legacy host-keyed copy.
func TestSetProfileFingerprintOnlyUpdatesProfileKey(t *testing.T) {
	inner := newMemStore()

	require.NoError(t, inner.Put("TEST", Entry{Token: &oauth2.Token{AccessToken: "token"}}))
	require.NoError(t, inner.Put("https://workspace.example.com", Entry{Token: &oauth2.Token{AccessToken: "token"}}))
	require.NoError(t, SetProfileFingerprint(inner, "TEST", "current"))

	entry, err := inner.Lookup("TEST")
	require.NoError(t, err)

	assert.Equal(t, "current", entry.ProfileFingerprint)

	// A host can be shared by multiple profiles, so its compatibility copy is
	// not bound to any one profile.
	hostEntry, err := inner.Lookup("https://workspace.example.com")
	require.NoError(t, err)

	assert.Empty(t, hostEntry.ProfileFingerprint)
}
