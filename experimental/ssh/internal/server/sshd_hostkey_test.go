package server

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/databricks/cli/experimental/ssh/internal/keys"
	sshWorkspace "github.com/databricks/cli/experimental/ssh/internal/workspace"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareSSHDConfigReplacesHostKeyInScope(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)

	const scope = "tester-session1-ssh-tunnel-keys"
	ctx := t.Context()
	require.NoError(t, client.Secrets.CreateScope(ctx, workspace.CreateScope{Scope: scope}))
	// A host key pair the scope already holds before the server starts.
	for key, value := range map[string]string{
		"client-public-key":  "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBpslsfLaLLxmaaNGXZyodNwaHuO1pTQxGX1/3j/DeJo",
		"server-private-key": "existing-private",
		"server-public-key":  "ssh-ed25519 AAAAexisting",
	} {
		require.NoError(t, client.Secrets.PutSecret(ctx, workspace.PutSecret{Scope: scope, Key: key, StringValue: value}))
	}

	var mu sync.Mutex
	var reads []string
	server.RequestCallback = func(r *testserver.Request) {
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/2.0/secrets/") {
			mu.Lock()
			defer mu.Unlock()
			reads = append(reads, r.URL.Query().Get("key"))
		}
	}

	home := t.TempDir()
	ctx = env.WithUserHomeDir(cmdio.MockDiscard(ctx), home)
	opts := ServerOptions{
		ConfigDir:               ".ssh-tunnel",
		SecretScopeName:         scope,
		AuthorizedKeySecretName: "client-public-key",
		ServerPrivateKeyName:    "server-private-key",
		ServerPublicKeyName:     "server-public-key",
	}
	_, returnedPublic, returnedClientPublic, err := prepareSSHDConfig(ctx, client, opts, 7772)
	require.NoError(t, err)

	// The existing host key is not read, only overwritten.
	mu.Lock()
	assert.Equal(t, []string{"client-public-key"}, reads)
	mu.Unlock()

	privateKey, publicKey, err := keys.GetSSHKeyPairFromSecrets(ctx, client, scope, opts.ServerPrivateKeyName, opts.ServerPublicKeyName)
	require.NoError(t, err)
	assert.NotEqual(t, "ssh-ed25519 AAAAexisting", string(publicKey))
	assert.Equal(t, publicKey, returnedPublic)
	// The client key sshd authorizes is returned as it is written to authorized_keys.
	authorizedKeys, err := os.ReadFile(filepath.Join(home, opts.ConfigDir, "7772", "authorized_keys"))
	require.NoError(t, err)
	assert.Equal(t, authorizedKeys, returnedClientPublic)

	// sshd serves the key the scope now holds, which is the one clients pin.
	keyPath := filepath.Join(home, opts.ConfigDir, "7772", "keys", opts.ServerPrivateKeyName)
	_, localPublic, err := keys.LoadSSHKeyPair(keyPath)
	require.NoError(t, err)
	assert.Equal(t, publicKey, localPublic)
	localPrivate, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	assert.Equal(t, privateKey, localPrivate)
}

func TestListenOnAvailablePortHoldsThePort(t *testing.T) {
	// Port 0 makes the system pick a free port.
	first, _, err := listenOnAvailablePort(0, 1)
	require.NoError(t, err)
	defer first.Close()
	port := first.Addr().(*net.TCPAddr).Port

	// A second server starting while the first holds the port fails to get it, instead of
	// finding it free and losing it once it listens.
	_, _, err = listenOnAvailablePort(port, 1)
	assert.ErrorContains(t, err, "no available port found")
}

func TestRunPublishesNoHostKeyWithoutTheListener(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)

	const scope = "tester-session1-ssh-tunnel-keys"
	ctx := t.Context()
	require.NoError(t, client.Secrets.CreateScope(ctx, workspace.CreateScope{Scope: scope}))
	// The host key published by the server that started alongside this one and owns the port.
	const winnerPublicKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBpslsfLaLLxmaaNGXZyodNwaHuO1pTQxGX1/3j/DeJo"
	require.NoError(t, client.Secrets.PutSecret(ctx, workspace.PutSecret{Scope: scope, Key: "server-public-key", StringValue: winnerPublicKey}))

	var mu sync.Mutex
	var requests []string
	server.RequestCallback = func(r *testserver.Request) {
		mu.Lock()
		defer mu.Unlock()
		requests = append(requests, r.Method+" "+r.URL.Path)
	}

	winner, err := net.Listen("tcp", "0.0.0.0:0")
	require.NoError(t, err)
	defer winner.Close()

	ctx = env.WithUserHomeDir(cmdio.MockDiscard(ctx), t.TempDir())
	err = Run(ctx, client, ServerOptions{
		Serverless:              true,
		ConfigDir:               ".ssh-tunnel",
		SecretScopeName:         scope,
		AuthorizedKeySecretName: "client-public-key",
		ServerPrivateKeyName:    "server-private-key",
		ServerPublicKeyName:     "server-public-key",
		DefaultPort:             winner.Addr().(*net.TCPAddr).Port,
		PortRange:               1,
	})
	assert.ErrorContains(t, err, "failed to find available port")

	mu.Lock()
	assert.Empty(t, requests)
	mu.Unlock()
	publicKey, err := keys.GetSecret(t.Context(), client, scope, "server-public-key")
	require.NoError(t, err)
	assert.Equal(t, winnerPublicKey, string(publicKey))
}

const (
	testHostKeysScope = "tester-session1-ssh-tunnel-keys"
	testVersion       = "v0.0.0-test"
	testSessionID     = "session1"
)

var testPublishOpts = ServerOptions{
	Version:                 testVersion,
	SessionID:               testSessionID,
	ConfigDir:               ".ssh-tunnel",
	SecretScopeName:         testHostKeysScope,
	AuthorizedKeySecretName: "client-public-key",
	ServerPrivateKeyName:    "server-private-key",
	ServerPublicKeyName:     "server-public-key",
}

// newPublishTestClient returns a client for a fake workspace whose testHostKeysScope holds a
// client public key, the fake server, and the workspace home of its user.
func newPublishTestClient(t *testing.T) (*databricks.WorkspaceClient, *testserver.Server, string) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)

	ctx := t.Context()
	require.NoError(t, client.Secrets.CreateScope(ctx, workspace.CreateScope{Scope: testHostKeysScope}))
	require.NoError(t, client.Secrets.PutSecret(ctx, workspace.PutSecret{
		Scope:       testHostKeysScope,
		Key:         "client-public-key",
		StringValue: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBpslsfLaLLxmaaNGXZyodNwaHuO1pTQxGX1/3j/DeJo",
	}))
	wsHome, err := sshWorkspace.GetWorkspaceHomeDir(ctx, client)
	require.NoError(t, err)
	return client, server, wsHome
}

// localHostKeyFingerprint returns the fingerprint of the host key sshd serves for the server
// on port.
func localHostKeyFingerprint(t *testing.T, home string, port int) string {
	_, publicKey, err := keys.LoadSSHKeyPair(filepath.Join(home, testPublishOpts.ConfigDir, strconv.Itoa(port), "keys", testPublishOpts.ServerPrivateKeyName))
	require.NoError(t, err)
	fingerprint, err := keys.PublicKeyFingerprint(publicKey)
	require.NoError(t, err)
	return fingerprint
}

// localClientKeyFingerprint returns the fingerprint of the client key sshd authorizes for the
// server on port.
func localClientKeyFingerprint(t *testing.T, home string, port int) string {
	authorizedKeys, err := os.ReadFile(filepath.Join(home, testPublishOpts.ConfigDir, strconv.Itoa(port), "authorized_keys"))
	require.NoError(t, err)
	fingerprint, err := keys.PublicKeyFingerprint(authorizedKeys)
	require.NoError(t, err)
	return fingerprint
}

func scopeHostKeyFingerprint(t *testing.T, client *databricks.WorkspaceClient) string {
	publicKey, err := keys.GetSecret(t.Context(), client, testHostKeysScope, testPublishOpts.ServerPublicKeyName)
	require.NoError(t, err)
	fingerprint, err := keys.PublicKeyFingerprint(publicKey)
	require.NoError(t, err)
	return fingerprint
}

func TestPublishHostKeyAndMetadataWritesTheKeyFirst(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("prepareSSHDConfig creates /run/sshd; the SSH tunnel server only runs on Linux")
	}
	client, server, wsHome := newPublishTestClient(t)

	var mu sync.Mutex
	var writes []string
	server.RequestCallback = func(r *testserver.Request) {
		if r.Method != http.MethodPost {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		writes = append(writes, r.URL.Path)
	}

	home := t.TempDir()
	ctx := env.WithUserHomeDir(cmdio.MockDiscard(t.Context()), home)
	_, err := publishHostKeyAndMetadata(ctx, client, wsHome, testPublishOpts, 7772)
	require.NoError(t, err)

	// metadata.json lands only after both halves of the host key, so a client that finds it
	// can already read the key it names.
	mu.Lock()
	require.NotEmpty(t, writes)
	assert.Equal(t, []string{"/api/2.0/secrets/put", "/api/2.0/secrets/put"}, writes[:2])
	assert.True(t, strings.HasSuffix(writes[len(writes)-1], "/metadata.json"), "last write: %s", writes[len(writes)-1])
	mu.Unlock()

	metadata, err := sshWorkspace.GetWorkspaceMetadata(ctx, client, testVersion, testSessionID)
	require.NoError(t, err)
	assert.Equal(t, 7772, metadata.Port)
	assert.Equal(t, scopeHostKeyFingerprint(t, client), metadata.HostKeyFingerprint)
	assert.Equal(t, localHostKeyFingerprint(t, home, 7772), metadata.HostKeyFingerprint)
	// It also names the client key the server authorized.
	assert.Equal(t, localClientKeyFingerprint(t, home, 7772), metadata.ClientKeyFingerprint)
}

func TestPublishHostKeyAndMetadataConcurrentServersOnOneCompute(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("prepareSSHDConfig creates /run/sshd; the SSH tunnel server only runs on Linux")
	}
	client, _, wsHome := newPublishTestClient(t)

	// Another process holds the default port, so both servers move up the port range.
	taken, err := net.Listen("tcp", "0.0.0.0:0")
	require.NoError(t, err)
	defer taken.Close()
	defaultPort := taken.Addr().(*net.TCPAddr).Port
	const portRange = 100

	// Both servers run on the same compute, so they share the home directory.
	home := t.TempDir()
	ctx := env.WithUserHomeDir(cmdio.MockDiscard(t.Context()), home)

	ports := make([]int, 2)
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i := range ports {
		listener, port, err := listenOnAvailablePort(defaultPort, portRange)
		require.NoError(t, err)
		defer listener.Close()
		ports[i] = port
		wg.Go(func() {
			_, errs[i] = publishHostKeyAndMetadata(ctx, client, wsHome, testPublishOpts, port)
		})
	}
	wg.Wait()
	require.NoError(t, errs[0])
	require.NoError(t, errs[1])
	require.NotEqual(t, ports[0], ports[1])

	// Each server keeps serving the key it published, though the other one wrote its own key
	// to the same home directory and the same secret scope.
	fingerprints := []string{localHostKeyFingerprint(t, home, ports[0]), localHostKeyFingerprint(t, home, ports[1])}
	assert.NotEqual(t, fingerprints[0], fingerprints[1])

	// metadata.json names one of the servers together with the key that server serves. The
	// scope's key and metadata.json are separate writes, so the scope may hold the other
	// server's key; clients then refuse to pin it for this server.
	metadata, err := sshWorkspace.GetWorkspaceMetadata(ctx, client, testVersion, testSessionID)
	require.NoError(t, err)
	require.Contains(t, ports, metadata.Port)
	assert.Equal(t, localHostKeyFingerprint(t, home, metadata.Port), metadata.HostKeyFingerprint)
	assert.Contains(t, fingerprints, scopeHostKeyFingerprint(t, client))
}
