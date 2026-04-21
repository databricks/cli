package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	u2m_cache "github.com/databricks/databricks-sdk-go/credentials/u2m/cache"
	"golang.org/x/oauth2"
)

const (
	// tokenCacheFile is the path of the default token cache, relative to the
	// user's home directory.
	tokenCacheFilePath = ".databricks/token-cache.json"

	// ownerExecReadWrite is the permission for the .databricks directory.
	ownerExecReadWrite = 0o700

	// ownerReadWrite is the permission for the token-cache.json file.
	ownerReadWrite = 0o600

	// tokenCacheVersion is the version of the token cache file format.
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
	tokenCacheVersion = 1
)

// tokenCacheFile is the format of the token cache file.
type tokenCacheFile struct {
	Version int                      `json:"version"`
	Tokens  map[string]*oauth2.Token `json:"tokens"`
}

type FileTokenCacheOption func(*fileTokenCache)

func WithFileLocation(fileLocation string) FileTokenCacheOption {
	return func(c *fileTokenCache) {
		c.fileLocation = fileLocation
	}
}

// fileTokenCache caches tokens in "~/.databricks/token-cache.json". fileTokenCache
// implements the TokenCache interface.
type fileTokenCache struct {
	fileLocation string

	// locker protects the token cache file from concurrent reads and writes.
	locker sync.Mutex
}

// NewFileTokenCache creates a new FileTokenCache. By default, the cache is
// stored in "~/.databricks/token-cache.json". The cache file is created if it
// does not already exist. The cache file is created with owner permissions
// 0600 and the directory is created with owner permissions 0700. If the cache
// file is corrupt or if its version does not match tokenCacheVersion, an error
// is returned.
func NewFileTokenCache(opts ...FileTokenCacheOption) (u2m_cache.TokenCache, error) {
	c := &fileTokenCache{}
	for _, opt := range opts {
		opt(c)
	}
	if err := c.init(); err != nil {
		return nil, err
	}
	// Fail fast if the cache is not working.
	if _, err := c.load(); err != nil {
		return nil, fmt.Errorf("load: %w", err)
	}
	return c, nil
}

// Store implements the TokenCache interface.
func (c *fileTokenCache) Store(key string, t *oauth2.Token) error {
	c.locker.Lock()
	defer c.locker.Unlock()
	f, err := c.load()
	if err != nil {
		return fmt.Errorf("load: %w", err)
	}
	if f.Tokens == nil {
		f.Tokens = map[string]*oauth2.Token{}
	}
	if t == nil {
		delete(f.Tokens, key)
	} else {
		f.Tokens[key] = t
	}
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := c.atomicWriteFile(raw); err != nil {
		return fmt.Errorf("error storing token in local cache: %w", err)
	}
	return nil
}

// Lookup implements the TokenCache interface.
func (c *fileTokenCache) Lookup(key string) (*oauth2.Token, error) {
	c.locker.Lock()
	defer c.locker.Unlock()
	f, err := c.load()
	if err != nil {
		return nil, fmt.Errorf("load: %w", err)
	}
	t, ok := f.Tokens[key]
	if !ok {
		return nil, u2m_cache.ErrNotFound
	}
	return t, nil
}

// init initializes the token cache file. It creates the file and directory if
// they do not already exist.
func (c *fileTokenCache) init() error {
	// set the default file location
	if c.fileLocation == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed loading home directory: %w", err)
		}
		c.fileLocation = filepath.Join(home, tokenCacheFilePath)
	}
	// Create the cache file if it does not exist.
	if _, err := os.Stat(c.fileLocation); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("stat file: %w", err)
		}
		// Create the parent directories if needed.
		if err := os.MkdirAll(filepath.Dir(c.fileLocation), ownerExecReadWrite); err != nil {
			return fmt.Errorf("mkdir: %w", err)
		}

		// Create an empty cache file.
		f := &tokenCacheFile{
			Version: tokenCacheVersion,
			Tokens:  map[string]*oauth2.Token{},
		}
		raw, err := json.MarshalIndent(f, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}
		if err := c.atomicWriteFile(raw); err != nil {
			return fmt.Errorf("error creating token cache file: %w", err)
		}
	}
	return nil
}

// load loads the token cache file from disk. If the file is corrupt or if its
// version does not match tokenCacheVersion, it returns an error.
func (c *fileTokenCache) load() (*tokenCacheFile, error) {
	raw, err := os.ReadFile(c.fileLocation)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	f := &tokenCacheFile{}
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	if f.Version != tokenCacheVersion {
		// in the later iterations we could do state upgraders,
		// so that we transform token cache from v1 to v2 without
		// losing the tokens and asking the user to re-authenticate.
		return nil, fmt.Errorf("needs version %d, got version %d", tokenCacheVersion, f.Version)
	}
	return f, nil
}

// atomicWriteFile writes data to the file atomically by first writing to a
// temporary file in the same directory and then renaming it to the target.
// This prevents corruption from interrupted writes.
func (c *fileTokenCache) atomicWriteFile(data []byte) error {
	tmp, err := c.writeTmpFile(data)
	if err != nil {
		return err
	}
	defer os.Remove(tmp)
	return os.Rename(tmp, c.fileLocation)
}

func (c *fileTokenCache) writeTmpFile(data []byte) (string, error) {
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
