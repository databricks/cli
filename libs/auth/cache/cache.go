package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
)

const (
	// where the token cache is stored
	tokenCacheFile = ".databricks/token-cache.json"

	// only the owner of the file has full execute, read, and write access
	ownerExecReadWrite = 0o700

	// only the owner of the file has full read and write access
	ownerReadWrite = 0o600

	// format versioning leaves some room for format improvement
	tokenCacheVersion = 1
)

var ErrNotConfigured = errors.New("databricks OAuth is not configured for this host")

// this implementation requires the calling code to do a machine-wide lock,
// otherwise the file might get corrupt.
type TokenCache struct {
	Version int                      `json:"version"`
	Tokens  map[string]*oauth2.Token `json:"tokens"`

	fileLocation string
}

func (c *TokenCache) Store(key string, t *oauth2.Token) error {
	err := c.load()
	if errors.Is(err, fs.ErrNotExist) {
		dir := filepath.Dir(c.fileLocation)
		err = os.MkdirAll(dir, ownerExecReadWrite)
		if err != nil {
			return fmt.Errorf("mkdir: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("load: %w", err)
	}
	c.Version = tokenCacheVersion
	if c.Tokens == nil {
		c.Tokens = map[string]*oauth2.Token{}
	}
	c.Tokens[key] = t
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return os.WriteFile(c.fileLocation, raw, ownerReadWrite)
}

func (c *TokenCache) Lookup(key string) (*oauth2.Token, error) {
	err := c.load()
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNotConfigured
	} else if err != nil {
		return nil, fmt.Errorf("load: %w", err)
	}
	t, ok := c.Tokens[key]
	if !ok {
		return nil, ErrNotConfigured
	}
	return t, nil
}

func (c *TokenCache) location() (string, error) {
	// Allow users to override the location of the token cache file
	tokenCacheDirName, exists := os.LookupEnv("DATABRICKS_TOKEN_CACHE_DIR")
	if exists {
		return filepath.Join(tokenCacheDirName, tokenCacheFile), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home: %w", err)
	}
	return filepath.Join(home, tokenCacheFile), nil
}

func (c *TokenCache) load() error {
	loc, err := c.location()
	if err != nil {
		return err
	}
	c.fileLocation = loc
	raw, err := os.ReadFile(loc)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	err = json.Unmarshal(raw, c)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}
	if c.Version != tokenCacheVersion {
		// in the later iterations we could do state upgraders,
		// so that we transform token cache from v1 to v2 without
		// losing the tokens and asking the user to re-authenticate.
		return fmt.Errorf("needs version %d, got version %d",
			tokenCacheVersion, c.Version)
	}
	return nil
}
