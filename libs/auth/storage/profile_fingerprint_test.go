package storage

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// TestProfileFingerprintStoreLookup accepts an entry with the current fingerprint.
func TestProfileFingerprintStoreLookup(t *testing.T) {
	inner := newMemStore()
	store := NewProfileFingerprintStore(inner, "TEST", "current")
	require.NoError(t, inner.Put("TEST", Entry{
		Token:              &oauth2.Token{AccessToken: "token"},
		ProfileFingerprint: "current",
	}))

	got, err := store.Lookup("TEST")
	require.NoError(t, err)
	assert.Equal(t, "token", got.Token.AccessToken)
}

// TestProfileFingerprintStoreRejectsInvalidFingerprint verifies the distinct
// errors returned for changed and legacy cache entries.
func TestProfileFingerprintStoreRejectsInvalidFingerprint(t *testing.T) {
	currentFingerprint := "current"

	tests := []struct {
		name              string
		storedFingerprint string
		wantMissing       bool
	}{
		{
			name:              "changed fingerprint",
			storedFingerprint: "old",
		},
		{
			name:        "missing legacy fingerprint",
			wantMissing: true,
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

			_, err := store.Lookup("TEST")

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
	assert.Equal(t, "current", inner.entries["TEST"].ProfileFingerprint)
}

// TestSetProfileFingerprintOnlyUpdatesProfileKey verifies that login binds the
// profile-keyed token without binding the shared legacy host-keyed copy.
func TestSetProfileFingerprintOnlyUpdatesProfileKey(t *testing.T) {
	inner := newMemStore()

	require.NoError(t, inner.Put("TEST", Entry{Token: &oauth2.Token{AccessToken: "token"}}))
	require.NoError(t, inner.Put("https://workspace.example.com", Entry{Token: &oauth2.Token{AccessToken: "token"}}))
	require.NoError(t, SetProfileFingerprint(inner, "TEST", "current"))

	assert.Equal(t, "current", inner.entries["TEST"].ProfileFingerprint)

	// A host can be shared by multiple profiles, so its compatibility copy is
	// not bound to any one profile.
	assert.Empty(t, inner.entries["https://workspace.example.com"].ProfileFingerprint)
}
