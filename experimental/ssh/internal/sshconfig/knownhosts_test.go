package sshconfig

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testHostKey      = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAICTzJweKcgiNoBUAyuvCY2Qu1Od8mKBON5aJeA03Nl+D"
	testOtherHostKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIErh/dj+R5gtO5dZ5ToM4KYZTpNHOQTHW/RsjRXeOBt3"
)

func TestGetKnownHostsPath(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(env.HomeEnvVar(), tmpDir)

	path, err := GetKnownHostsPath(t.Context(), "databricks-cpu-6e7644d0")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmpDir, ".databricks", "ssh-tunnel-known-hosts", "databricks-cpu-6e7644d0"), path)
}

func TestPinHostKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "known-hosts", "myhost")

	err := PinHostKey(path, "myhost", []byte(testHostKey+" a-comment\n"))
	require.NoError(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	// The comment is dropped: known_hosts keeps host and key, and the line is terminated
	// so ssh doesn't ignore the last entry.
	assert.Equal(t, "myhost "+testHostKey+"\n", string(content))

	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
}

func TestPinHostKeyReplacesPreviousEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "known-hosts", "myhost")

	// A key recorded for this name earlier, in the hashed form OpenSSH writes when
	// HashKnownHosts is on - it cannot be matched by host name, which is why the file is
	// rewritten rather than edited (DECO-27882).
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte("|1|Lvcr5SHFVcEwAhYIbCCuADji6QQ=|rRdETW4qVc96QcC/TtDezxhkfp4= "+testOtherHostKey+"\n"), 0o600))

	err := PinHostKey(path, "myhost", []byte(testHostKey))
	require.NoError(t, err)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "myhost "+testHostKey+"\n", string(content))
}

func TestPinHostKeyLeavesACorrectPinAlone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "known-hosts", "myhost")
	require.NoError(t, PinHostKey(path, "myhost", []byte(testHostKey)))

	// An IDE opens several connections at once and each refreshes the pin, so an
	// already-correct one must not be rewritten. Backdating the file makes the rewrite
	// observable without depending on timer resolution.
	backdated := time.Now().Add(-time.Hour).Truncate(time.Second)
	require.NoError(t, os.Chtimes(path, backdated, backdated))

	require.NoError(t, PinHostKey(path, "myhost", []byte(testHostKey)))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, backdated, info.ModTime().Truncate(time.Second))
}

func TestPinHostKeyRejectsMalformedKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "known-hosts", "myhost")

	err := PinHostKey(path, "myhost", []byte("not-a-key"))
	assert.ErrorContains(t, err, "failed to parse the SSH server host key")

	// A malformed key must not leave a file ssh would then fail to verify against.
	_, err = os.Stat(path)
	assert.ErrorIs(t, err, os.ErrNotExist)
}
