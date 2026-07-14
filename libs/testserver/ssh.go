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

// sshTunnelRemoteUser reports the login user the driver-proxy /metadata endpoint
// returns. The local sshd runs as the current OS user, so `ssh -l <user>` must use
// that same account; the real tunnel server likewise reports its current user.
func sshTunnelRemoteUser() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return ""
}

// sshClientPublicKeySecretKey is the secret key name the CLI stores the client's
// authorized public key under (mirrors clientPublicKeyName in experimental/ssh).
// The tunnel server on a real cluster reads it to build authorized_keys; the test
// server does the same so the local handshake authenticates the CLI's key.
const sshClientPublicKeySecretKey = "client-public-key"

// sshTunnelUpgrader upgrades the driver-proxy /ssh request to a websocket.
var sshTunnelUpgrader = websocket.Upgrader{}

// sshdTerminationTimeout bounds how long we wait for sshd to exit after the
// tunnel closes before Cmd.Wait gives up, so a stuck child can't wedge the test.
const sshdTerminationTimeout = 10 * time.Second

// findSSHD returns the path to the OpenSSH server binary, or "" if it isn't installed.
// LookPath usually misses sshd because it lives in sbin (not on a non-root PATH), so
// probe the conventional locations directly. Mirrors the acceptance test's own probe.
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

// sshClientAuthorizedKey returns the client's authorized public key the CLI uploaded
// via secrets/put, scanning every scope since the scope name is derived from the
// session and current user. Returns nil if no such secret exists.
func (s *FakeWorkspace) sshClientAuthorizedKey() []byte {
	defer s.LockUnlock()()
	for _, scope := range s.Secrets {
		if v, ok := scope[sshClientPublicKeySecretKey]; ok {
			return []byte(v)
		}
	}
	return nil
}

// sshTunnelHandler stands in for the tunnel server a real cluster runs behind the
// driver proxy. With no Databricks compute locally, it upgrades the /ssh request to
// a websocket and drives a real `sshd -i` over that transport, so `ssh connect`
// completes an actual SSH handshake, public-key auth, and remote command exec end to
// end. It is the local stand-in for cluster connectivity: the tunnel must carry the
// SSH protocol faithfully in both directions.
//
// Requires a real OpenSSH server; callers gate the test to skip where sshd is absent
// (see acceptance/ssh/connect-serverless-gpu). sshd -i as a non-root user with a
// throwaway config is reliable on Linux, which is where that test runs.
func (s *Server) sshTunnelHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := sshTunnelUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	sshdPath := findSSHD()
	if sshdPath == "" {
		return
	}

	// The ws dial carries the CLI's bearer token, so the token-scoped workspace here
	// is the same one the connect flow uploaded the client's public key to.
	authorizedKey := s.getWorkspaceForToken(getToken(r)).sshClientAuthorizedKey()
	if authorizedKey == nil {
		return
	}

	dir, err := os.MkdirTemp("", "testserver-sshd-")
	if err != nil {
		return
	}
	defer os.RemoveAll(dir)

	configPath, err := writeSSHDFiles(dir, authorizedKey)
	if err != nil {
		return
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	cmd := exec.CommandContext(ctx, sshdPath, "-f", configPath, "-i", "-e")
	cmd.WaitDelay = sshdTerminationTimeout
	cmd.Stderr = io.Discard
	sshdStdin, err := cmd.StdinPipe()
	if err != nil {
		return
	}
	sshdStdout, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	if err := cmd.Start(); err != nil {
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

// writeSSHDFiles lays out the host key, authorized_keys, and a minimal sshd_config
// in dir and returns the config path. StrictModes/UsePAM are off so sshd doesn't
// reject the temp-dir key files or try to switch users when run as a non-root user
// in inetd mode (-i).
func writeSSHDFiles(dir string, authorizedKey []byte) (string, error) {
	_, hostPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", err
	}
	block, err := ssh.MarshalPrivateKey(hostPriv, "")
	if err != nil {
		return "", err
	}
	hostKeyPath := filepath.Join(dir, "host_key")
	if err := os.WriteFile(hostKeyPath, pem.EncodeToMemory(block), 0o600); err != nil {
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
