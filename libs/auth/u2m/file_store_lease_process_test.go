package u2m_test

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/libs/auth/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fileStoreLeaseActor = "DATABRICKS_TEST_SHARED_FILE_LEASE_ACTOR"

// Delegate the owner's actual API on both the original and corrected source.
func withActualFileLease(ctx context.Context, store storage.Store, callback func() error) error {
	if owner, ok := store.(interface {
		WithFileTransaction(context.Context, func(storage.Store) error, func(storage.Store) error) (bool, error)
	}); ok {
		handled, err := owner.WithFileTransaction(ctx, func(storage.Store) error { return callback() }, nil)
		if !handled {
			return errors.New("file transaction was not handled")
		}
		return err
	}
	if owner, ok := store.(interface {
		Lock(context.Context) (func(), error)
	}); ok {
		release, err := owner.Lock(ctx)
		if err != nil {
			return err
		}
		defer release()
		return callback()
	}
	return errors.New("no file lease API")
}

func TestFileStoreSharedPathLeaseAcrossHomes(t *testing.T) {
	if os.Getenv(fileStoreLeaseActor) == "1" {
		var path string
		require.NoError(t, json.NewDecoder(os.Stdin).Decode(&path))
		store, err := storage.NewFileStore(t.Context(), storage.WithFileLocation(path))
		require.NoError(t, err)
		require.NoError(t, withActualFileLease(t.Context(), store, func() error {
			if _, err := io.WriteString(os.Stdout, "held\n"); err != nil {
				return err
			}
			_, err := io.Copy(io.Discard, os.Stdin)
			return err
		}))
		return
	}
	path := filepath.Join(t.TempDir(), "shared", "tokens.json")
	store, err := storage.NewFileStore(t.Context(), storage.WithFileLocation(path))
	require.NoError(t, err)
	home := t.TempDir()
	command := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestFileStoreSharedPathLeaseAcrossHomes$", "-test.timeout=30s")
	command.Env = []string{fileStoreLeaseActor + "=1", "HOME=" + home, "USERPROFILE=" + home, "PATH=" + os.Getenv("PATH"), "TMPDIR=" + os.TempDir(), "TMP=" + os.TempDir(), "TEMP=" + os.TempDir()}
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
	acquired := false
	err = withActualFileLease(ctx, store, func() error { acquired = true; return nil })
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.False(t, acquired)
	require.NoError(t, input.Close())
	require.NoError(t, command.Wait())
}
