package auth

import (
	"context"
	"testing"

	"github.com/databricks/databricks-sdk-go/credentials/u2m"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

type fakePersistentAuth struct {
	token    *oauth2.Token
	tokenErr error
	closeErr error
}

func (f *fakePersistentAuth) Token() (*oauth2.Token, error) {
	return f.token, f.tokenErr
}

func (f *fakePersistentAuth) Close() error {
	return f.closeErr
}

func TestAcquireTokenDoesNotMutatePersistentAuthOpts(t *testing.T) {
	defaultFactory := persistentAuthFactory
	t.Cleanup(func() {
		persistentAuthFactory = defaultFactory
	})

	opts := []u2m.PersistentAuthOption{
		func(pa *u2m.PersistentAuth) {},
		func(pa *u2m.PersistentAuth) {},
	}
	initialLen := len(opts)
	factoryCalls := 0

	persistentAuthFactory = func(ctx context.Context, providedOpts ...u2m.PersistentAuthOption) (persistentAuth, error) {
		factoryCalls++
		require.Len(t, providedOpts, initialLen+1)
		require.Len(t, opts, initialLen) // original slice must not change
		return &fakePersistentAuth{
			token: &oauth2.Token{
				AccessToken: "token",
			},
		}, nil
	}

	req := AcquireTokenRequest{
		AuthArguments: &AuthArguments{
			Host:      "https://accounts.cloud.databricks.com",
			AccountID: "active",
		},
		PersistentAuthOpts: opts,
	}

	_, err := AcquireToken(context.Background(), req)
	require.NoError(t, err)
	require.Len(t, opts, initialLen)

	_, err = AcquireToken(context.Background(), req)
	require.NoError(t, err)
	require.Len(t, opts, initialLen)
	require.Equal(t, 2, factoryCalls)
}
