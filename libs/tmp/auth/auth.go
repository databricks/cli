// Package auth is an internal package that provides authentication utilities.
//
// IMPORTANT: This package is not meant to be used directly by consumers of the
// SDK and is subject to change without notice.
package auth

import (
	"context"
	"fmt"
	"time"
)

// Header represents a header that can be used to sign requests.
type Header struct {
	Key   string
	Value string
}

// Credentials represents a named source of authentication headers.
type Credentials interface {
	Name() string

	AuthHeaders(context.Context) ([]Header, error)
}

// Token represents a token that can be used to sign requests.
type Token struct {
	// Value is the raw value to sign requests with. It typically is an
	// access token but can represent other types of tokens (e.g. ID tokens).
	Value string

	// Type is the type of token. If Type is empty, the token type is
	// assumed to be "Bearer".
	Type string

	// Expiry is the time at which the token expires. If Expiry is zero, the
	// token is considered to be valid indefinitely.
	Expiry time.Time
}

// A TokenProvider is anything that can return a token.
type TokenProvider interface {
	// Token returns a token or an error. The returned Token should be
	// considered immutable and should not be modified.
	Token(context.Context) (*Token, error)
}

// TokenProviderFn is an adapter to allow the use of ordinary functions as
// TokenProvider.
//
// Example:
//
//	   ts := TokenProviderFn(func(ctx context.Context) (*Token, error) {
//			return &Token{}, nil
//	   })
type TokenProviderFn func(context.Context) (*Token, error)

func (fn TokenProviderFn) Token(ctx context.Context) (*Token, error) {
	return fn(ctx)
}

type TokenCredentials interface {
	TokenProvider
	Credentials
}

// NewTokenCredentials returns TokenCredentials with the given name that use the
// given TokenProvider to return authentication headers.
func NewTokenCredentials(name string, p TokenProvider) TokenCredentials {
	return &tokenCredentials{name: name, p: p}
}

type tokenCredentials struct {
	name string
	p    TokenProvider
}

func (c *tokenCredentials) Name() string {
	return c.name
}

func (c *tokenCredentials) AuthHeaders(ctx context.Context) ([]Header, error) {
	token, err := c.p.Token(ctx)
	if err != nil {
		return nil, err
	}
	tt := token.Type
	if tt == "" {
		tt = "Bearer"
	}
	return []Header{
		{Key: "Authorization", Value: fmt.Sprintf("%s %s", tt, token.Value)},
	}, nil
}

func (c *tokenCredentials) Token(ctx context.Context) (*Token, error) {
	return c.p.Token(ctx)
}
