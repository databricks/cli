package sshconfig

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/env"
	"golang.org/x/crypto/ssh"
)

// knownHostsDirName is the directory holding the known_hosts files the CLI maintains for
// tunnel connections, relative to the user's home directory.
const knownHostsDirName = ".databricks/ssh-tunnel-known-hosts"

// GetKnownHostsPath returns the known_hosts file the CLI maintains for a session
// (sessionID is the connection name for serverless, the cluster ID otherwise).
//
// Tunnel host keys are deliberately kept out of the user's ~/.ssh/known_hosts. A session
// name identifies compute within one workspace, while ~/.ssh/known_hosts is global and
// keyed by name alone, so the same name used in a second workspace - or against compute
// whose host key was regenerated - collides with the entry left by the first and trips
// strict host key checking on a connection that is perfectly legitimate.
func GetKnownHostsPath(ctx context.Context, sessionID string) (string, error) {
	homeDir, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, filepath.FromSlash(knownHostsDirName), sessionID), nil
}

// PinHostKey makes publicKey the entry for hostName in the known_hosts file at path,
// replacing the file's previous contents.
//
// The tunnel server publishes its host key to the workspace, which makes the workspace the
// authority on it: pinning that key before every connection lets ssh verify the server
// with no trust-on-first-use window, and leaves no way for a key recorded under the same
// name earlier to fail a connection. The file belongs to the CLI, which is why it is
// rewritten rather than edited - an entry recorded with HashKnownHosts cannot be matched
// by host name, so selectively dropping the stale one is not possible.
func PinHostKey(path, hostName string, publicKey []byte) error {
	parsed, _, _, _, err := ssh.ParseAuthorizedKey(publicKey)
	if err != nil {
		return fmt.Errorf("failed to parse the SSH server host key: %w", err)
	}
	// MarshalAuthorizedKey drops any comment and terminates the line with "\n".
	line := hostName + " " + string(ssh.MarshalAuthorizedKey(parsed))

	// Leave an already-correct pin alone, which is the common case: every ssh invocation
	// refreshes it through the ProxyCommand, and an IDE opens several at once. Renaming over
	// a file another ssh has open fails on Windows.
	if existing, err := os.ReadFile(path); err == nil && string(existing) == line {
		return nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create known hosts directory: %w", err)
	}

	// Write and rename so a connection racing this one (every ssh invocation runs the
	// ProxyCommand, which refreshes the pin) never reads a half-written file.
	tmp, err := os.CreateTemp(dir, ".known-hosts-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create known hosts file: %w", err)
	}
	defer os.Remove(tmp.Name())

	_, err = tmp.WriteString(line)
	if err == nil {
		err = tmp.Chmod(0o600)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("failed to write known hosts file: %w", err)
	}

	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("failed to replace known hosts file: %w", err)
	}
	return nil
}
