package client

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/databricks/cli/experimental/ssh/internal/keys"
	sshWorkspace "github.com/databricks/cli/experimental/ssh/internal/workspace"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testKeysScope = "tester-session1-ssh-tunnel-keys"

var testKeyOpts = ClientOptions{
	ClientPrivateKeyName: "client-private-key",
	ClientPublicKeyName:  "client-public-key",
}

// existingClientKeys is a client key pair the scope already holds before a new server starts.
var existingClientKeys = map[string]string{
	"client-private-key": "existing-private",
	"client-public-key":  `command="echo existing" ssh-ed25519 AAAAexisting`,
}

// newKeysScopeClient returns a client for a fake workspace whose testKeysScope holds the given
// secrets, and a function listing the secret requests made after setup.
func newKeysScopeClient(t *testing.T, secrets map[string]string) (*databricks.WorkspaceClient, func() []string) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)

	ctx := t.Context()
	require.NoError(t, client.Secrets.CreateScope(ctx, workspace.CreateScope{Scope: testKeysScope}))
	for key, value := range secrets {
		require.NoError(t, client.Secrets.PutSecret(ctx, workspace.PutSecret{Scope: testKeysScope, Key: key, StringValue: value}))
	}

	var mu sync.Mutex
	var requests []string
	server.RequestCallback = func(r *testserver.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/2.0/secrets/") {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		requests = append(requests, r.Method+" "+r.URL.Path)
	}
	return client, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return requests
	}
}

func scopeClientKeys(t *testing.T, client *databricks.WorkspaceClient) ([]byte, []byte) {
	privateKey, publicKey, err := keys.GetSSHKeyPairFromSecrets(t.Context(), client, testKeysScope, testKeyOpts.ClientPrivateKeyName, testKeyOpts.ClientPublicKeyName)
	require.NoError(t, err)
	return privateKey, publicKey
}

func assertLocalKeys(t *testing.T, keyPath string, privateKey, publicKey []byte) {
	gotPrivate, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	assert.Equal(t, privateKey, gotPrivate)
	gotPublic, err := os.ReadFile(keyPath + ".pub")
	require.NoError(t, err)
	assert.Equal(t, publicKey, gotPublic)
}

func TestStoreClientKeysForNewServerReplacesScopeAndLocalKeys(t *testing.T) {
	client, requests := newKeysScopeClient(t, existingClientKeys)
	keyPath := filepath.Join(t.TempDir(), "session1")
	oldPrivate, oldPublic, err := keys.GenerateSSHKeyPair()
	require.NoError(t, err)
	require.NoError(t, keys.SaveSSHKeyPair(keyPath, oldPrivate, oldPublic))

	require.NoError(t, storeClientKeysForNewServer(t.Context(), client, testKeysScope, keyPath, testKeyOpts))
	// The existing keys are overwritten without being read.
	assert.Equal(t, []string{"POST /api/2.0/secrets/put", "POST /api/2.0/secrets/put"}, requests())

	privateKey, publicKey := scopeClientKeys(t, client)
	assert.NotEqual(t, existingClientKeys["client-public-key"], string(publicKey))
	// Outside proxy mode nothing has loaded the local key yet, so it is replaced too.
	assert.NotEqual(t, oldPublic, publicKey)
	assertLocalKeys(t, keyPath, privateKey, publicKey)
}

func TestStoreClientKeysForNewServerInProxyModeKeepsLocalKey(t *testing.T) {
	client, requests := newKeysScopeClient(t, existingClientKeys)
	keyPath := filepath.Join(t.TempDir(), "session1")
	localPrivate, localPublic, err := keys.GenerateSSHKeyPair()
	require.NoError(t, err)
	// The local public key file came from the scope before, so it may carry options.
	require.NoError(t, keys.SaveSSHKeyPair(keyPath, localPrivate, append([]byte(`command="echo existing" `), localPublic...)))

	opts := testKeyOpts
	opts.ProxyMode = true
	require.NoError(t, storeClientKeysForNewServer(t.Context(), client, testKeysScope, keyPath, opts))
	assert.Equal(t, []string{"POST /api/2.0/secrets/put", "POST /api/2.0/secrets/put"}, requests())

	// The ssh client running us has loaded the local key, so the server must authorize that
	// one, with the public key derived from the private key.
	privateKey, publicKey := scopeClientKeys(t, client)
	assert.Equal(t, localPrivate, privateKey)
	assert.Equal(t, localPublic, publicKey)
	assertLocalKeys(t, keyPath, localPrivate, localPublic)
}

func TestStoreClientKeysForNewServerInProxyModeGeneratesMissingLocalKey(t *testing.T) {
	client, requests := newKeysScopeClient(t, existingClientKeys)
	keyPath := filepath.Join(t.TempDir(), "session1")

	opts := testKeyOpts
	opts.ProxyMode = true
	require.NoError(t, storeClientKeysForNewServer(t.Context(), client, testKeysScope, keyPath, opts))
	assert.Equal(t, []string{"POST /api/2.0/secrets/put", "POST /api/2.0/secrets/put"}, requests())

	privateKey, publicKey := scopeClientKeys(t, client)
	assert.NotEqual(t, existingClientKeys["client-public-key"], string(publicKey))
	assertLocalKeys(t, keyPath, privateKey, publicKey)
}

func TestSaveRunningServerClientKeysUsesScopeKeys(t *testing.T) {
	privateKey, publicKey, err := keys.GenerateSSHKeyPair()
	require.NoError(t, err)
	client, requests := newKeysScopeClient(t, map[string]string{
		"client-private-key": string(privateKey),
		"client-public-key":  string(publicKey),
	})
	keyPath := filepath.Join(t.TempDir(), "session1")

	require.NoError(t, saveRunningServerClientKeys(t.Context(), client, testKeysScope, keyPath, testKeyOpts))
	assert.Equal(t, []string{"GET /api/2.0/secrets/get", "GET /api/2.0/secrets/get"}, requests())
	assertLocalKeys(t, keyPath, privateKey, publicKey)
}

func TestSaveRunningServerClientKeysFailsWhenMissing(t *testing.T) {
	client, requests := newKeysScopeClient(t, nil)
	keyPath := filepath.Join(t.TempDir(), "session1")

	err := saveRunningServerClientKeys(t.Context(), client, testKeysScope, keyPath, testKeyOpts)
	// The running server would not accept a new pair, so none is generated.
	assert.ErrorIs(t, err, errClientKeyPair)
	assert.Equal(t, []string{"GET /api/2.0/secrets/get"}, requests())
	assert.NoFileExists(t, keyPath)
}

func TestServerHostKeyNamedByMetadata(t *testing.T) {
	_, publicA, err := keys.GenerateSSHKeyPair()
	require.NoError(t, err)
	_, publicB, err := keys.GenerateSSHKeyPair()
	require.NoError(t, err)
	fingerprintA, err := keys.PublicKeyFingerprint(publicA)
	require.NoError(t, err)
	opts := ClientOptions{ServerPublicKeyName: "server-public-key"}
	// metadata.json names server A on port 7772.
	metadata := &sshWorkspace.WorkspaceMetadata{Port: 7772, HostKeyFingerprint: fingerprintA}

	t.Run("scope holds the named server's key", func(t *testing.T) {
		client, _ := newKeysScopeClient(t, map[string]string{"server-public-key": string(publicA)})
		hostKey, err := serverHostKeyNamedBy(t.Context(), client, testKeysScope, metadata, opts)
		require.NoError(t, err)
		assert.Equal(t, publicA, hostKey)
	})

	t.Run("another server replaced the key", func(t *testing.T) {
		// Server B, started at the same time, wrote its key over A's, and A's metadata.json
		// landed after B's.
		client, _ := newKeysScopeClient(t, map[string]string{"server-public-key": string(publicB)})
		_, err := serverHostKeyNamedBy(t.Context(), client, testKeysScope, metadata, opts)
		// Reported as a server that is not ready, so the caller waits for or starts one whose
		// key is in the scope rather than pinning B's key for A.
		assert.ErrorIs(t, err, errServerMetadata)
		assert.ErrorContains(t, err, "is not the key of the server on port 7772")
	})
}

func TestEnsureSSHServerIsRunningReplacesServerWithTornClientKeys(t *testing.T) {
	privateA, publicA, err := keys.GenerateSSHKeyPair()
	require.NoError(t, err)
	privateB, publicB, err := keys.GenerateSSHKeyPair()
	require.NoError(t, err)
	_, hostKeyA, err := keys.GenerateSSHKeyPair()
	require.NoError(t, err)
	fingerprintA, err := keys.PublicKeyFingerprint(hostKeyA)
	require.NoError(t, err)

	client, _ := newKeysScopeClient(t, map[string]string{"server-public-key": string(hostKeyA)})
	ctx := cmdio.MockDiscard(t.Context())
	// Two connects start a server for the session at the same time and interleave their
	// writes of the client pair: A's private key, B's pair, then A's public key. Server A,
	// healthy and named by metadata.json, authorized public key A, whose private key is gone.
	for _, secret := range []workspace.PutSecret{
		{Key: "client-private-key", StringValue: string(privateA)},
		{Key: "client-private-key", StringValue: string(privateB)},
		{Key: "client-public-key", StringValue: string(publicB)},
		{Key: "client-public-key", StringValue: string(publicA)},
	} {
		secret.Scope = testKeysScope
		require.NoError(t, client.Secrets.PutSecret(ctx, secret))
	}
	const version = "test-version"
	wsHome, err := sshWorkspace.GetWorkspaceHomeDir(ctx, client)
	require.NoError(t, err)
	clientFingerprintA, err := keys.PublicKeyFingerprint(publicA)
	require.NoError(t, err)
	require.NoError(t, sshWorkspace.SaveWorkspaceMetadata(ctx, client, wsHome, version, "session1", &sshWorkspace.WorkspaceMetadata{
		Port:                 7772,
		ClusterID:            "1234-567890-serverless",
		HostKeyFingerprint:   fingerprintA,
		ClientKeyFingerprint: clientFingerprintA,
	}))

	// A later connect reuses no server it cannot authenticate to: it replaces server A.
	opts := testKeyOpts
	opts.ConnectionName = "session1"
	opts.ServerPublicKeyName = "server-public-key"
	opts.SSHKeysDir = t.TempDir()
	keyPath := filepath.Join(opts.SSHKeysDir, "session1")
	meta, authorizedKeyPath, err := ensureSSHServerIsRunning(ctx, client, version, testKeysScope, keyPath, opts)
	require.NoError(t, err)

	// The test server stands in for the new server and publishes its own host key.
	assert.NotEqual(t, hostKeyA, meta.HostKey)
	privateKey, publicKey := scopeClientKeys(t, client)
	assert.NotEqual(t, publicA, publicKey)
	assert.NotEqual(t, publicB, publicKey)
	assertLocalKeys(t, authorizedKeyPath, privateKey, publicKey)
}

const (
	testVersion = "test-version"
	testSession = "session1"
	// The port and cluster the test server runs SSH servers on.
	testServerPort    = 7772
	testServerCluster = "1234-567890-serverless"
)

// testServer is the key material of one SSH server of testSession: its host key, and the
// client key pair it authorized.
type testServer struct {
	hostKey           []byte
	hostFingerprint   string
	clientPrivate     []byte
	clientPublic      []byte
	clientFingerprint string
}

func newTestServer(t *testing.T) testServer {
	_, hostKey, err := keys.GenerateSSHKeyPair()
	require.NoError(t, err)
	hostFingerprint, err := keys.PublicKeyFingerprint(hostKey)
	require.NoError(t, err)
	clientPrivate, clientPublic, err := keys.GenerateSSHKeyPair()
	require.NoError(t, err)
	clientFingerprint, err := keys.PublicKeyFingerprint(clientPublic)
	require.NoError(t, err)
	return testServer{hostKey, hostFingerprint, clientPrivate, clientPublic, clientFingerprint}
}

// newSessionClient returns a client for a fake workspace with an empty testKeysScope, a
// function listing the secret requests made after setup, options for testSession, and the
// session's key file.
func newSessionClient(t *testing.T) (*databricks.WorkspaceClient, func() []string, ClientOptions, string) {
	client, requests := newKeysScopeClient(t, nil)
	opts := testKeyOpts
	opts.ConnectionName = testSession
	opts.ServerPublicKeyName = "server-public-key"
	opts.SSHKeysDir = t.TempDir()
	opts.KnownHostsDir = t.TempDir()
	return client, requests, opts, filepath.Join(opts.SSHKeysDir, testSession)
}

// startTestServer makes server the running SSH server of testSession on testServerPort: it
// publishes the server's host key and names the server in metadata.json, as the server does
// when it starts. The scope's client key pair is set to scopePrivate and scopePublic, which
// is the pair the server authorized unless a connect started another server at the same time.
func startTestServer(t *testing.T, client *databricks.WorkspaceClient, server testServer, scopePrivate, scopePublic []byte) {
	ctx := t.Context()
	for key, value := range map[string][]byte{
		"server-public-key":  server.hostKey,
		"client-private-key": scopePrivate,
		"client-public-key":  scopePublic,
	} {
		require.NoError(t, client.Secrets.PutSecret(ctx, workspace.PutSecret{Scope: testKeysScope, Key: key, StringValue: string(value)}))
	}
	wsHome, err := sshWorkspace.GetWorkspaceHomeDir(ctx, client)
	require.NoError(t, err)
	require.NoError(t, sshWorkspace.SaveWorkspaceMetadata(ctx, client, wsHome, testVersion, testSession, &sshWorkspace.WorkspaceMetadata{
		Port:                 testServerPort,
		ClusterID:            testServerCluster,
		HostKeyFingerprint:   server.hostFingerprint,
		ClientKeyFingerprint: server.clientFingerprint,
	}))
}

// connectSession selects the running server of testSession the way a connect that spawns ssh
// or launches an IDE does. It returns the ProxyCommand metadata it writes, and the key file
// and known_hosts file it points ssh at.
func connectSession(t *testing.T, client *databricks.WorkspaceClient, opts ClientOptions, sessionKeyPath string) (string, string, string) {
	ctx := cmdio.MockDiscard(t.Context())
	meta, keyPath, err := ensureSSHServerIsRunning(ctx, client, testVersion, testKeysScope, sessionKeyPath, opts)
	require.NoError(t, err)
	knownHostsPath, err := pinServerHostKey(ctx, testSession, meta.HostKeyFingerprint, meta.HostKey, opts)
	require.NoError(t, err)
	return FormatMetadata(meta.UserName, meta.Port, meta.ClusterID, meta.HostKeyFingerprint, meta.ClientKeyFingerprint), keyPath, knownHostsPath
}

// proxyCommandKeys does what Run, as the ProxyCommand with --metadata, does with the keys
// before the tunnel carries a byte. It returns the key file and the known_hosts file it set up
// for ssh, and whether it reused the server --metadata names rather than finding or starting
// one.
func proxyCommandKeys(t *testing.T, client *databricks.WorkspaceClient, opts ClientOptions, sessionKeyPath, metadata string) (string, string, bool) {
	ctx := cmdio.MockDiscard(t.Context())
	opts.ProxyMode = true
	_, port, clusterID, hostFingerprint, clientFingerprint, err := parseServerMetadata(metadata, "")
	require.NoError(t, err)
	keyPath, hostKey, err := namedServerKeys(ctx, client, testVersion, testKeysScope, sessionKeyPath, port, clusterID, hostFingerprint, clientFingerprint, opts)
	reused := true
	if errors.Is(err, errNamedServerReplaced) || errors.Is(err, errClientKeysNotAuthorized) {
		reused = false
		var meta serverMetadata
		meta, keyPath, err = ensureSSHServerIsRunning(ctx, client, testVersion, testKeysScope, keyPath, opts)
		hostKey = meta.HostKey
	}
	require.NoError(t, err)
	knownHostsPath, err := pinServerHostKey(ctx, testSession, hostFingerprint, hostKey, opts)
	require.NoError(t, err)
	return keyPath, knownHostsPath, reused
}

func assertPinned(t *testing.T, knownHostsPath string, hostKey []byte) {
	pinned, err := os.ReadFile(knownHostsPath)
	require.NoError(t, err)
	assert.Equal(t, testSession+" "+string(hostKey), string(pinned))
}

func TestEnsureSSHServerIsRunningChecksTheServerAuthorizedTheScopeKeys(t *testing.T) {
	serverA := newTestServer(t)
	serverB := newTestServer(t)

	t.Run("scope holds the pair the server authorized", func(t *testing.T) {
		client, _, opts, keyPath := newSessionClient(t)
		startTestServer(t, client, serverA, serverA.clientPrivate, serverA.clientPublic)
		meta, authorizedKeyPath, err := ensureSSHServerIsRunning(cmdio.MockDiscard(t.Context()), client, testVersion, testKeysScope, keyPath, opts)
		require.NoError(t, err)

		// Server A is reused with the pair it authorized, saved in A's own file.
		assert.Equal(t, serverA.hostKey, meta.HostKey)
		wantKeyPath, err := keys.GetLocalServerSSHKeyPath(t.Context(), testSession, serverA.clientFingerprint, opts.SSHKeysDir)
		require.NoError(t, err)
		assert.Equal(t, wantKeyPath, authorizedKeyPath)
		assertLocalKeys(t, authorizedKeyPath, serverA.clientPrivate, serverA.clientPublic)
	})

	t.Run("scope holds another complete pair", func(t *testing.T) {
		// Server B, started at the same time, wrote both halves of pair B after server A read
		// public key A, and A's host key and metadata.json landed after B's.
		client, _, opts, keyPath := newSessionClient(t)
		startTestServer(t, client, serverA, serverB.clientPrivate, serverB.clientPublic)
		meta, authorizedKeyPath, err := ensureSSHServerIsRunning(cmdio.MockDiscard(t.Context()), client, testVersion, testKeysScope, keyPath, opts)
		require.NoError(t, err)

		// Server A would reject pair B, so it is replaced rather than reused with pair B. The
		// test server stands in for the new server and publishes its own host key.
		assert.NotEqual(t, serverA.hostKey, meta.HostKey)
		privateKey, publicKey := scopeClientKeys(t, client)
		assert.NotEqual(t, serverA.clientPublic, publicKey)
		assert.NotEqual(t, serverB.clientPublic, publicKey)
		assertLocalKeys(t, authorizedKeyPath, privateKey, publicKey)
		fingerprint, err := keys.PublicKeyFingerprint(publicKey)
		require.NoError(t, err)
		assert.Equal(t, fingerprint, meta.ClientKeyFingerprint)
	})
}

func TestSaveAuthorizedClientKeys(t *testing.T) {
	privateA, publicA, err := keys.GenerateSSHKeyPair()
	require.NoError(t, err)
	_, publicB, err := keys.GenerateSSHKeyPair()
	require.NoError(t, err)
	fingerprintA, err := keys.PublicKeyFingerprint(publicA)
	require.NoError(t, err)
	meta := serverMetadata{Port: 7772, ClientKeyFingerprint: fingerprintA}

	t.Run("pair the server authorized", func(t *testing.T) {
		client, _ := newKeysScopeClient(t, map[string]string{"client-private-key": string(privateA), "client-public-key": string(publicA)})
		keyPath := filepath.Join(t.TempDir(), "session1")
		require.NoError(t, saveAuthorizedClientKeys(t.Context(), client, testKeysScope, keyPath, meta, testKeyOpts))
		assertLocalKeys(t, keyPath, privateA, publicA)
	})

	t.Run("torn pair", func(t *testing.T) {
		client, _ := newKeysScopeClient(t, map[string]string{"client-private-key": string(privateA), "client-public-key": string(publicB)})
		keyPath := filepath.Join(t.TempDir(), "session1")
		err := saveAuthorizedClientKeys(t.Context(), client, testKeysScope, keyPath, meta, testKeyOpts)
		assert.ErrorIs(t, err, errClientKeysNotAuthorized)
		assert.ErrorIs(t, err, keys.ErrMismatchedKeyPair)
		assert.NoFileExists(t, keyPath)
	})

	t.Run("missing pair", func(t *testing.T) {
		client, _ := newKeysScopeClient(t, nil)
		keyPath := filepath.Join(t.TempDir(), "session1")
		err := saveAuthorizedClientKeys(t.Context(), client, testKeysScope, keyPath, meta, testKeyOpts)
		assert.ErrorIs(t, err, errClientKeyPair)
		assert.NotErrorIs(t, err, errClientKeysNotAuthorized)
		assert.NoFileExists(t, keyPath)
	})
}

func TestProxyCommandPinsOnlyTheNamedServerHostKey(t *testing.T) {
	serverA := newTestServer(t)
	serverB := newTestServer(t)

	t.Run("scope holds the named server's key", func(t *testing.T) {
		client, _, opts, sessionKeyPath := newSessionClient(t)
		startTestServer(t, client, serverA, serverA.clientPrivate, serverA.clientPublic)
		metadata, _, knownHostsPath := connectSession(t, client, opts, sessionKeyPath)
		require.NoError(t, os.WriteFile(knownHostsPath, []byte(testSession+" "+string(serverB.hostKey)), 0o600))

		_, got, reused := proxyCommandKeys(t, client, opts, sessionKeyPath, metadata)
		assert.True(t, reused)
		assert.Equal(t, knownHostsPath, got)
		assertPinned(t, knownHostsPath, serverA.hostKey)
	})

	t.Run("another server is starting", func(t *testing.T) {
		client, _, opts, sessionKeyPath := newSessionClient(t)
		startTestServer(t, client, serverA, serverA.clientPrivate, serverA.clientPublic)
		metadata, _, knownHostsPath := connectSession(t, client, opts, sessionKeyPath)
		// Server B, started for the session at the same time, publishes its key after
		// connect verified server A and before the ProxyCommand it spawned runs. B has not
		// written metadata.json yet, which still names server A.
		require.NoError(t, client.Secrets.PutSecret(t.Context(), workspace.PutSecret{Scope: testKeysScope, Key: "server-public-key", StringValue: string(serverB.hostKey)}))

		_, got, reused := proxyCommandKeys(t, client, opts, sessionKeyPath, metadata)
		assert.True(t, reused)
		assert.Equal(t, knownHostsPath, got)
		// B's key is not pinned for server A, so ssh still accepts A.
		assertPinned(t, knownHostsPath, serverA.hostKey)
	})

	t.Run("another connect pinned the other server's key", func(t *testing.T) {
		client, _, opts, sessionKeyPath := newSessionClient(t)
		startTestServer(t, client, serverA, serverA.clientPrivate, serverA.clientPublic)
		metadata, _, knownHostsPath := connectSession(t, client, opts, sessionKeyPath)
		// Before A's ProxyCommand runs, server B publishes its key, and connects to B pin it:
		// one that spawns ssh in B's own file, and one run by the ssh configuration
		// `databricks ssh setup` writes in the session's file.
		require.NoError(t, client.Secrets.PutSecret(t.Context(), workspace.PutSecret{Scope: testKeysScope, Key: "server-public-key", StringValue: string(serverB.hostKey)}))
		knownHostsPathB, err := pinServerHostKey(t.Context(), testSession, serverB.hostFingerprint, serverB.hostKey, opts)
		require.NoError(t, err)
		sessionKnownHostsPath, err := pinServerHostKey(t.Context(), testSession, "", serverB.hostKey, opts)
		require.NoError(t, err)
		assert.NotEqual(t, knownHostsPath, knownHostsPathB)
		assert.NotEqual(t, knownHostsPath, sessionKnownHostsPath)

		_, got, reused := proxyCommandKeys(t, client, opts, sessionKeyPath, metadata)
		assert.True(t, reused)
		assert.Equal(t, knownHostsPath, got)
		// The file ssh reads for server A still holds only A's key.
		assertPinned(t, knownHostsPath, serverA.hostKey)
		assertPinned(t, knownHostsPathB, serverB.hostKey)
		assertPinned(t, sessionKnownHostsPath, serverB.hostKey)
	})

	t.Run("metadata from an earlier CLI version", func(t *testing.T) {
		client, _, opts, sessionKeyPath := newSessionClient(t)
		startTestServer(t, client, serverB, serverB.clientPrivate, serverB.clientPublic)
		// Its ssh configuration points ssh at the session's file.
		knownHostsPath := filepath.Join(opts.KnownHostsDir, testSession)
		require.NoError(t, os.WriteFile(knownHostsPath, []byte(testSession+" "+string(serverA.hostKey)), 0o600))

		// It names no host key, so the scope's key is pinned unchecked, as that version did.
		_, got, reused := proxyCommandKeys(t, client, opts, sessionKeyPath, "root,7772,"+testServerCluster)
		assert.True(t, reused)
		assert.Equal(t, knownHostsPath, got)
		assertPinned(t, knownHostsPath, serverB.hostKey)
	})
}

func TestProxyCommandUsesOnlyTheNamedServerClientKey(t *testing.T) {
	serverA := newTestServer(t)

	t.Run("another server overwrote the pair after connect", func(t *testing.T) {
		client, requests, opts, sessionKeyPath := newSessionClient(t)
		startTestServer(t, client, serverA, serverA.clientPrivate, serverA.clientPublic)
		metadata, keyPath, _ := connectSession(t, client, opts, sessionKeyPath)
		// A connect that starts server B for the session at the same time overwrites the
		// scope's pair and the session's key file with pair B. B has not written its host key
		// or metadata.json yet.
		require.NoError(t, storeClientKeysForNewServer(t.Context(), client, testKeysScope, sessionKeyPath, opts))
		before := len(requests())

		// The ProxyCommand of a spawned ssh, or of a later IDE reconnect, uses A's file, which
		// still holds the pair server A authorized; the scope's pair is not read.
		got, _, reused := proxyCommandKeys(t, client, opts, sessionKeyPath, metadata)
		assert.True(t, reused)
		assert.Equal(t, keyPath, got)
		assertLocalKeys(t, keyPath, serverA.clientPrivate, serverA.clientPublic)
		// Only the host key is read.
		assert.Equal(t, []string{"GET /api/2.0/secrets/get"}, requests()[before:])
	})

	t.Run("key file missing, scope holds the pair", func(t *testing.T) {
		client, _, opts, sessionKeyPath := newSessionClient(t)
		startTestServer(t, client, serverA, serverA.clientPrivate, serverA.clientPublic)
		metadata, keyPath, _ := connectSession(t, client, opts, sessionKeyPath)
		require.NoError(t, os.Remove(keyPath))

		got, _, reused := proxyCommandKeys(t, client, opts, sessionKeyPath, metadata)
		assert.True(t, reused)
		assert.Equal(t, keyPath, got)
		assertLocalKeys(t, keyPath, serverA.clientPrivate, serverA.clientPublic)
	})

	t.Run("key file missing, another server overwrote the pair", func(t *testing.T) {
		client, _, opts, sessionKeyPath := newSessionClient(t)
		startTestServer(t, client, serverA, serverA.clientPrivate, serverA.clientPublic)
		metadata, keyPath, _ := connectSession(t, client, opts, sessionKeyPath)
		require.NoError(t, storeClientKeysForNewServer(t.Context(), client, testKeysScope, sessionKeyPath, opts))
		_, publicB := scopeClientKeys(t, client)
		require.NoError(t, os.Remove(keyPath))
		require.NoError(t, os.Remove(keyPath+".pub"))

		// Pair B is not used for server A, which would reject it. The server is replaced
		// instead, and the pair it authorized is saved in the file ssh reads.
		got, _, reused := proxyCommandKeys(t, client, opts, sessionKeyPath, metadata)
		assert.False(t, reused)
		assert.Equal(t, keyPath, got)
		privateKey, publicKey := scopeClientKeys(t, client)
		assert.NotEqual(t, serverA.clientPublic, publicKey)
		assert.NotEqual(t, publicB, publicKey)
		assertLocalKeys(t, keyPath, privateKey, publicKey)
	})

	t.Run("metadata from an earlier CLI version", func(t *testing.T) {
		client, _, opts, sessionKeyPath := newSessionClient(t)
		startTestServer(t, client, serverA, serverA.clientPrivate, serverA.clientPublic)

		// It names no client key, and its ssh configuration points ssh at the session's file.
		got, _, reused := proxyCommandKeys(t, client, opts, sessionKeyPath, "root,7772,"+testServerCluster+","+serverA.hostFingerprint)
		assert.True(t, reused)
		assert.Equal(t, sessionKeyPath, got)
		assertLocalKeys(t, sessionKeyPath, serverA.clientPrivate, serverA.clientPublic)
	})
}

func TestProxyCommandAfterTheNamedServerWasReplaced(t *testing.T) {
	serverA := newTestServer(t)
	serverB := newTestServer(t)

	for _, tc := range []struct {
		name string
		// metadata returns the --metadata value of the ProxyCommand, given the one the
		// current CLI version writes for server A.
		metadata func(current string) string
		// legacy is set for a ProxyCommand written by an earlier CLI version, which points
		// ssh at the session's files.
		legacy bool
	}{
		{name: "current metadata", metadata: func(current string) string { return current }},
		{name: "metadata from an earlier CLI version", metadata: func(string) string { return "root,7772," + testServerCluster }, legacy: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, _, opts, sessionKeyPath := newSessionClient(t)
			startTestServer(t, client, serverA, serverA.clientPrivate, serverA.clientPublic)
			current, keyPath, knownHostsPath := connectSession(t, client, opts, sessionKeyPath)
			metadata := tc.metadata(current)
			if tc.legacy {
				keyPath = sessionKeyPath
				knownHostsPath = filepath.Join(opts.KnownHostsDir, testSession)
			}
			// Another connect replaces server A with server B on the same port.
			startTestServer(t, client, serverB, serverB.clientPrivate, serverB.clientPublic)

			// The files ssh reads for the ProxyCommand get server B's keys, so the
			// connection succeeds.
			gotKeyPath, gotKnownHostsPath, _ := proxyCommandKeys(t, client, opts, sessionKeyPath, metadata)
			assert.Equal(t, keyPath, gotKeyPath)
			assert.Equal(t, knownHostsPath, gotKnownHostsPath)
			assertLocalKeys(t, keyPath, serverB.clientPrivate, serverB.clientPublic)
			assertPinned(t, knownHostsPath, serverB.hostKey)

			// A later reconnect with the same ProxyCommand reuses server B with those files.
			gotKeyPath, gotKnownHostsPath, reused := proxyCommandKeys(t, client, opts, sessionKeyPath, metadata)
			assert.True(t, reused)
			assert.Equal(t, keyPath, gotKeyPath)
			assert.Equal(t, knownHostsPath, gotKnownHostsPath)
			assertLocalKeys(t, keyPath, serverB.clientPrivate, serverB.clientPublic)
			assertPinned(t, knownHostsPath, serverB.hostKey)
		})
	}
}

func TestParseServerMetadata(t *testing.T) {
	for _, tc := range []struct {
		metadata          string
		cluster           string
		fingerprint       string
		clientFingerprint string
	}{
		{metadata: FormatMetadata("root", 7772, "cluster-1", "SHA256:abc+/d", "SHA256:client+/e"), cluster: "cluster-1", fingerprint: "SHA256:abc+/d", clientFingerprint: "SHA256:client+/e"},
		{metadata: FormatMetadata("root", 7772, "", "SHA256:abc", "SHA256:client"), cluster: "default-cluster", fingerprint: "SHA256:abc", clientFingerprint: "SHA256:client"},
		// Written by earlier CLI versions.
		{metadata: "root,7772,cluster-1,SHA256:abc", cluster: "cluster-1", fingerprint: "SHA256:abc"},
		{metadata: "root,7772,cluster-1", cluster: "cluster-1"},
		{metadata: "root,7772", cluster: "default-cluster"},
	} {
		t.Run(tc.metadata, func(t *testing.T) {
			userName, port, clusterID, fingerprint, clientFingerprint, err := parseServerMetadata(tc.metadata, "default-cluster")
			require.NoError(t, err)
			assert.Equal(t, "root", userName)
			assert.Equal(t, 7772, port)
			assert.Equal(t, tc.cluster, clusterID)
			assert.Equal(t, tc.fingerprint, fingerprint)
			assert.Equal(t, tc.clientFingerprint, clientFingerprint)
		})
	}
}
