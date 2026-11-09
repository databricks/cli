package u2m_test

import (
	"crypto/sha256"
	"errors"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/cli/libs/auth/u2m"
	"golang.org/x/oauth2"
)

func TestFileRefreshIntentPreservesFallbackAndGuardsForce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tokens.json")
	store, err := storage.NewFileStore(t.Context(), storage.WithFileLocation(path))
	if err != nil {
		t.Fatal("store setup failed")
	}
	arg, err := u2m.NewBasicAccountOAuthArgument("https://accounts.databricks.test", "account")
	if err != nil {
		t.Fatal("argument setup failed")
	}
	old := storage.Entry{Token: &oauth2.Token{AccessToken: "generation-0", RefreshToken: "generation-0", Expiry: time.Now().Add(time.Minute)}}
	if err := store.Put(arg.GetCacheKey(), old); err != nil {
		t.Fatal("seed failed")
	}
	_, err = storage.WithFileTransaction(t.Context(), store, func(view storage.Store) error {
		if _, err := storage.BeginFileRefresh(t.Context(), view, arg.GetCacheKey(), sha256.Sum256([]byte("route"))); err != nil {
			return err
		}
		return errors.New("closed unknown outcome")
	}, nil)
	if err == nil {
		t.Fatal("prepared setup failed")
	}
	probe := &fileCommitFailureTransport{path: path}
	auth, err := u2m.NewPersistentAuth(t.Context(), u2m.WithTokenStore(store), u2m.WithOAuthArgument(arg), u2m.WithOAuthEndpointSupplier(&u2m.BasicOAuthEndpointSupplier{}), u2m.WithHttpClient(&http.Client{Transport: probe}))
	if err != nil {
		t.Fatal("auth setup failed")
	}
	t.Cleanup(func() { _ = auth.Close() })
	token, err := auth.Token()
	if err != nil || token == nil || tokenGeneration(token.AccessToken) != 0 || token.RefreshToken != "" {
		t.Fatal("still-valid fallback changed")
	}
	if _, err := auth.ForceRefreshToken(); !errors.Is(err, storage.ErrRefreshRecoveryRequired) {
		t.Fatal("force bypassed unknown generation")
	}
	old.Token.Expiry = time.Now().Add(-time.Hour)
	if err := store.Put(arg.GetCacheKey(), old); err != nil {
		t.Fatal("expired setup failed")
	}
	if _, err := auth.Token(); !errors.Is(err, storage.ErrRefreshRecoveryRequired) {
		t.Fatal("expired token did not fail closed")
	}
	if len(probe.calls) != 0 {
		t.Fatal("blocked credential reached provider")
	}
}

func TestFileRefreshIntentRepeatedDenialKeepsMappingAndNoReplay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tokens.json")
	store, err := storage.NewFileStore(t.Context(), storage.WithFileLocation(path))
	if err != nil {
		t.Fatal("store setup failed")
	}
	arg, err := u2m.NewBasicAccountOAuthArgument("https://accounts.databricks.test", "account")
	if err != nil {
		t.Fatal("argument setup failed")
	}
	if err := store.Put(arg.GetCacheKey(), storage.Entry{Token: &oauth2.Token{AccessToken: "generation-0", RefreshToken: "generation-0", Expiry: time.Now().Add(-time.Hour)}}); err != nil {
		t.Fatal("seed failed")
	}
	probe := &fileCommitFailureTransport{path: path, spent: true}
	fresh := func() *u2m.PersistentAuth {
		current, err := storage.NewFileStore(t.Context(), storage.WithFileLocation(path))
		if err != nil {
			t.Fatal("fresh store failed")
		}
		auth, err := u2m.NewPersistentAuth(t.Context(), u2m.WithTokenStore(current), u2m.WithOAuthArgument(arg), u2m.WithOAuthEndpointSupplier(&u2m.BasicOAuthEndpointSupplier{}), u2m.WithHttpClient(&http.Client{Transport: probe}))
		if err != nil {
			t.Fatal("fresh auth failed")
		}
		t.Cleanup(func() { _ = auth.Close() })
		return auth
	}
	if _, err := fresh().ForceRefreshToken(); err == nil {
		t.Fatal("initiating provider denial was lost")
	} else if _, ok := errors.AsType[*u2m.InvalidRefreshTokenError](err); !ok {
		t.Fatal("initiating provider denial type changed")
	}
	if _, err := fresh().ForceRefreshToken(); !errors.Is(err, storage.ErrRefreshRecoveryRequired) {
		t.Fatal("later client did not retain guarded state")
	}
	if len(probe.calls) != 1 || probe.calls[0] != 0 {
		t.Fatal("repeated denial replayed cached credential")
	}
}
