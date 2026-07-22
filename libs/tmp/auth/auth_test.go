package auth

import (
	"context"
	"testing"
)

func TestNewTokenCredentials_name(t *testing.T) {
	credentials := NewTokenCredentials("test-strategy", TokenProviderFn(func(context.Context) (*Token, error) {
		return &Token{Value: "token"}, nil
	}))

	if got, want := credentials.Name(), "test-strategy"; got != want {
		t.Errorf("Name() = %q, want %q", got, want)
	}
}
