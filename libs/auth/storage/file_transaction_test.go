package storage_test

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/databricks/cli/libs/auth/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

func newFileStore(t *testing.T) (storage.Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tokens.json")
	store, err := storage.NewFileStore(t.Context(), storage.WithFileLocation(path))
	require.NoError(t, err)
	return store, path
}

func entry(value string) storage.Entry {
	return storage.Entry{Token: &oauth2.Token{AccessToken: value}}
}

func TestFileTransactionPublishesOnlyCommittedState(t *testing.T) {
	for _, abort := range []bool{false, true} {
		t.Run(map[bool]string{false: "commit", true: "abort"}[abort], func(t *testing.T) {
			store, path := newFileStore(t)
			require.NoError(t, store.Put("primary", entry("old-primary")))
			require.NoError(t, store.Put("legacy", entry("old-legacy")))
			before, err := os.ReadFile(path)
			require.NoError(t, err)
			callbackErr := errors.New("abort staged update")
			handled, err := storage.WithFileTransaction(t.Context(), store, func(view storage.Store) error {
				require.NoError(t, view.Put("primary", entry("new-primary")))
				visible, err := store.Lookup("primary")
				require.NoError(t, err)
				assert.Equal(t, "old-primary", visible.Token.AccessToken)
				require.NoError(t, view.Put("legacy", entry("new-legacy")))
				visible, err = store.Lookup("legacy")
				require.NoError(t, err)
				assert.Equal(t, "old-legacy", visible.Token.AccessToken)
				if abort {
					return callbackErr
				}
				return nil
			}, nil)
			require.True(t, handled)
			if abort {
				require.ErrorIs(t, err, callbackErr)
				after, err := os.ReadFile(path)
				require.NoError(t, err)
				assert.Equal(t, before, after)
			} else {
				require.NoError(t, err)
				for _, key := range []string{"primary", "legacy"} {
					value, err := store.Lookup(key)
					require.NoError(t, err)
					assert.Equal(t, "new-"+key, value.Token.AccessToken)
				}
			}
		})
	}
}

func TestFileTransactionCommitFailureCleanupUsesOriginalMap(t *testing.T) {
	store, path := newFileStore(t)
	require.NoError(t, store.Put("login", entry("stale")))
	require.NoError(t, store.Put("unrelated", entry("keep")))
	backup := path + ".saved"
	cleanupErr := errors.New("cleanup callback failed")
	for _, failCleanup := range []bool{false, true} {
		t.Run(map[bool]string{false: "cleanup", true: "cleanup-error"}[failCleanup], func(t *testing.T) {
			require.NoError(t, store.Put("login", entry("stale")))
			handled, err := storage.WithFileTransaction(t.Context(), store, func(view storage.Store) error {
				require.NoError(t, view.Put("login", entry("new")))
				require.NoError(t, os.Rename(path, backup))
				require.NoError(t, os.Mkdir(path, 0o700))
				return nil
			}, func(view storage.Store) error {
				require.NoError(t, os.Remove(path))
				require.NoError(t, os.Rename(backup, path))
				if failCleanup {
					return cleanupErr
				}
				return view.Delete("login")
			})
			require.True(t, handled)
			require.Error(t, err)
			assert.ErrorContains(t, err, "cache update")
			if failCleanup {
				assert.ErrorIs(t, err, cleanupErr)
				value, err := store.Lookup("login")
				require.NoError(t, err)
				assert.Equal(t, "stale", value.Token.AccessToken)
			} else {
				_, err := store.Lookup("login")
				assert.ErrorIs(t, err, storage.ErrNotFound)
			}
			other, err := store.Lookup("unrelated")
			require.NoError(t, err)
			assert.Equal(t, "keep", other.Token.AccessToken)
		})
	}
}

func TestFileTransactionPreservesPendingLoginAndClonesViews(t *testing.T) {
	store, _ := newFileStore(t)
	pending := &storage.PendingLogin{DeviceCode: "synthetic-device", UserCode: "synthetic-user", Host: "https://host.test", ClientID: "client", Scopes: []string{"scope"}}
	require.NoError(t, store.Put("pending", storage.Entry{PendingLogin: pending}))
	handled, err := storage.WithFileTransaction(t.Context(), store, func(view storage.Store) error {
		value, err := view.Lookup("pending")
		require.NoError(t, err)
		value.PendingLogin.Scopes[0] = "unpublished"
		value, err = view.Lookup("pending")
		require.NoError(t, err)
		assert.Equal(t, pending, value.PendingLogin)
		return view.Put("unrelated", entry("new"))
	}, nil)
	require.True(t, handled)
	require.NoError(t, err)
	value, err := store.Lookup("pending")
	require.NoError(t, err)
	assert.Equal(t, pending, value.PendingLogin)
}

func TestFileTransactionUsesOriginalContextAndDifferentStoresRemainIndependent(t *testing.T) {
	store, path := newFileStore(t)
	other, _ := newFileStore(t)
	held, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	releaseLease := sync.OnceFunc(func() { close(release) })
	t.Cleanup(releaseLease)
	go func() {
		_, err := storage.WithFileTransaction(t.Context(), store, func(view storage.Store) error {
			close(held)
			<-release
			return view.Put("first", entry("first"))
		}, nil)
		done <- err
	}()
	<-held
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	second, err := storage.NewFileStore(ctx, storage.WithFileLocation(path))
	require.NoError(t, err)
	err = storage.PutWithContext(ctx, second, "second", entry("second"))
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	require.NoError(t, other.Put("independent", entry("independent")))
	releaseLease()
	require.NoError(t, <-done)
	require.NoError(t, store.Put("second", entry("second")))
	for _, key := range []string{"first", "second"} {
		_, err := store.Lookup(key)
		assert.NoError(t, err)
	}
}

func TestFileStoreLegacyWritesOutliveInitializationContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	path := filepath.Join(t.TempDir(), "tokens.json")
	store, err := storage.NewFileStore(ctx, storage.WithFileLocation(path))
	require.NoError(t, err)
	cancel()
	require.NoError(t, store.Put("legacy", entry("value")))
	value, err := store.Lookup("legacy")
	require.NoError(t, err)
	assert.Equal(t, "value", value.Token.AccessToken)
	require.NoError(t, store.Delete("legacy"))
	_, err = store.Lookup("legacy")
	assert.ErrorIs(t, err, storage.ErrNotFound)
}

func TestFileStoreCancelledOperationDoesNotMutate(t *testing.T) {
	store, path := newFileStore(t)
	require.NoError(t, store.Put("keep", entry("old")))
	before, err := os.ReadFile(path)
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	assert.ErrorIs(t, storage.PutWithContext(ctx, store, "keep", entry("new")), context.Canceled)
	assert.ErrorIs(t, storage.DeleteWithContext(ctx, store, "keep"), context.Canceled)
	called := false
	handled, err := storage.WithFileTransaction(ctx, store, func(storage.Store) error { called = true; return nil }, nil)
	assert.True(t, handled)
	assert.ErrorIs(t, err, context.Canceled)
	assert.False(t, called)
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, before, after)
}

func TestFileStoreConstructorJoinsTheSameLease(t *testing.T) {
	store, path := newFileStore(t)
	require.NoError(t, store.Put("original", entry("keep")))
	held, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	releaseLease := sync.OnceFunc(func() { close(release) })
	t.Cleanup(releaseLease)
	go func() {
		_, err := storage.WithFileTransaction(t.Context(), store, func(view storage.Store) error {
			if err := os.Remove(path); err != nil {
				return err
			}
			close(held)
			<-release
			return view.Put("restored", entry("restored"))
		}, nil)
		done <- err
	}()
	<-held
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	_, err := storage.NewFileStore(ctx, storage.WithFileLocation(path))
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	_, err = os.Stat(path)
	assert.ErrorIs(t, err, os.ErrNotExist)
	releaseLease()
	require.NoError(t, <-done)
	for _, key := range []string{"original", "restored"} {
		_, err := store.Lookup(key)
		assert.NoError(t, err)
	}
}

func TestUnsupportedStoreDoesNotInvokeFileCallbacks(t *testing.T) {
	store := storage.NewMemoryStore()
	require.NoError(t, store.Put("keep", entry("keep")))
	for _, subject := range []storage.Store{store, storage.WrapForOAuthArgument(t.Context(), store, storage.StorageModeSecure, aliasKey{})} {
		called := false
		handled, err := storage.WithFileTransaction(t.Context(), subject, func(storage.Store) error { called = true; return nil }, nil)
		assert.False(t, handled)
		assert.False(t, called)
		assert.NoError(t, err)
		value, err := subject.Lookup("keep")
		require.NoError(t, err)
		assert.Equal(t, "keep", value.Token.AccessToken)
	}
}

type aliasKey struct{}

func (aliasKey) GetCacheKey() string     { return "primary" }
func (aliasKey) GetHostCacheKey() string { return "https://host.test" }

func TestFileTransactionWrappersShareOneStagedView(t *testing.T) {
	store, _ := newFileStore(t)
	wrapped := storage.WrapForOAuthArgument(t.Context(), store, storage.StorageModePlaintext, aliasKey{})
	require.NoError(t, wrapped.Put("primary", entry("old")))
	handled, err := storage.WithFileTransaction(t.Context(), wrapped, func(view storage.Store) error {
		require.NoError(t, view.Put("primary", entry("new")))
		for _, key := range []string{"primary", "https://host.test"} {
			committed, err := store.Lookup(key)
			require.NoError(t, err)
			assert.Equal(t, "old", committed.Token.AccessToken)
			staged, err := view.Lookup(key)
			require.NoError(t, err)
			assert.Equal(t, "new", staged.Token.AccessToken)
		}
		_, err := view.Lookup("missing")
		assert.ErrorIs(t, err, storage.ErrNotFound)
		return nil
	}, nil)
	require.True(t, handled)
	require.NoError(t, err)
	for _, key := range []string{"primary", "https://host.test"} {
		value, err := store.Lookup(key)
		require.NoError(t, err)
		assert.Equal(t, "new", value.Token.AccessToken)
	}
}

const fileLeaseActor = "DATABRICKS_TEST_FILE_LEASE_ACTOR"

func TestFileTransactionOwnerExitReleasesStableSidecar(t *testing.T) {
	if os.Getenv(fileLeaseActor) == "1" {
		var path string
		require.NoError(t, json.NewDecoder(os.Stdin).Decode(&path))
		store, err := storage.NewFileStore(t.Context(), storage.WithFileLocation(path))
		require.NoError(t, err)
		_, err = storage.WithFileTransaction(t.Context(), store, func(storage.Store) error {
			_, err := io.WriteString(os.Stdout, "held\n")
			if err != nil {
				return err
			}
			_, err = io.Copy(io.Discard, os.Stdin)
			return err
		}, nil)
		require.NoError(t, err)
		return
	}
	store, path := newFileStore(t)
	before, err := os.Stat(path + ".lock")
	require.NoError(t, err)
	command := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestFileTransactionOwnerExitReleasesStableSidecar$", "-test.timeout=30s")
	home := t.TempDir()
	command.Env = []string{fileLeaseActor + "=1", "HOME=" + home, "USERPROFILE=" + home, "PATH=" + os.Getenv("PATH"), "TMPDIR=" + os.TempDir(), "TMP=" + os.TempDir(), "TEMP=" + os.TempDir()}
	input, err := command.StdinPipe()
	require.NoError(t, err)
	output, err := command.StdoutPipe()
	require.NoError(t, err)
	command.Stderr = io.Discard
	require.NoError(t, command.Start())
	t.Cleanup(func() { _ = input.Close(); _ = command.Process.Kill(); _ = command.Wait() })
	require.NoError(t, json.NewEncoder(input).Encode(path))
	ready, err := bufio.NewReader(output).ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "held\n", ready)
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	_, err = storage.WithFileTransaction(ctx, store, func(storage.Store) error { return nil }, nil)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	require.NoError(t, command.Process.Kill())
	require.Error(t, command.Wait())
	require.NoError(t, store.Put("after-exit", entry("after-exit")))
	after, err := os.Stat(path + ".lock")
	require.NoError(t, err)
	assert.True(t, os.SameFile(before, after))
}

func TestFileTransactionPreservesNullEntries(t *testing.T) {
	for _, test := range []struct {
		name     string
		initial  string
		afterPut string
	}{
		{
			name:     "healthy",
			initial:  `{"version":1,"tokens":{"healthy":{"access_token":"synthetic-before","expiry":"0001-01-01T00:00:00Z"}}}`,
			afterPut: `{"version":1,"tokens":{"healthy":{"access_token":"synthetic-before","expiry":"0001-01-01T00:00:00Z"},"added":{"access_token":"synthetic-added","expiry":"0001-01-01T00:00:00Z"}}}`,
		},
		{
			name:     "unrelated_null",
			initial:  `{"version":1,"tokens":{"healthy":{"access_token":"synthetic-before","expiry":"0001-01-01T00:00:00Z"},"unrelated":null}}`,
			afterPut: `{"version":1,"tokens":{"healthy":{"access_token":"synthetic-before","expiry":"0001-01-01T00:00:00Z"},"unrelated":null,"added":{"access_token":"synthetic-added","expiry":"0001-01-01T00:00:00Z"}}}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "tokens.json")
			require.NoError(t, os.WriteFile(path, []byte(test.initial), 0o600))
			store, err := storage.NewFileStore(t.Context(), storage.WithFileLocation(path))
			require.NoError(t, err)
			_, err = store.Lookup("unrelated")
			require.ErrorIs(t, err, storage.ErrNotFound)

			handled, err := storage.WithFileTransaction(t.Context(), store, func(view storage.Store) error {
				_, err := view.Lookup("unrelated")
				require.ErrorIs(t, err, storage.ErrNotFound)
				healthy, err := view.Lookup("healthy")
				require.NoError(t, err)
				require.NotNil(t, healthy.Token)
				assert.Equal(t, "synthetic-before", healthy.Token.AccessToken)
				return nil
			}, nil)
			require.True(t, handled)
			require.NoError(t, err)
			after, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, []byte(test.initial), after)

			require.NoError(t, store.Put("added", entry("synthetic-added")))
			after, err = os.ReadFile(path)
			require.NoError(t, err)
			assert.JSONEq(t, test.afterPut, string(after))
			healthy, err := store.Lookup("healthy")
			require.NoError(t, err)
			require.NotNil(t, healthy.Token)
			assert.Equal(t, "synthetic-before", healthy.Token.AccessToken)

			require.NoError(t, store.Delete("missing"))
			unchanged, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, after, unchanged)
			require.NoError(t, store.Delete("added"))
			after, err = os.ReadFile(path)
			require.NoError(t, err)
			assert.JSONEq(t, test.initial, string(after))
		})
	}
}
