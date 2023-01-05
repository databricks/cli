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
	tokenCacheChmod = 0o700

	// format versioning leaves some room for format improvement
	tokenCacheVersion = 1
)

var ErrNotConfigured = errors.New("databricks OAuth is not configured for this host")

// this implementation requires the calling code to do a machine-wide lock,
// otherwise the file might get corrupt.
type TokenCache struct {
	Version int                      `json:"version"`
	Tokens  map[string]*oauth2.Token `json:"tokens"`

	fileLocation string `json:"-"`
}

func (c *TokenCache) Store(key string, t *oauth2.Token) error {
	err := c.load()
	if errors.Is(err, fs.ErrNotExist) {
		dir := filepath.Dir(c.fileLocation)
		err = os.MkdirAll(dir, tokenCacheChmod)
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
	return os.WriteFile(c.fileLocation, raw, tokenCacheChmod)
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
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home: %w", err)
	}
	// we can also store all cached credentials in one single file,
	// like ~/.azure/msal_token_cache.json for az login - in our case it'll be
	// something like ~/.databricks/token-cache.json
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
		// loosing the tokens and asking the user to re-authenticate.
		return fmt.Errorf("needs version %d, got version %d",
			tokenCacheVersion, c.Version)
	}
	return nil
}
