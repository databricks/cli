package filelock

import (
	"bufio"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcquireRejectsNonpositiveWait(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")
	for _, wait := range []time.Duration{0, -time.Second} {
		t.Run(wait.String(), func(t *testing.T) {
			release, err := Acquire(t.Context(), path, wait)
			assert.Nil(t, release)
			assert.EqualError(t, err, "wait duration must be greater than zero")
			assert.NoFileExists(t, path)
		})
	}
}

func TestAcquireHonorsContextCancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")
	cmd, stdin := startLockHelper(t, path)

	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Millisecond)
	defer cancel()

	release, err := Acquire(ctx, path, time.Minute)
	assert.Nil(t, release)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	stopLockHelper(t, cmd, stdin)
}

func TestAcquireWaitsForRelease(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")
	cmd, stdin := startLockHelper(t, path)

	releaseOnTimeout, err := Acquire(t.Context(), path, 25*time.Millisecond)
	assert.Nil(t, releaseOnTimeout)
	assert.ErrorIs(t, err, errTimeout)

	released := make(chan error, 1)
	go func() {
		timer := time.NewTimer(25 * time.Millisecond)
		defer timer.Stop()
		<-timer.C
		_, err := io.WriteString(stdin, "release\n")
		if err == nil {
			err = stdin.Close()
		}
		released <- err
	}()

	release, err := Acquire(t.Context(), path, time.Second)
	require.NoError(t, err)
	require.NoError(t, release())
	require.NoError(t, <-released)
	require.NoError(t, cmd.Wait())
	assert.FileExists(t, path)
}

const lockHelperPath = "DATABRICKS_CLI_FILELOCK_TEST_PATH"

func TestFileLockHelperProcess(t *testing.T) {
	path := os.Getenv(lockHelperPath)
	if path == "" {
		return
	}

	release, err := Acquire(t.Context(), path, time.Minute)
	require.NoError(t, err)
	_, err = os.Stdout.WriteString("locked\n")
	require.NoError(t, err)

	_, err = bufio.NewReader(os.Stdin).ReadString('\n')
	require.NoError(t, err)
	require.NoError(t, release())
}

func startLockHelper(t *testing.T, path string) (*exec.Cmd, io.WriteCloser) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestFileLockHelperProcess$")
	cmd.Env = append(os.Environ(), lockHelperPath+"="+path)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		_ = stdin.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	})

	locked, err := bufio.NewReader(stdout).ReadString('\n')
	require.NoError(t, err)
	require.Equal(t, "locked\n", locked)
	return cmd, stdin
}

func stopLockHelper(t *testing.T, cmd *exec.Cmd, stdin io.WriteCloser) {
	_, err := io.WriteString(stdin, "release\n")
	require.NoError(t, err)
	require.NoError(t, stdin.Close())
	require.NoError(t, cmd.Wait())
}

func TestAcquireAfterOwnerProcessDies(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.lock")
	cmd, _ := startLockHelper(t, path)

	require.NoError(t, cmd.Process.Kill())
	_ = cmd.Wait()

	release, err := Acquire(t.Context(), path, time.Second)
	require.NoError(t, err)
	require.NoError(t, release())
	assert.FileExists(t, path)
}
