package storage

import (
	"context"
	"sync"

	"golang.org/x/oauth2"
)

// NewMemoryStore returns a Store that stores entries in process
// memory only. Tokens do not persist across process restarts. This is the
// default store used by PersistentAuth when no WithTokenStore option is
// provided. Most production consumers should supply a persistent Store.
func NewMemoryStore() Store {
	return &memoryStore{entries: map[string]Entry{}}
}

type memoryStore struct {
	mu      sync.Mutex
	entries map[string]Entry
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

// Put implements LockedStore.
func (s *memoryStore) Put(key string, e Entry) error {
	s.entries[key] = cloneEntry(e)
	return nil
}

// Delete implements LockedStore.
func (s *memoryStore) Delete(key string) error {
	delete(s.entries, key)
	return nil
}

// Lookup implements Store.
func (s *memoryStore) Lookup(key string) (Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.LookupLocked(key)
}

func (s *memoryStore) LookupLocked(key string) (Entry, error) {
	e, ok := s.entries[key]
	if !ok {
		return Entry{}, ErrNotFound
	}
	return cloneEntry(e), nil
}

// Lookup implements LockedStore.
func (s *memoryLockedStore) Lookup(key string) (Entry, error) {
	return s.store.LookupLocked(key)
}

// WithLock implements Store.
func (s *memoryStore) WithLock(_ context.Context, fn func(LockedStore) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fn(&memoryLockedStore{store: s})
}

type memoryLockedStore struct {
	store *memoryStore
}

// Put implements LockedStore.
func (s *memoryLockedStore) Put(key string, e Entry) error {
	return s.store.Put(key, e)
}

// Delete implements LockedStore.
func (s *memoryLockedStore) Delete(key string) error {
	return s.store.Delete(key)
}

var _ Store = (*memoryStore)(nil)
