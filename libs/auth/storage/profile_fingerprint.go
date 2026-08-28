package storage

import (
	"fmt"
)

// ProfileFingerprintError tells the user why an otherwise valid cached token
// cannot be reused after the corresponding profile changed.
type ProfileFingerprintError struct {
	Profile string
	Missing bool
}

func (e *ProfileFingerprintError) Error() string {
	if e.Missing {
		return fmt.Sprintf("cached credentials for profile %q predate profile change detection; run `databricks auth login --profile %q` to sign in again", e.Profile, e.Profile)
	}
	return fmt.Sprintf("profile %q has changed since the last login; run `databricks auth login --profile %q` to sign in again", e.Profile, e.Profile)
}

func (e *ProfileFingerprintError) Unwrap() error {
	return ErrProfileChanged
}

// ProfileFingerprintStore stamps token writes and rejects reads whose metadata
// does not match the current profile.
type ProfileFingerprintStore struct {
	inner       Store
	profile     string
	fingerprint string
}

// NewProfileFingerprintStore binds token reads and refresh writes to fingerprint.
func NewProfileFingerprintStore(inner Store, profile, fingerprint string) *ProfileFingerprintStore {
	return &ProfileFingerprintStore{
		inner:       inner,
		profile:     profile,
		fingerprint: fingerprint,
	}
}

func (s *ProfileFingerprintStore) Put(key string, entry Entry) error {
	// The SDK replaces the entire token entry after a refresh, so stamp the
	// binding again instead of losing it with the old access token.
	entry.ProfileFingerprint = s.fingerprint

	return s.inner.Put(key, entry)
}

func (s *ProfileFingerprintStore) Lookup(key string) (Entry, error) {
	entry, err := s.inner.Lookup(key)
	if err != nil {
		return Entry{}, err
	}

	if entry.ProfileFingerprint == "" {
		return Entry{}, &ProfileFingerprintError{Profile: s.profile, Missing: true}
	}

	if entry.ProfileFingerprint != s.fingerprint {
		return Entry{}, &ProfileFingerprintError{Profile: s.profile}
	}

	return entry, nil
}

func (s *ProfileFingerprintStore) Delete(key string) error {
	return s.inner.Delete(key)
}

// SetProfileFingerprint binds an existing profile-keyed token to the profile
// left on disk after login. Legacy host-key copies are intentionally excluded
// because one host can be shared by multiple profiles.
func SetProfileFingerprint(store Store, profile, fingerprint string) error {
	entry, err := store.Lookup(profile)
	if err != nil {
		return fmt.Errorf("load token %q: %w", profile, err)
	}

	entry.ProfileFingerprint = fingerprint

	if err := store.Put(profile, entry); err != nil {
		return fmt.Errorf("update token %q: %w", profile, err)
	}

	return nil
}

var _ Store = (*ProfileFingerprintStore)(nil)
