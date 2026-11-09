package auth

import (
	"crypto/sha256"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/cli/libs/auth/u2m"
	"golang.org/x/oauth2"
)

func TestStoreLoginAccessOnlyReplacesJournalBoundGeneration(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "publish", true: "failed-commit-cleanup"}[fail], func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "tokens.json")
			store, err := storage.NewFileStore(t.Context(), storage.WithFileLocation(path))
			if err != nil {
				t.Fatal("store setup failed")
			}
			const host = "https://workspace.test"
			old := storage.Entry{Token: &oauth2.Token{AccessToken: "old", RefreshToken: "synthetic-refresh", Expiry: time.Now().Add(time.Hour)}}
			for _, key := range []string{"profile", host, "copy"} {
				if err := store.Put(key, old); err != nil {
					t.Fatal("seed failed")
				}
			}
			_, err = storage.WithFileTransaction(t.Context(), store, func(view storage.Store) error {
				if _, err := storage.BeginFileRefresh(t.Context(), view, "profile", sha256.Sum256([]byte("route"))); err != nil {
					return err
				}
				return errors.New("closed unknown outcome")
			}, nil)
			if err == nil {
				t.Fatal("prepared setup failed")
			}
			arg, err := u2m.NewProfileWorkspaceOAuthArgument(host, "profile")
			if err != nil {
				t.Fatal("argument setup failed")
			}
			subject := store
			if fail {
				subject = obstructFileCommitStore{Store: store, path: path}
			}
			err = storeLoginToken(t.Context(), subject, storage.StorageModePlaintext, arg, &oauth2.Token{AccessToken: "access-only"})
			if (err != nil) != fail {
				t.Fatal("access-only publication outcome changed")
			}
			for _, key := range []string{"profile", host} {
				entry, lookupErr := store.Lookup(key)
				if fail {
					if !errors.Is(lookupErr, storage.ErrNotFound) {
						t.Fatal("failed publication left stale entry")
					}
				} else if lookupErr != nil || entry.Token == nil || entry.Token.AccessToken != "access-only" || entry.Token.RefreshToken != "" {
					t.Fatal("access-only result changed")
				}
			}
			_, err = storage.WithFileTransaction(t.Context(), store, func(view storage.Store) error {
				_, err := storage.BeginFileRefresh(t.Context(), view, "copy", sha256.Sum256([]byte("route")))
				return err
			}, nil)
			if !errors.Is(err, storage.ErrRefreshRecoveryRequired) {
				t.Fatal("replacement reopened copied old generation")
			}
		})
	}
}
