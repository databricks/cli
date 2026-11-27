package keys_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/databricks/cli/experimental/ssh/internal/keys"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestGetLocalServerSSHKeyPath(t *testing.T) {
	dir := filepath.Join("/home/me", "keys")
	pathA, err := keys.GetLocalServerSSHKeyPath(t.Context(), "session1", "SHA256:abc+/../d", dir)
	require.NoError(t, err)
	pathB, err := keys.GetLocalServerSSHKeyPath(t.Context(), "session1", "SHA256:other", dir)
	require.NoError(t, err)
	again, err := keys.GetLocalServerSSHKeyPath(t.Context(), "session1", "SHA256:abc+/../d", dir)
	require.NoError(t, err)
	sessionPath, err := keys.GetLocalSSHKeyPath(t.Context(), "session1", dir)
	require.NoError(t, err)

	// Each server of the session has a key file of its own next to the session's, whatever
	// characters its fingerprint holds.
	assert.Equal(t, dir, filepath.Dir(pathA))
	assert.Regexp(t, `^session1\.[0-9a-f]{16}$`, filepath.Base(pathA))
	assert.NotEqual(t, pathA, pathB)
	assert.NotEqual(t, sessionPath, pathA)
	assert.Equal(t, pathA, again)
}

func TestRemoveOtherServerFiles(t *testing.T) {
	dir := t.TempDir()
	keep, err := keys.GetLocalServerSSHKeyPath(t.Context(), "session1", "SHA256:kept", dir)
	require.NoError(t, err)
	other, err := keys.GetLocalServerSSHKeyPath(t.Context(), "session1", "SHA256:other", dir)
	require.NoError(t, err)
	otherSession, err := keys.GetLocalServerSSHKeyPath(t.Context(), "session10", "SHA256:other", dir)
	require.NoError(t, err)
	sessionPath, err := keys.GetLocalSSHKeyPath(t.Context(), "session1", dir)
	require.NoError(t, err)
	for _, path := range []string{keep, other, otherSession, sessionPath} {
		require.NoError(t, keys.SaveSSHKeyPair(path, []byte("private"), []byte("public")))
	}

	require.NoError(t, keys.RemoveOtherServerFiles("session1", keep))

	// Only the files of the session's other servers are removed: the kept server's, the
	// session's own and another session's stay.
	assert.NoFileExists(t, other)
	assert.NoFileExists(t, other+".pub")
	for _, path := range []string{keep, otherSession, sessionPath} {
		assert.FileExists(t, path)
		assert.FileExists(t, path+".pub")
	}
}

func TestSaveSSHKeyPairNewFiles(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "session1")
	privateKey := []byte("private-key-content")
	publicKey := []byte("public-key-content")

	err := keys.SaveSSHKeyPair(keyPath, privateKey, publicKey)
	require.NoError(t, err)

	gotPrivate, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	assert.Equal(t, privateKey, gotPrivate)

	gotPublic, err := os.ReadFile(keyPath + ".pub")
	require.NoError(t, err)
	assert.Equal(t, publicKey, gotPublic)
}

func TestSaveSSHKeyPairOverwritesExistingFiles(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "session1")

	// Write initial keys.
	require.NoError(t, keys.SaveSSHKeyPair(keyPath, []byte("old-private"), []byte("old-public")))

	// Overwrite with new keys.
	newPrivate := []byte("new-private-key-content")
	newPublic := []byte("new-public-key-content")
	err := keys.SaveSSHKeyPair(keyPath, newPrivate, newPublic)
	require.NoError(t, err)

	gotPrivate, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	assert.Equal(t, newPrivate, gotPrivate)

	gotPublic, err := os.ReadFile(keyPath + ".pub")
	require.NoError(t, err)
	assert.Equal(t, newPublic, gotPublic)
}

func TestSaveSSHKeyPairCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "nonexistent-subdir", "session1")
	privateKey := []byte("private-key-content")
	publicKey := []byte("public-key-content")

	err := keys.SaveSSHKeyPair(keyPath, privateKey, publicKey)
	require.NoError(t, err)

	gotPrivate, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	assert.Equal(t, privateKey, gotPrivate)

	gotPublic, err := os.ReadFile(keyPath + ".pub")
	require.NoError(t, err)
	assert.Equal(t, publicKey, gotPublic)
}

func TestSaveSSHKeyPairDoesNotAffectOtherSessions(t *testing.T) {
	dir := t.TempDir()
	keyPath1 := filepath.Join(dir, "session1")
	keyPath2 := filepath.Join(dir, "session2")

	require.NoError(t, keys.SaveSSHKeyPair(keyPath1, []byte("private-1"), []byte("public-1")))
	require.NoError(t, keys.SaveSSHKeyPair(keyPath2, []byte("private-2"), []byte("public-2")))

	// Overwrite session1 — session2 must be untouched.
	require.NoError(t, keys.SaveSSHKeyPair(keyPath1, []byte("private-1-new"), []byte("public-1-new")))

	gotPrivate2, err := os.ReadFile(keyPath2)
	require.NoError(t, err)
	assert.Equal(t, []byte("private-2"), gotPrivate2)

	gotPublic2, err := os.ReadFile(keyPath2 + ".pub")
	require.NoError(t, err)
	assert.Equal(t, []byte("public-2"), gotPublic2)
}

func TestLoadSSHKeyPairDerivesPublicKeyFromPrivateKey(t *testing.T) {
	privateKey, publicKey, err := keys.GenerateSSHKeyPair()
	require.NoError(t, err)
	keyPath := filepath.Join(t.TempDir(), "session1")
	// A .pub file carrying authorized_keys options must not leak into the returned key.
	require.NoError(t, keys.SaveSSHKeyPair(keyPath, privateKey, append([]byte(`command="echo hi" `), publicKey...)))

	gotPrivate, gotPublic, err := keys.LoadSSHKeyPair(keyPath)
	require.NoError(t, err)
	assert.Equal(t, privateKey, gotPrivate)
	assert.Equal(t, publicKey, gotPublic)
}

func TestLoadSSHKeyPairMissing(t *testing.T) {
	_, _, err := keys.LoadSSHKeyPair(filepath.Join(t.TempDir(), "session1"))
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

const testScope = "tester-session1-ssh-tunnel-keys"

// newSecretsClient returns a client for a fake workspace that has testScope, holding the
// given secrets, and a function listing the secret requests made after setup.
func newSecretsClient(t *testing.T, secrets map[string]string) (*databricks.WorkspaceClient, func() []string) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)

	ctx := t.Context()
	require.NoError(t, client.Secrets.CreateScope(ctx, workspace.CreateScope{Scope: testScope}))
	for key, value := range secrets {
		require.NoError(t, client.Secrets.PutSecret(ctx, workspace.PutSecret{Scope: testScope, Key: key, StringValue: value}))
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

func TestGenerateSSHKeyPairToSecretsOverwritesWithoutReading(t *testing.T) {
	client, requests := newSecretsClient(t, map[string]string{
		"private": "existing-private",
		"public":  "existing-public",
	})

	privateKey, publicKey, err := keys.GenerateSSHKeyPairToSecrets(t.Context(), client, testScope, "private", "public")
	require.NoError(t, err)
	assert.Equal(t, []string{"POST /api/2.0/secrets/put", "POST /api/2.0/secrets/put"}, requests())

	signer, err := ssh.ParsePrivateKey(privateKey)
	require.NoError(t, err)
	assert.Equal(t, ssh.MarshalAuthorizedKey(signer.PublicKey()), publicKey)

	gotPrivate, gotPublic, err := keys.GetSSHKeyPairFromSecrets(t.Context(), client, testScope, "private", "public")
	require.NoError(t, err)
	assert.Equal(t, privateKey, gotPrivate)
	assert.Equal(t, publicKey, gotPublic)
}

func TestGetSSHKeyPairFromSecretsFailsWhenMissing(t *testing.T) {
	client, requests := newSecretsClient(t, map[string]string{"private": "existing-private"})

	_, _, err := keys.GetSSHKeyPairFromSecrets(t.Context(), client, testScope, "private", "public")
	assert.ErrorContains(t, err, "failed to get public key from secrets scope")
	// A missing key is reported, not replaced by a new pair.
	assert.NotContains(t, requests(), "POST /api/2.0/secrets/put")
}

func TestGetSSHKeyPairFromSecretsDerivesPublicKey(t *testing.T) {
	privateKey, publicKey, err := keys.GenerateSSHKeyPair()
	require.NoError(t, err)
	// The public secret's comment is not part of the key.
	client, _ := newSecretsClient(t, map[string]string{
		"private": string(privateKey),
		"public":  strings.TrimSpace(string(publicKey)) + " comment",
	})

	gotPrivate, gotPublic, err := keys.GetSSHKeyPairFromSecrets(t.Context(), client, testScope, "private", "public")
	require.NoError(t, err)
	assert.Equal(t, privateKey, gotPrivate)
	assert.Equal(t, publicKey, gotPublic)
}

func TestGetSSHKeyPairFromSecretsRejectsMismatchedHalves(t *testing.T) {
	privateA, publicA, err := keys.GenerateSSHKeyPair()
	require.NoError(t, err)
	privateB, publicB, err := keys.GenerateSSHKeyPair()
	require.NoError(t, err)

	for name, secrets := range map[string]map[string]string{
		// B's private key replaced A's, and B's public key write failed or A's landed after it.
		"new private key, old public key": {"private": string(privateB), "public": string(publicA)},
		// B wrote its pair, then A's private key landed on top of it.
		"old private key, new public key": {"private": string(privateA), "public": string(publicB)},
	} {
		t.Run(name, func(t *testing.T) {
			client, _ := newSecretsClient(t, secrets)
			_, _, err := keys.GetSSHKeyPairFromSecrets(t.Context(), client, testScope, "private", "public")
			assert.ErrorIs(t, err, keys.ErrMismatchedKeyPair)
			assert.ErrorContains(t, err, "secrets private and public of secret scope "+testScope+" hold halves of different key pairs")
		})
	}
}

func TestPutSSHKeyPairToSecretsInterleavedWritersAreDetected(t *testing.T) {
	client, _ := newSecretsClient(t, nil)
	ctx := t.Context()
	privateA, publicA, err := keys.GenerateSSHKeyPair()
	require.NoError(t, err)
	privateB, publicB, err := keys.GenerateSSHKeyPair()
	require.NoError(t, err)

	// Writer A's pair lands, then writer B's private key, and B fails before its public key.
	require.NoError(t, keys.PutSSHKeyPairToSecrets(ctx, client, testScope, "private", "public", privateA, publicA))
	require.NoError(t, client.Secrets.PutSecret(ctx, workspace.PutSecret{Scope: testScope, Key: "private", StringValue: string(privateB)}))
	_, _, err = keys.GetSSHKeyPairFromSecrets(ctx, client, testScope, "private", "public")
	assert.ErrorContains(t, err, "hold halves of different key pairs")

	// A complete write makes the pair consistent again.
	require.NoError(t, keys.PutSSHKeyPairToSecrets(ctx, client, testScope, "private", "public", privateB, publicB))
	gotPrivate, gotPublic, err := keys.GetSSHKeyPairFromSecrets(ctx, client, testScope, "private", "public")
	require.NoError(t, err)
	assert.Equal(t, privateB, gotPrivate)
	assert.Equal(t, publicB, gotPublic)
}
