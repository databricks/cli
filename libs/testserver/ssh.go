package testserver

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

// sshTunnelRemoteUser is the login user /metadata reports. It must be the OS user
// the local sshd runs as, so `ssh -l <user>` matches; the real server reports the same.
func sshTunnelRemoteUser() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return ""
}

// sshClientPublicKeySecretKey is the secret the CLI stores its authorized public key
// under (mirrors clientPublicKeyName in experimental/ssh); sshd authorizes it below.
const sshClientPublicKeySecretKey = "client-public-key"

// sshServerPublicKeySecretKey is the secret the tunnel server publishes its host key
// under (mirrors serverPublicKeyName in experimental/ssh); the client reads it to pin
// the host key before connecting.
const sshServerPublicKeySecretKey = "server-public-key"

var sshTunnelUpgrader = websocket.Upgrader{}

// sshdTerminationTimeout caps how long Cmd.Wait blocks on a stuck sshd once the tunnel closes.
const sshdTerminationTimeout = 10 * time.Second

// findSSHD returns the sshd path, or "" if absent. LookPath misses it in sbin, so also
// probe the usual locations. Mirrors the acceptance test's probe.
func findSSHD() string {
	if p, err := exec.LookPath("sshd"); err == nil {
		return p
	}
	for _, p := range []string{"/usr/sbin/sshd", "/usr/local/sbin/sshd", "/sbin/sshd"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// sshClientAuthorizedKey returns the public key the CLI uploaded via secrets/put, or nil.
// Scans all scopes since the scope name embeds the session and current user.
func (s *FakeWorkspace) sshClientAuthorizedKey() []byte {
	defer s.LockUnlock()()
	for _, scope := range s.Secrets {
		if v, ok := scope[sshClientPublicKeySecretKey]; ok {
			return []byte(v)
		}
	}
	return nil
}

// sshTunnelHostKey returns the host key in PEM form that every sshd of this workspace
// serves, generating it on first use. The real tunnel server generates its host key once,
// stores it in the session's secret scope and reuses it for every sshd it launches, so one
// key per workspace is what a client sees across connections.
func (s *FakeWorkspace) sshTunnelHostKey() ([]byte, error) {
	defer s.LockUnlock()()
	return s.ensureSSHTunnelHostKey()
}

// ensureSSHTunnelHostKey generates the workspace's tunnel host key if it has none.
// Callers must hold the workspace lock.
func (s *FakeWorkspace) ensureSSHTunnelHostKey() ([]byte, error) {
	if s.sshTunnelHostKeyPEM != nil {
		return s.sshTunnelHostKeyPEM, nil
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	block, err := ssh.MarshalPrivateKey(priv, "")
	if err != nil {
		return nil, err
	}
	sshPub, err := ssh.NewPublicKey(pub)
	if err != nil {
		return nil, err
	}
	s.sshTunnelHostKeyPEM = pem.EncodeToMemory(block)
	s.sshTunnelHostPublicKey = ssh.MarshalAuthorizedKey(sshPub)
	return s.sshTunnelHostKeyPEM, nil
}

// sshTunnelHandler stands in for a cluster's tunnel server: it upgrades /ssh to a
// websocket and drives a real `sshd -i` over it, so `ssh connect` runs a full handshake,
// auth, and remote exec locally. Requires sshd; the acceptance test skips when it's absent
// and is pinned to Linux, where sshd -i as a non-root user is reliable.
func (s *Server) sshTunnelHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := sshTunnelUpgrader.Upgrade(w, r, nil)
	if err != nil {
		s.t.Logf("ssh tunnel: websocket upgrade failed: %s", err)
		return
	}
	defer conn.Close()

	sshdPath := findSSHD()
	if sshdPath == "" {
		s.t.Logf("ssh tunnel: sshd not found; tunnel cannot start")
		return
	}

	// The ws dial carries the CLI's bearer token, so this token-scoped workspace is the
	// one that stored the uploaded public key.
	ws := s.getWorkspaceForToken(getToken(r))
	authorizedKey := ws.sshClientAuthorizedKey()
	if authorizedKey == nil {
		s.t.Logf("ssh tunnel: client public key secret %q not found", sshClientPublicKeySecretKey)
		return
	}

	hostKey, err := ws.sshTunnelHostKey()
	if err != nil {
		s.t.Logf("ssh tunnel: host key: %s", err)
		return
	}

	dir, err := os.MkdirTemp("", "testserver-sshd-")
	if err != nil {
		s.t.Logf("ssh tunnel: temp dir: %s", err)
		return
	}
	defer os.RemoveAll(dir)

	configPath, err := writeSSHDFiles(dir, authorizedKey, hostKey)
	if err != nil {
		s.t.Logf("ssh tunnel: write sshd files: %s", err)
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	cmd := exec.CommandContext(ctx, sshdPath, "-f", configPath, "-i", "-e")
	cmd.WaitDelay = sshdTerminationTimeout
	cmd.Stderr = io.Discard
	sshdStdin, err := cmd.StdinPipe()
	if err != nil {
		s.t.Logf("ssh tunnel: stdin pipe: %s", err)
		return
	}
	sshdStdout, err := cmd.StdoutPipe()
	if err != nil {
		s.t.Logf("ssh tunnel: stdout pipe: %s", err)
		return
	}
	if err := cmd.Start(); err != nil {
		s.t.Logf("ssh tunnel: start sshd: %s", err)
		return
	}

	closeConn := sync.OnceFunc(func() { _ = conn.Close() })

	var wg sync.WaitGroup

	// sshd stdout -> websocket binary frames.
	wg.Go(func() {
		buf := make([]byte, 4096)
		for {
			n, readErr := sshdStdout.Read(buf)
			if n > 0 {
				if writeErr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); writeErr != nil {
					break
				}
			}
			if readErr != nil {
				break
			}
		}
		// sshd exited (or the client is gone): unblock the reader below.
		closeConn()
	})

	// websocket binary frames -> sshd stdin.
	wg.Go(func() {
		for {
			messageType, data, readErr := conn.ReadMessage()
			if readErr != nil {
				break
			}
			if messageType != websocket.BinaryMessage {
				continue
			}
			if _, writeErr := sshdStdin.Write(data); writeErr != nil {
				break
			}
		}
		// The client closed its side (ssh session ended): EOF sshd's stdin so it exits.
		_ = sshdStdin.Close()
	})

	wg.Wait()
	cancel()
	_ = cmd.Wait()
}

// writeSSHDFiles writes the host key, authorized_keys, and a minimal sshd_config,
// returning the config path. StrictModes/UsePAM are off so sshd accepts the temp-dir keys
// and doesn't try to switch users when run non-root in inetd mode (-i).
func writeSSHDFiles(dir string, authorizedKey, hostKeyPEM []byte) (string, error) {
	hostKeyPath := filepath.Join(dir, "host_key")
	if err := os.WriteFile(hostKeyPath, hostKeyPEM, 0o600); err != nil {
		return "", err
	}

	authKeysPath := filepath.Join(dir, "authorized_keys")
	if err := os.WriteFile(authKeysPath, authorizedKey, 0o600); err != nil {
		return "", err
	}

	cfg := fmt.Sprintf(
		"HostKey %s\n"+
			"AuthorizedKeysFile %s\n"+
			"PidFile none\n"+
			"StrictModes no\n"+
			"UsePAM no\n"+
			"PubkeyAuthentication yes\n"+
			"PasswordAuthentication no\n"+
			"LogLevel ERROR\n",
		hostKeyPath, authKeysPath)
	configPath := filepath.Join(dir, "sshd_config")
	if err := os.WriteFile(configPath, []byte(cfg), 0o600); err != nil {
		return "", err
	}
	return configPath, nil
}
