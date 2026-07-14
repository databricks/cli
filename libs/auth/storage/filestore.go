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
	c.locker.Lock()
	defer c.locker.Unlock()
	f, err := c.load()
	if err != nil {
		return fmt.Errorf("load: %w", err)
	}
	if f.Tokens == nil {
		f.Tokens = map[string]*fileEntry{}
	}
	f.Tokens[key] = &fileEntry{Token: e.Token}
	return c.write(f)
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
	if !ok {
		return Entry{}, ErrNotFound
	}
	return Entry{Token: fe.Token}, nil
}

// Delete implements the Store interface. Removing a missing key is a no-op.
func (c *fileStore) Delete(key string) error {
	c.locker.Lock()
	defer c.locker.Unlock()
	f, err := c.load()
	if err != nil {
		return fmt.Errorf("load: %w", err)
	}
	delete(f.Tokens, key)
	return c.write(f)
}

// write marshals f and atomically replaces the store file.
func (c *fileStore) write(f *tokenStoreFile) error {
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := c.atomicWriteFile(raw); err != nil {
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
	// Create the store file if it does not exist.
	if _, err := os.Stat(c.fileLocation); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("stat file: %w", err)
		}
		// Create the parent directories if needed.
		if err := os.MkdirAll(filepath.Dir(c.fileLocation), ownerExecReadWrite); err != nil {
			return fmt.Errorf("mkdir: %w", err)
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
		if err := c.atomicWriteFile(raw); err != nil {
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

// atomicWriteFile writes data to the file atomically by first writing to a
// temporary file in the same directory and then renaming it to the target.
// This prevents corruption from interrupted writes.
func (c *fileStore) atomicWriteFile(data []byte) error {
	tmp, err := c.writeTmpFile(data)
	if err != nil {
		return err
	}
	defer os.Remove(tmp)
	return os.Rename(tmp, c.fileLocation)
}

func (c *fileStore) writeTmpFile(data []byte) (string, error) {
	tmp, err := os.CreateTemp(filepath.Dir(c.fileLocation), ".token-cache-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer tmp.Close()

	if _, err := tmp.Write(data); err != nil {
		return "", err
	}
	if err := tmp.Chmod(ownerReadWrite); err != nil {
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	return tmp.Name(), nil
}
