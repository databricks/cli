package storage

import "context"

type fileTransactionStore interface {
	WithFileTransaction(context.Context, func(Store) error, func(Store) error) (bool, error)
}

// SupportsFileTransactions reports whether a known wrapper owns a file cache.
func SupportsFileTransactions(store Store) bool {
	switch store := store.(type) {
	case *fileStore:
		return true
	case *DualWritingStore:
		return SupportsFileTransactions(store.inner)
	case *notFoundHintStore:
		return SupportsFileTransactions(store.inner)
	default:
		return false
	}
}

// WithFileTransaction stages file-backed mutations and publishes them atomically.
// A false result leaves callbacks uncalled: other backends retain their behavior.
// Cleanup runs under the same lease only when a successful callback cannot commit.
func WithFileTransaction(ctx context.Context, store Store, update, cleanup func(Store) error) (bool, error) {
	owner, ok := store.(fileTransactionStore)
	if !ok {
		return false, nil
	}
	return owner.WithFileTransaction(ctx, update, cleanup)
}

// PutWithContext preserves the caller's context for file-backed mutations.
func PutWithContext(ctx context.Context, store Store, key string, entry Entry) error {
	handled, err := WithFileTransaction(ctx, store, func(view Store) error { return view.Put(key, entry) }, nil)
	if handled {
		return err
	}
	return store.Put(key, entry)
}

// DeleteWithContext preserves the caller's context for file-backed mutations.
func DeleteWithContext(ctx context.Context, store Store, key string) error {
	handled, err := WithFileTransaction(ctx, store, func(view Store) error { return view.Delete(key) }, nil)
	if handled {
		return err
	}
	return store.Delete(key)
}

type fileTransaction struct {
	file              *tokenStoreFile
	dirty             bool
	owner             *fileStore
	ctx               context.Context
	original          *tokenStoreFile
	originalRaw       []byte
	journal           *refreshJournal
	touched           map[string]bool
	live              bool
	attempt           *FileRefreshAttempt
	beforeBindings    map[string]*refreshBinding
	beforeGenerations map[string]*refreshGeneration
}

func (t *fileTransaction) Lookup(key string) (Entry, error) {
	if !t.live {
		return Entry{}, intentError(errRefreshOperation, nil)
	}
	entry, ok := t.file.Tokens[key]
	if !ok || entry == nil {
		return Entry{}, ErrNotFound
	}
	return cloneEntry(Entry{Token: entry.Token, PendingLogin: entry.PendingLogin}), nil
}

func (t *fileTransaction) Put(key string, entry Entry) error {
	if !t.live {
		return intentError(errRefreshOperation, nil)
	}
	t.touched[key] = true
	entry = cloneEntry(entry)
	if t.file.Tokens == nil {
		t.file.Tokens = make(map[string]*fileEntry)
	}
	t.file.Tokens[key] = &fileEntry{Token: entry.Token, PendingLogin: entry.PendingLogin}
	t.dirty = true
	return nil
}

func (t *fileTransaction) Delete(key string) error {
	if !t.live {
		return intentError(errRefreshOperation, nil)
	}
	t.touched[key] = true
	if _, ok := t.file.Tokens[key]; ok {
		delete(t.file.Tokens, key)
		t.dirty = true
	}
	return nil
}

func cloneStoreFile(file *tokenStoreFile) *tokenStoreFile {
	clone := &tokenStoreFile{Version: file.Version, Tokens: make(map[string]*fileEntry, len(file.Tokens))}
	for key, entry := range file.Tokens {
		// Preserve JSON null entries accepted by the cache loader.
		if entry == nil {
			clone.Tokens[key] = nil
			continue
		}
		value := cloneEntry(Entry{Token: entry.Token, PendingLogin: entry.PendingLogin})
		clone.Tokens[key] = &fileEntry{Token: value.Token, PendingLogin: value.PendingLogin}
	}
	return clone
}
