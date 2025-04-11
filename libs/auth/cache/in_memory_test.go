package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func TestInMemoryCacheHit(t *testing.T) {
	token := &oauth2.Token{
		AccessToken: "abc",
	}
	c := &InMemoryTokenCache{
		Tokens: map[string]*oauth2.Token{
			"key": token,
		},
	}
	res, err := c.Lookup("key")
	assert.Equal(t, res, token)
	assert.NoError(t, err)
}

func TestInMemoryCacheMiss(t *testing.T) {
	c := &InMemoryTokenCache{
		Tokens: map[string]*oauth2.Token{},
	}
	_, err := c.Lookup("key")
	assert.ErrorIs(t, err, ErrNotConfigured)
}

func TestInMemoryCacheStore(t *testing.T) {
	token := &oauth2.Token{
		AccessToken: "abc",
	}
	c := &InMemoryTokenCache{
		Tokens: map[string]*oauth2.Token{},
	}
	err := c.Store("key", token)
	assert.NoError(t, err)
	res, err := c.Lookup("key")
	assert.Equal(t, res, token)
	assert.NoError(t, err)
}

func TestInMemoryDeleteKey(t *testing.T) {
	c := &InMemoryTokenCache{
		Tokens: map[string]*oauth2.Token{},
	}
	err := c.Store("x", &oauth2.Token{
		AccessToken: "abc",
	})
	require.NoError(t, err)

	err = c.Store("y", &oauth2.Token{
		AccessToken: "bcd",
	})
	require.NoError(t, err)

	err = c.Delete("x")
	require.NoError(t, err)
	assert.Equal(t, 1, len(c.Tokens))

	_, err = c.Lookup("x")
	assert.Equal(t, ErrNotConfigured, err)

	tok, err := c.Lookup("y")
	require.NoError(t, err)
	assert.Equal(t, "bcd", tok.AccessToken)
}

func TestInMemoryDeleteKeyNotExist(t *testing.T) {
	c := &InMemoryTokenCache{
		Tokens: map[string]*oauth2.Token{},
	}
    err := c.Delete("x")
	assert.Equal(t, ErrNotConfigured, err)

	_, err = c.Lookup("x")
	assert.Equal(t, ErrNotConfigured, err)
}
