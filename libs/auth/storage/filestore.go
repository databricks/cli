package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/databricks/cli/libs/atomicfile"
	"github.com/databricks/cli/libs/env"
	"golang.org/x/oauth2"
)

const (
	// tokenStoreFilePath is the path of the default token store, relative to
	// the user's home directory. The on-disk filename stays "token-cache.json"
	// for backward compatibility with tokens written by older CLI versions,
	// even though the Go identifiers now use the "store" vocabulary.
	tokenStoreFilePath = ".databricks/token-cache.json"

	// ownerExecReadWrite is the permission for the .databricks directory.
	ownerExecReadWrite = 0o700

	// ownerReadWrite is the permission for the token-cache.json file.
	ownerReadWrite = 0o600

	// tokenStoreVersion is the version of the token store file format.
	//
	// Version 1 format:
	//
	// {
	//   "version": 1,
	//   "tokens": {
	//     "<key>": {
	//       "access_token": "<access_token>",
	//       "token_type": "<token_type>",
	//       "refresh_token": "<refresh_token>",
	//       "expiry": "<expiry>"
	//     }
	//   }
	// }
	tokenStoreVersion = 1
)

// fileEntry is the per-key on-disk shape. The embedded *oauth2.Token promotes
// the token fields to the top level of the entry object so the layout matches
// the historical bare-token format, leaving room for additive sibling fields.
type fileEntry struct {
	*oauth2.Token
	PendingLogin *PendingLogin `json:"pending_login,omitempty"`
}

// tokenStoreFile is the format of the token store file.
type tokenStoreFile struct {
	Version int                   `json:"version"`
	Tokens  map[string]*fileEntry `json:"tokens"`
}

type FileStoreOption func(*fileStore)

func WithFileLocation(fileLocation string) FileStoreOption {
	return func(c *fileStore) {
		c.fileLocation = fileLocation
	}
}

// fileStore stores tokens in "~/.databricks/token-cache.json". fileStore
// implements the Store interface.
type fileStore struct {
	fileLocation string

	// locker protects the token store file from concurrent reads and writes.
	locker sync.Mutex

	beforeIntentCompletion func()
}

// NewFileStore creates a new file-backed Store. By default, tokens are stored
// in "~/.databricks/token-cache.json". The store file is created if it does
// not already exist, with owner permissions 0600 and the directory with owner
// permissions 0700. If the store file is corrupt or if its version does not
// match tokenStoreVersion, an error is returned.
func NewFileStore(ctx context.Context, opts ...FileStoreOption) (Store, error) {
	c := &fileStore{}
	for _, opt := range opts {
		opt(c)
	}
	if err := c.init(ctx); err != nil {
		return nil, err
	}
	// Fail fast if the store is not working.
	if _, err := c.load(); err != nil {
		return nil, fmt.Errorf("load: %w", err)
	}
	return c, nil
}

// Put implements the Store interface.
func (c *fileStore) Put(key string, e Entry) error {
	return PutWithContext(context.Background(), c, key, e) //nolint:gocritic // Legacy Store interface has no operation context.
}

// Lookup implements the Store interface.
func (c *fileStore) Lookup(key string) (Entry, error) {
	c.locker.Lock()
	defer c.locker.Unlock()
	f, err := c.load()
	if err != nil {
		return Entry{}, fmt.Errorf("load: %w", err)
	}
	fe, ok := f.Tokens[key]
	if !ok || fe == nil {
		return Entry{}, ErrNotFound
	}
	return Entry{Token: fe.Token, PendingLogin: fe.PendingLogin}, nil
}

// Delete implements the Store interface. Removing a missing key is a no-op.
func (c *fileStore) Delete(key string) error {
	return DeleteWithContext(context.Background(), c, key) //nolint:gocritic // Legacy Store interface has no operation context.
}

// WithFileTransaction publishes one staged map while holding the cache lease.
func (c *fileStore) WithFileTransaction(ctx context.Context, update, cleanup func(Store) error) (bool, error) {
	unlock, err := lockTokenStore(ctx, c.fileLocation+".lock")
	if err != nil {
		return true, fmt.Errorf("token store lock: %w", err)
	}
	defer unlock()
	file, err := c.load()
	if err != nil {
		return true, fmt.Errorf("load: %w", err)
	}
	raw, err := os.ReadFile(c.fileLocation)
	if err != nil {
		return true, fmt.Errorf("load: %w", err)
	}
	journal, err := c.loadRefreshJournal(ctx)
	if err != nil {
		return true, err
	}
	if err := c.reconcileRefreshJournal(ctx, journal, raw); err != nil {
		return true, err
	}
	staged := newFileTransaction(ctx, c, file, raw, journal)
	defer func() { staged.live = false }()
	if err := update(staged); err != nil {
		return true, err
	}
	staged.live = false
	return true, c.publishFileTransaction(staged, cleanup)
}

// write marshals f and atomically replaces the store file.
func encodeTokenStoreFile(file *tokenStoreFile) ([]byte, error) {
	raw, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	return raw, nil
}

func (c *fileStore) write(file *tokenStoreFile) error {
	raw, err := encodeTokenStoreFile(file)
	if err != nil {
		return err
	}
	return c.writeTokenStoreBytes(raw)
}

func (c *fileStore) writeTokenStoreBytes(raw []byte) error {
	if err := atomicfile.Write(c.fileLocation, raw, ownerReadWrite); err != nil {
		return fmt.Errorf("error storing token in local cache: %w", err)
	}
	return nil
}

// init initializes the token store file. It creates the file and directory if
// they do not already exist.
func (c *fileStore) init(ctx context.Context) error {
	// set the default file location
	if c.fileLocation == "" {
		home, err := env.UserHomeDir(ctx)
		if err != nil {
			return fmt.Errorf("failed loading home directory: %w", err)
		}
		c.fileLocation = filepath.Join(home, tokenStoreFilePath)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	location, err := filepath.Abs(c.fileLocation)
	if err != nil {
		return fmt.Errorf("resolve file location: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(location), ownerExecReadWrite); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(location))
	if err != nil {
		return fmt.Errorf("resolve store directory: %w", err)
	}
	c.fileLocation = filepath.Join(parent, filepath.Base(location))
	if target, err := filepath.EvalSymlinks(c.fileLocation); err == nil {
		c.fileLocation = target
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("resolve store file: %w", err)
	}
	// Create the store file if it does not exist.
	if _, err := os.Stat(c.fileLocation); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("stat file: %w", err)
		}
		unlock, err := lockTokenStore(ctx, c.fileLocation+".lock")
		if err != nil {
			return fmt.Errorf("token store lock: %w", err)
		}
		defer unlock()
		if _, err := os.Stat(c.fileLocation); err == nil {
			return nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("stat file: %w", err)
		}

		// Create an empty store file.
		f := &tokenStoreFile{
			Version: tokenStoreVersion,
			Tokens:  map[string]*fileEntry{},
		}
		raw, err := json.MarshalIndent(f, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}
		if err := atomicfile.Write(c.fileLocation, raw, ownerReadWrite); err != nil {
			return fmt.Errorf("error creating token store file: %w", err)
		}
	}
	return nil
}

// load loads the token store file from disk. If the file is corrupt or if its
// version does not match tokenStoreVersion, it returns an error.
func (c *fileStore) load() (*tokenStoreFile, error) {
	raw, err := os.ReadFile(c.fileLocation)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	f := &tokenStoreFile{}
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	if f.Version != tokenStoreVersion {
		// in the later iterations we could do state upgraders,
		// so that we transform token store from v1 to v2 without
		// losing the tokens and asking the user to re-authenticate.
		return nil, fmt.Errorf("needs version %d, got version %d", tokenStoreVersion, f.Version)
	}
	return f, nil
}
