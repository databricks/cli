package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
