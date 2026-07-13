//go:build linux

package proxy

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http/httptest"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

// findSSHD returns the path to the OpenSSH server binary, or "" if it isn't installed.
func findSSHD() string {
	if p, err := exec.LookPath("sshd"); err == nil {
		return p
	}
	// LookPath misses sshd because it lives in sbin, which is usually not on a
	// non-root user's PATH; probe the conventional locations directly.
	for _, p := range []string{"/usr/sbin/sshd", "/usr/local/sbin/sshd", "/sbin/sshd"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// pipeConn adapts a pair of pipes into a net.Conn so an in-process SSH client can speak the SSH
// protocol over the proxy transport (RunClientProxy reads/writes the other ends of these pipes).
type pipeConn struct {
	r *io.PipeReader
	w *io.PipeWriter
}

func (c pipeConn) Read(p []byte) (int, error)  { return c.r.Read(p) }
func (c pipeConn) Write(p []byte) (int, error) { return c.w.Write(p) }

func (c pipeConn) Close() error {
	_ = c.r.Close()
	return c.w.Close()
}

func (pipeConn) LocalAddr() net.Addr                { return pipeAddr{} }
func (pipeConn) RemoteAddr() net.Addr               { return pipeAddr{} }
func (pipeConn) SetDeadline(_ time.Time) error      { return nil }
func (pipeConn) SetReadDeadline(_ time.Time) error  { return nil }
func (pipeConn) SetWriteDeadline(_ time.Time) error { return nil }

type pipeAddr struct{}

func (pipeAddr) Network() string { return "pipe" }
func (pipeAddr) String() string  { return "pipe" }

func writeOpenSSHPrivateKey(t *testing.T, path string, key ed25519.PrivateKey) {
	block, err := ssh.MarshalPrivateKey(key, "")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, pem.EncodeToMemory(block), 0o600))
}

// writeSSHDConfig writes a minimal sshd_config that lets a real sshd run as a non-root user in
// inetd mode (-i) and authenticate the current user by public key. StrictModes/UsePAM are off so
// sshd doesn't reject the temp-dir key files or try to switch users.
func writeSSHDConfig(t *testing.T, dir, hostKeyPath, authKeysPath string) string {
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
	path := filepath.Join(dir, "sshd_config")
	require.NoError(t, os.WriteFile(path, []byte(cfg), 0o600))
	return path
}

// TestClientServerRealSSHD runs a real OpenSSH server (sshd) behind the websocket proxy and a real
// SSH client through RunClientProxy, exercising the actual SSH handshake and a remote command
// end-to-end without any Databricks compute. It is the local stand-in for the ssh-connectivity
// coverage that otherwise requires a cluster: the tunnel must carry the SSH protocol faithfully in
// both directions. Sibling tests in client_server_test.go cover the transport with a `cat` echo
// server; this one adds a genuine sshd so the handshake, auth, and channel exec are validated too.
//
// The test is Linux-only: running sshd -i as a non-root user with a throwaway config is reliable
// there (CI installs openssh-server), but not on macOS. It still skips gracefully when sshd is
// absent so the general `test` job, which doesn't install it, doesn't fail.
func TestClientServerRealSSHD(t *testing.T) {
	sshdPath := findSSHD()
	if sshdPath == "" {
		t.Skip("sshd (openssh-server) not installed; skipping real SSH handshake test")
	}

	currentUser, err := user.Current()
	require.NoError(t, err)

	dir := t.TempDir()

	// Host key: identifies the server; the client pins it via FixedHostKey.
	_, hostPriv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	hostKeyPath := filepath.Join(dir, "host_key")
	writeOpenSSHPrivateKey(t, hostKeyPath, hostPriv)
	hostSigner, err := ssh.NewSignerFromSigner(hostPriv)
	require.NoError(t, err)

	// Client key: authorized on the server via authorized_keys, presented by the SSH client to log in.
	_, clientPriv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	clientSigner, err := ssh.NewSignerFromSigner(clientPriv)
	require.NoError(t, err)
	authKeysPath := filepath.Join(dir, "authorized_keys")
	require.NoError(t, os.WriteFile(authKeysPath, ssh.MarshalAuthorizedKey(clientSigner.PublicKey()), 0o600))

	sshdConfigPath := writeSSHDConfig(t, dir, hostKeyPath, authKeysPath)

	ctx := cmdio.MockDiscard(t.Context())

	// Server proxy: each websocket connection spawns `sshd -i`, which speaks SSH over its stdio.
	connections := NewConnectionsManager(2, time.Hour)
	server := httptest.NewServer(NewProxyServer(ctx, connections, func(ctx context.Context) *exec.Cmd {
		return exec.CommandContext(ctx, sshdPath, "-f", sshdConfigPath, "-i", "-e")
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	createConn := func(ctx context.Context, connID string) (*websocket.Conn, error) {
		conn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("%s?id=%s", wsURL, connID), nil) // nolint:bodyclose
		return conn, err
	}

	// Wire the SSH client's transport through RunClientProxy:
	//   ssh client --writes--> clientToProxy --> proxy sends over ws --> sshd stdin
	//   ssh client <--reads--- proxyToClient <-- proxy recvs from ws <-- sshd stdout
	clientToProxyR, clientToProxyW := io.Pipe()
	proxyToClientR, proxyToClientW := io.Pipe()

	proxyDone := make(chan error, 1)
	go func() {
		requestHandoverTick := func() <-chan time.Time { return time.After(time.Hour) }
		proxyDone <- RunClientProxy(ctx, clientToProxyR, proxyToClientW, requestHandoverTick, createConn)
	}()

	conn := pipeConn{r: proxyToClientR, w: clientToProxyW}
	sshConfig := &ssh.ClientConfig{
		User:            currentUser.Username,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(clientSigner)},
		HostKeyCallback: ssh.FixedHostKey(hostSigner.PublicKey()),
		Timeout:         30 * time.Second,
	}

	clientConn, chans, reqs, err := ssh.NewClientConn(conn, "pipe", sshConfig)
	require.NoError(t, err, "SSH handshake failed")
	sshClient := ssh.NewClient(clientConn, chans, reqs)
	defer sshClient.Close()

	session, err := sshClient.NewSession()
	require.NoError(t, err)
	defer session.Close()

	out, err := session.Output("echo 'Connection successful'")
	require.NoError(t, err)
	assert.Equal(t, "Connection successful\n", string(out))

	// Tear down the client side and confirm the proxy shuts down cleanly rather than hanging.
	require.NoError(t, sshClient.Close())
	_ = conn.Close()

	select {
	case err := <-proxyDone:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("client proxy did not shut down after the SSH session ended")
	}
}
