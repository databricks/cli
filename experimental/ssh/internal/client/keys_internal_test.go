package client

import (
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

func TestEnsureSSHServerIsRunningChecksTheServerAuthorizedTheScopeKeys(t *testing.T) {
	privateA, publicA, err := keys.GenerateSSHKeyPair()
	require.NoError(t, err)
	privateB, publicB, err := keys.GenerateSSHKeyPair()
	require.NoError(t, err)
	_, hostKeyA, err := keys.GenerateSSHKeyPair()
	require.NoError(t, err)
	hostFingerprintA, err := keys.PublicKeyFingerprint(hostKeyA)
	require.NoError(t, err)
	clientFingerprintA, err := keys.PublicKeyFingerprint(publicA)
	require.NoError(t, err)

	// startServerA sets up server A, healthy and named by metadata.json, which authorized
	// client key A, with the scope holding the given client key pair.
	startServerA := func(t *testing.T, privateKey, publicKey []byte) (*databricks.WorkspaceClient, ClientOptions, string) {
		client, _ := newKeysScopeClient(t, map[string]string{
			"server-public-key":  string(hostKeyA),
			"client-private-key": string(privateKey),
			"client-public-key":  string(publicKey),
		})
		ctx := cmdio.MockDiscard(t.Context())
		wsHome, err := sshWorkspace.GetWorkspaceHomeDir(ctx, client)
		require.NoError(t, err)
		require.NoError(t, sshWorkspace.SaveWorkspaceMetadata(ctx, client, wsHome, "test-version", "session1", &sshWorkspace.WorkspaceMetadata{
			Port:                 7772,
			ClusterID:            "1234-567890-serverless",
			HostKeyFingerprint:   hostFingerprintA,
			ClientKeyFingerprint: clientFingerprintA,
		}))
		opts := testKeyOpts
		opts.ConnectionName = "session1"
		opts.ServerPublicKeyName = "server-public-key"
		opts.SSHKeysDir = t.TempDir()
		return client, opts, filepath.Join(opts.SSHKeysDir, "session1")
	}

	t.Run("scope holds the pair the server authorized", func(t *testing.T) {
		client, opts, keyPath := startServerA(t, privateA, publicA)
		meta, authorizedKeyPath, err := ensureSSHServerIsRunning(cmdio.MockDiscard(t.Context()), client, "test-version", testKeysScope, keyPath, opts)
		require.NoError(t, err)

		// Server A is reused with the pair it authorized, saved in A's own file.
		assert.Equal(t, hostKeyA, meta.HostKey)
		wantKeyPath, err := keys.GetLocalServerSSHKeyPath(t.Context(), "session1", clientFingerprintA, opts.SSHKeysDir)
		require.NoError(t, err)
		assert.Equal(t, wantKeyPath, authorizedKeyPath)
		assertLocalKeys(t, authorizedKeyPath, privateA, publicA)
	})

	t.Run("scope holds another complete pair", func(t *testing.T) {
		// Server B, started at the same time, wrote both halves of pair B after server A read
		// public key A, and A's host key and metadata.json landed after B's.
		client, opts, keyPath := startServerA(t, privateB, publicB)
		meta, authorizedKeyPath, err := ensureSSHServerIsRunning(cmdio.MockDiscard(t.Context()), client, "test-version", testKeysScope, keyPath, opts)
		require.NoError(t, err)

		// Server A would reject pair B, so it is replaced rather than reused with pair B. The
		// test server stands in for the new server and publishes its own host key.
		assert.NotEqual(t, hostKeyA, meta.HostKey)
		privateKey, publicKey := scopeClientKeys(t, client)
		assert.NotEqual(t, publicA, publicKey)
		assert.NotEqual(t, publicB, publicKey)
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
	_, hostKeyA, err := keys.GenerateSSHKeyPair()
	require.NoError(t, err)
	_, hostKeyB, err := keys.GenerateSSHKeyPair()
	require.NoError(t, err)
	fingerprintA, err := keys.PublicKeyFingerprint(hostKeyA)
	require.NoError(t, err)
	fingerprintB, err := keys.PublicKeyFingerprint(hostKeyB)
	require.NoError(t, err)

	// connect returns the ProxyCommand metadata for server A, and the known_hosts file it
	// pinned A's key in and points ssh at.
	outerConnect := func(t *testing.T, client *databricks.WorkspaceClient, opts ClientOptions) (string, string) {
		hostKey, err := serverHostKeyNamedBy(t.Context(), client, testKeysScope, &sshWorkspace.WorkspaceMetadata{Port: 7772, HostKeyFingerprint: fingerprintA}, opts)
		require.NoError(t, err)
		knownHostsPath, err := pinServerHostKey(t.Context(), "session1", fingerprintA, hostKey, opts)
		require.NoError(t, err)
		return FormatMetadata("root", 7772, "cluster-1", fingerprintA, "SHA256:client"), knownHostsPath
	}
	// proxyCommand does what the ProxyCommand that connect spawns does before the tunnel
	// carries a byte: pin the host key of the server its --metadata names. It returns the
	// file it pinned the key in.
	proxyCommand := func(t *testing.T, client *databricks.WorkspaceClient, opts ClientOptions, metadata string) string {
		_, port, _, fingerprint, _, err := parseServerMetadata(metadata, "")
		require.NoError(t, err)
		hostKey, err := namedServerHostKey(t.Context(), client, testKeysScope, port, fingerprint, opts)
		require.NoError(t, err)
		knownHostsPath, err := pinServerHostKey(t.Context(), "session1", fingerprint, hostKey, opts)
		require.NoError(t, err)
		return knownHostsPath
	}
	assertPinned := func(t *testing.T, knownHostsPath string, hostKey []byte) {
		pinned, err := os.ReadFile(knownHostsPath)
		require.NoError(t, err)
		assert.Equal(t, "session1 "+string(hostKey), string(pinned))
	}
	newOpts := func(t *testing.T) ClientOptions {
		return ClientOptions{ServerPublicKeyName: "server-public-key", KnownHostsDir: t.TempDir()}
	}

	t.Run("scope holds the named server's key", func(t *testing.T) {
		opts := newOpts(t)
		client, _ := newKeysScopeClient(t, map[string]string{"server-public-key": string(hostKeyA)})
		metadata, knownHostsPath := outerConnect(t, client, opts)
		require.NoError(t, os.WriteFile(knownHostsPath, []byte("session1 "+string(hostKeyB)), 0o600))

		assert.Equal(t, knownHostsPath, proxyCommand(t, client, opts, metadata))
		assertPinned(t, knownHostsPath, hostKeyA)
	})

	t.Run("another server replaced the key after connect", func(t *testing.T) {
		opts := newOpts(t)
		client, _ := newKeysScopeClient(t, map[string]string{"server-public-key": string(hostKeyA)})
		metadata, knownHostsPath := outerConnect(t, client, opts)
		// Server B, started for the session at the same time, publishes its key after
		// connect verified server A and before the ProxyCommand it spawned runs.
		require.NoError(t, client.Secrets.PutSecret(t.Context(), workspace.PutSecret{Scope: testKeysScope, Key: "server-public-key", StringValue: string(hostKeyB)}))

		assert.Equal(t, knownHostsPath, proxyCommand(t, client, opts, metadata))
		// B's key is not pinned for server A, so ssh still accepts A.
		assertPinned(t, knownHostsPath, hostKeyA)
	})

	t.Run("another connect pinned the other server's key", func(t *testing.T) {
		opts := newOpts(t)
		client, _ := newKeysScopeClient(t, map[string]string{"server-public-key": string(hostKeyA)})
		metadata, knownHostsPath := outerConnect(t, client, opts)
		// Before A's ProxyCommand runs, server B publishes its key, and connects to B pin it:
		// one that spawns ssh in B's own file, and one run by the ssh configuration
		// `databricks ssh setup` writes in the session's file.
		require.NoError(t, client.Secrets.PutSecret(t.Context(), workspace.PutSecret{Scope: testKeysScope, Key: "server-public-key", StringValue: string(hostKeyB)}))
		knownHostsPathB, err := pinServerHostKey(t.Context(), "session1", fingerprintB, hostKeyB, opts)
		require.NoError(t, err)
		sessionKnownHostsPath, err := pinServerHostKey(t.Context(), "session1", "", hostKeyB, opts)
		require.NoError(t, err)
		assert.NotEqual(t, knownHostsPath, knownHostsPathB)
		assert.NotEqual(t, knownHostsPath, sessionKnownHostsPath)

		assert.Equal(t, knownHostsPath, proxyCommand(t, client, opts, metadata))
		// The file ssh reads for server A still holds only A's key.
		assertPinned(t, knownHostsPath, hostKeyA)
		assertPinned(t, knownHostsPathB, hostKeyB)
		assertPinned(t, sessionKnownHostsPath, hostKeyB)
	})

	t.Run("metadata from an earlier CLI version", func(t *testing.T) {
		opts := newOpts(t)
		client, requests := newKeysScopeClient(t, map[string]string{"server-public-key": string(hostKeyB)})
		// Its ssh configuration points ssh at the session's file.
		knownHostsPath := filepath.Join(opts.KnownHostsDir, "session1")
		require.NoError(t, os.WriteFile(knownHostsPath, []byte("session1 "+string(hostKeyA)), 0o600))

		// It names no host key, so the scope's key, which may be another server's, is not read.
		assert.Equal(t, knownHostsPath, proxyCommand(t, client, opts, "root,7772,cluster-1"))
		assert.Empty(t, requests())
		assertPinned(t, knownHostsPath, hostKeyA)
	})
}

func TestProxyCommandUsesOnlyTheNamedServerClientKey(t *testing.T) {
	privateA, publicA, err := keys.GenerateSSHKeyPair()
	require.NoError(t, err)
	_, hostKeyA, err := keys.GenerateSSHKeyPair()
	require.NoError(t, err)
	hostFingerprintA, err := keys.PublicKeyFingerprint(hostKeyA)
	require.NoError(t, err)
	clientFingerprintA, err := keys.PublicKeyFingerprint(publicA)
	require.NoError(t, err)

	// startServerA sets up server A, healthy and named by metadata.json, which authorized
	// client key A, the pair the scope holds.
	startServerA := func(t *testing.T) (*databricks.WorkspaceClient, func() []string, ClientOptions, string) {
		client, requests := newKeysScopeClient(t, map[string]string{
			"server-public-key":  string(hostKeyA),
			"client-private-key": string(privateA),
			"client-public-key":  string(publicA),
		})
		ctx := cmdio.MockDiscard(t.Context())
		wsHome, err := sshWorkspace.GetWorkspaceHomeDir(ctx, client)
		require.NoError(t, err)
		require.NoError(t, sshWorkspace.SaveWorkspaceMetadata(ctx, client, wsHome, "test-version", "session1", &sshWorkspace.WorkspaceMetadata{
			Port:                 7772,
			ClusterID:            "1234-567890-serverless",
			HostKeyFingerprint:   hostFingerprintA,
			ClientKeyFingerprint: clientFingerprintA,
		}))
		opts := testKeyOpts
		opts.ConnectionName = "session1"
		opts.ServerPublicKeyName = "server-public-key"
		opts.SSHKeysDir = t.TempDir()
		return client, requests, opts, filepath.Join(opts.SSHKeysDir, "session1")
	}
	// outerConnect selects server A the way a connect that spawns ssh or launches an IDE does.
	// It returns the ProxyCommand metadata it writes and the key file it points ssh at.
	outerConnect := func(t *testing.T, client *databricks.WorkspaceClient, opts ClientOptions, sessionKeyPath string) (string, string) {
		meta, keyPath, err := ensureSSHServerIsRunning(cmdio.MockDiscard(t.Context()), client, "test-version", testKeysScope, sessionKeyPath, opts)
		require.NoError(t, err)
		require.Equal(t, clientFingerprintA, meta.ClientKeyFingerprint)
		assertLocalKeys(t, keyPath, privateA, publicA)
		return FormatMetadata(meta.UserName, meta.Port, meta.ClusterID, meta.HostKeyFingerprint, meta.ClientKeyFingerprint), keyPath
	}
	// startServerB does what a connect that starts server B for the session at the same time
	// does: it overwrites the scope's pair and the session's key file with pair B, and saves
	// pair B in the file of B's own.
	startServerB := func(t *testing.T, client *databricks.WorkspaceClient, opts ClientOptions, sessionKeyPath string) string {
		require.NoError(t, storeClientKeysForNewServer(t.Context(), client, testKeysScope, sessionKeyPath, opts))
		_, publicB := scopeClientKeys(t, client)
		fingerprintB, err := keys.PublicKeyFingerprint(publicB)
		require.NoError(t, err)
		keyPathB, err := authorizedClientKeyPath(t.Context(), sessionKeyPath, serverMetadata{ClientKeyFingerprint: fingerprintB}, opts)
		require.NoError(t, err)
		require.NoError(t, saveAuthorizedClientKeys(t.Context(), client, testKeysScope, keyPathB, serverMetadata{ClientKeyFingerprint: fingerprintB}, opts))
		return keyPathB
	}
	// proxyCommand does what the ProxyCommand that ssh or the IDE runs for server A does
	// with the client key before the tunnel carries a byte.
	proxyCommand := func(t *testing.T, client *databricks.WorkspaceClient, opts ClientOptions, sessionKeyPath, metadata string) (string, error) {
		_, port, _, _, clientFingerprint, err := parseServerMetadata(metadata, "")
		require.NoError(t, err)
		opts.ProxyMode = true
		return namedServerClientKeys(t.Context(), client, testKeysScope, sessionKeyPath, port, clientFingerprint, opts)
	}

	t.Run("another server overwrote the pair after connect", func(t *testing.T) {
		client, requests, opts, sessionKeyPath := startServerA(t)
		metadata, keyPath := outerConnect(t, client, opts, sessionKeyPath)
		keyPathB := startServerB(t, client, opts, sessionKeyPath)
		assert.NotEqual(t, keyPath, keyPathB)
		before := len(requests())

		// The ProxyCommand of a spawned ssh, or of a later IDE reconnect, uses A's file, which
		// still holds the pair server A authorized; the scope is not read.
		got, err := proxyCommand(t, client, opts, sessionKeyPath, metadata)
		require.NoError(t, err)
		assert.Equal(t, keyPath, got)
		assertLocalKeys(t, keyPath, privateA, publicA)
		assert.Len(t, requests(), before)
	})

	t.Run("key file missing, scope holds the pair", func(t *testing.T) {
		client, _, opts, sessionKeyPath := startServerA(t)
		metadata, keyPath := outerConnect(t, client, opts, sessionKeyPath)
		require.NoError(t, os.Remove(keyPath))

		got, err := proxyCommand(t, client, opts, sessionKeyPath, metadata)
		require.NoError(t, err)
		assert.Equal(t, keyPath, got)
		assertLocalKeys(t, keyPath, privateA, publicA)
	})

	t.Run("key file missing, another server overwrote the pair", func(t *testing.T) {
		client, _, opts, sessionKeyPath := startServerA(t)
		metadata, keyPath := outerConnect(t, client, opts, sessionKeyPath)
		startServerB(t, client, opts, sessionKeyPath)
		require.NoError(t, os.Remove(keyPath))
		require.NoError(t, os.Remove(keyPath+".pub"))

		// Pair B is not saved for server A, which would reject it. The caller finds or
		// replaces the server instead.
		got, err := proxyCommand(t, client, opts, sessionKeyPath, metadata)
		assert.ErrorIs(t, err, errClientKeysNotAuthorized)
		assert.Equal(t, keyPath, got)
		assert.NoFileExists(t, keyPath)
	})

	t.Run("metadata from an earlier CLI version", func(t *testing.T) {
		client, _, opts, sessionKeyPath := startServerA(t)

		// It names no client key, and its ssh configuration points ssh at the session's file.
		got, err := proxyCommand(t, client, opts, sessionKeyPath, "root,7772,cluster-1,"+hostFingerprintA)
		require.NoError(t, err)
		assert.Equal(t, sessionKeyPath, got)
		assertLocalKeys(t, sessionKeyPath, privateA, publicA)
	})
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
