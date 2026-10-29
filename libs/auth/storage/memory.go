package storage

import (
	"sync"

	"golang.org/x/oauth2"
)

// NewMemoryStore returns a Store that stores entries in process
// memory only. Tokens do not persist across process restarts. This is the
// default store used by PersistentAuth when no WithTokenStore option is
// provided. Most production consumers should supply a persistent Store.
func NewMemoryStore() Store {
	return &memoryStore{entries: map[string]Entry{}, pendingLogins: map[string]PendingLogin{}}
}

type memoryStore struct {
	mu            sync.Mutex
	entries       map[string]Entry
	pendingLogins map[string]PendingLogin
}

func cloneToken(t *oauth2.Token) *oauth2.Token {
	if t == nil {
		return nil
	}
	clone := *t
	return &clone
}

func cloneEntry(e Entry) Entry {
	e.Token = cloneToken(e.Token)
	return e
}

// Put implements Store.
func (s *memoryStore) Put(key string, e Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[key] = cloneEntry(e)
	return nil
}

// Lookup implements Store. Returns ErrNotFound when the key is absent.
func (s *memoryStore) Lookup(key string) (Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[key]
	if !ok {
		return Entry{}, ErrNotFound
	}
	return cloneEntry(e), nil
}

// Delete implements Store.
func (s *memoryStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, key)
	return nil
}

// PutPendingLogin implements PendingLoginStore.
func (s *memoryStore) PutPendingLogin(key string, p PendingLogin) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingLogins[key] = p
	return nil
}

// LookupPendingLogin implements PendingLoginStore. Returns ErrNotFound when
// the key is absent.
func (s *memoryStore) LookupPendingLogin(key string) (PendingLogin, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.pendingLogins[key]
	if !ok {
		return PendingLogin{}, ErrNotFound
	}
	return p, nil
}

// DeletePendingLogin implements PendingLoginStore.
func (s *memoryStore) DeletePendingLogin(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pendingLogins, key)
	return nil
}

// Compile-time confirmation that memoryStore keeps pending logins.
var _ PendingLoginStore = (*memoryStore)(nil)
