package sshconfig

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/experimental/ssh/internal/keys"
	"github.com/databricks/cli/libs/atomicfile"
	"github.com/databricks/cli/libs/env"
	"golang.org/x/crypto/ssh"
)

// GetKnownHostsPath returns the known_hosts file the CLI maintains for a session
// (sessionID is the connection name for serverless, the cluster ID otherwise).
// knownHostsDir defaults to ~/.databricks/ssh-tunnel-known-hosts.
//
// The file name is always the session ID, never a path the caller picks, because PinHostKey
// rewrites the file: relocating the directory is all a caller can do, so no shared
// known_hosts can end up being replaced with a single tunnel entry.
//
// Tunnel host keys are deliberately kept out of the user's ~/.ssh/known_hosts. A session
// name identifies compute within one workspace, while ~/.ssh/known_hosts is global and
// keyed by name alone, so the same name used in a second workspace - or against compute
// whose host key was regenerated - collides with the entry left by the first and trips
// strict host key checking on a connection that is perfectly legitimate.
func GetKnownHostsPath(ctx context.Context, sessionID, knownHostsDir string) (string, error) {
	if knownHostsDir == "" {
		homeDir, err := env.UserHomeDir(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		knownHostsDir = filepath.Join(homeDir, ".databricks", "ssh-tunnel-known-hosts")
	}
	return filepath.Join(knownHostsDir, sessionID), nil
}

// GetServerKnownHostsPath returns the known_hosts file the CLI maintains for one server of a
// session, the one whose host key has hostKeyFingerprint. It sits next to the session's file
// (see GetKnownHostsPath), named after the session and a digest of the fingerprint; session
// IDs contain no dot, so it cannot be another session's file.
//
// Servers started for a session at the same time each publish a key of their own, and a
// connection to one of them must not find the other's key in the file it reads. The file
// holds only the key with hostKeyFingerprint, so a connect to another server of the session
// never rewrites it.
func GetServerKnownHostsPath(ctx context.Context, sessionID, hostKeyFingerprint, knownHostsDir string) (string, error) {
	sessionPath, err := GetKnownHostsPath(ctx, sessionID, knownHostsDir)
	if err != nil {
		return "", err
	}
	return sessionPath + keys.FingerprintFileSuffix(hostKeyFingerprint), nil
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

	// Write and rename so a connection racing this one (every ssh invocation runs the
	// ProxyCommand, which refreshes the pin) never reads a half-written file.
	if err := atomicfile.Write(path, []byte(line), 0o600, atomicfile.MkDir(0o700)); err != nil {
		return fmt.Errorf("failed to write known hosts file: %w", err)
	}
	return nil
}
