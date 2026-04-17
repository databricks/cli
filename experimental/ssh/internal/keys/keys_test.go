package keys

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestGetLocalSSHKeyPath_DefaultDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("USERPROFILE", tmpDir)

	path, err := GetLocalSSHKeyPath(t.Context(), "cluster-123", "")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmpDir, ".databricks", "ssh-tunnel-keys", "cluster-123"), path)
}

func TestGetLocalSSHKeyPath_CustomDir(t *testing.T) {
	customDir := "/custom/keys/dir"
	path, err := GetLocalSSHKeyPath(t.Context(), "my-session", customDir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(customDir, "my-session"), path)
}

func TestSaveSSHKeyPair(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "session-1", "id_rsa")

	privateKey := []byte("fake-private-key")
	publicKey := []byte("fake-public-key")

	err := SaveSSHKeyPair(keyPath, privateKey, publicKey)
	require.NoError(t, err)

	// Verify private key
	privData, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	assert.Equal(t, privateKey, privData)

	privInfo, err := os.Stat(keyPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), privInfo.Mode().Perm())

	// Verify public key
	pubData, err := os.ReadFile(keyPath + ".pub")
	require.NoError(t, err)
	assert.Equal(t, publicKey, pubData)

	pubInfo, err := os.Stat(keyPath + ".pub")
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), pubInfo.Mode().Perm())
}

func TestSaveSSHKeyPair_OverwritesExisting(t *testing.T) {
	tmpDir := t.TempDir()
	keyDir := filepath.Join(tmpDir, "session-1")
	keyPath := filepath.Join(keyDir, "id_rsa")

	// Create existing keys
	require.NoError(t, os.MkdirAll(keyDir, 0o700))
	require.NoError(t, os.WriteFile(keyPath, []byte("old-private"), 0o600))
	require.NoError(t, os.WriteFile(keyPath+".pub", []byte("old-public"), 0o644))

	err := SaveSSHKeyPair(keyPath, []byte("new-private"), []byte("new-public"))
	require.NoError(t, err)

	privData, err := os.ReadFile(keyPath)
	require.NoError(t, err)
	assert.Equal(t, []byte("new-private"), privData)

	pubData, err := os.ReadFile(keyPath + ".pub")
	require.NoError(t, err)
	assert.Equal(t, []byte("new-public"), pubData)
}

func TestGenerateSSHKeyPair(t *testing.T) {
	privateKeyBytes, publicKeyBytes, err := generateSSHKeyPair()
	require.NoError(t, err)

	// Verify private key is valid PEM-encoded RSA key
	block, _ := pem.Decode(privateKeyBytes)
	require.NotNil(t, block, "expected PEM block for private key")
	assert.Equal(t, "RSA PRIVATE KEY", block.Type)

	_, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	require.NoError(t, err)

	// Verify public key is in authorized_keys format
	assert.Contains(t, string(publicKeyBytes), "ssh-rsa ")
}

func TestCreateKeysSecretScope_ScopeAlreadyExists(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)

	currentUserAPI := m.GetMockCurrentUserAPI()
	currentUserAPI.EXPECT().Me(ctx).Return(&iam.User{UserName: "testuser@example.com"}, nil)

	secretsAPI := m.GetMockSecretsAPI()
	secretsAPI.EXPECT().ListSecretsByScope(ctx, "testuser@example.com-cluster-123-ssh-tunnel-keys").
		Return(&workspace.ListSecretsResponse{}, nil)

	scopeName, err := CreateKeysSecretScope(ctx, m.WorkspaceClient, "cluster-123")
	require.NoError(t, err)
	assert.Equal(t, "testuser@example.com-cluster-123-ssh-tunnel-keys", scopeName)
}

func TestCreateKeysSecretScope_ScopeDoesNotExist(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)

	currentUserAPI := m.GetMockCurrentUserAPI()
	currentUserAPI.EXPECT().Me(ctx).Return(&iam.User{UserName: "testuser@example.com"}, nil)

	scopeName := "testuser@example.com-my-conn-ssh-tunnel-keys"

	secretsAPI := m.GetMockSecretsAPI()
	secretsAPI.EXPECT().ListSecretsByScope(ctx, scopeName).
		Return(nil, databricks.ErrResourceDoesNotExist)
	secretsAPI.EXPECT().CreateScope(ctx, workspace.CreateScope{Scope: scopeName}).
		Return(nil)

	result, err := CreateKeysSecretScope(ctx, m.WorkspaceClient, "my-conn")
	require.NoError(t, err)
	assert.Equal(t, scopeName, result)
}

func TestCreateKeysSecretScope_ListError(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)

	currentUserAPI := m.GetMockCurrentUserAPI()
	currentUserAPI.EXPECT().Me(ctx).Return(&iam.User{UserName: "testuser@example.com"}, nil)

	secretsAPI := m.GetMockSecretsAPI()
	secretsAPI.EXPECT().ListSecretsByScope(ctx, "testuser@example.com-cluster-123-ssh-tunnel-keys").
		Return(nil, errors.New("permission denied"))

	_, err := CreateKeysSecretScope(ctx, m.WorkspaceClient, "cluster-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check if secret scope")
	assert.Contains(t, err.Error(), "permission denied")
}

func TestCreateKeysSecretScope_CreateError(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)

	currentUserAPI := m.GetMockCurrentUserAPI()
	currentUserAPI.EXPECT().Me(ctx).Return(&iam.User{UserName: "testuser@example.com"}, nil)

	scopeName := "testuser@example.com-cluster-123-ssh-tunnel-keys"

	secretsAPI := m.GetMockSecretsAPI()
	secretsAPI.EXPECT().ListSecretsByScope(ctx, scopeName).
		Return(nil, databricks.ErrResourceDoesNotExist)
	secretsAPI.EXPECT().CreateScope(ctx, workspace.CreateScope{Scope: scopeName}).
		Return(errors.New("limit exceeded"))

	_, err := CreateKeysSecretScope(ctx, m.WorkspaceClient, "cluster-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create secrets scope")
	assert.Contains(t, err.Error(), "limit exceeded")
}

func TestGetSecret_DecodesBase64(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)

	encoded := base64.StdEncoding.EncodeToString([]byte("my-secret-value"))

	secretsAPI := m.GetMockSecretsAPI()
	secretsAPI.EXPECT().GetSecret(ctx, workspace.GetSecretRequest{
		Scope: "my-scope",
		Key:   "my-key",
	}).Return(&workspace.GetSecretResponse{
		Key:   "my-key",
		Value: encoded,
	}, nil)

	value, err := GetSecret(ctx, m.WorkspaceClient, "my-scope", "my-key")
	require.NoError(t, err)
	assert.Equal(t, []byte("my-secret-value"), value)
}

func TestGetSecret_Error(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)

	secretsAPI := m.GetMockSecretsAPI()
	secretsAPI.EXPECT().GetSecret(ctx, workspace.GetSecretRequest{
		Scope: "my-scope",
		Key:   "my-key",
	}).Return(nil, errors.New("not found"))

	_, err := GetSecret(ctx, m.WorkspaceClient, "my-scope", "my-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get secret my-key from scope my-scope")
}

func TestCheckAndGenerateSSHKeyPairFromSecrets_ExistingKeys(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)

	privKeyContent := []byte("private-key-content")
	pubKeyContent := []byte("public-key-content")

	secretsAPI := m.GetMockSecretsAPI()
	secretsAPI.EXPECT().GetSecret(ctx, workspace.GetSecretRequest{
		Scope: "my-scope",
		Key:   "client-private-key",
	}).Return(&workspace.GetSecretResponse{
		Value: base64.StdEncoding.EncodeToString(privKeyContent),
	}, nil)
	secretsAPI.EXPECT().GetSecret(ctx, workspace.GetSecretRequest{
		Scope: "my-scope",
		Key:   "client-public-key",
	}).Return(&workspace.GetSecretResponse{
		Value: base64.StdEncoding.EncodeToString(pubKeyContent),
	}, nil)

	privBytes, pubBytes, err := CheckAndGenerateSSHKeyPairFromSecrets(ctx, m.WorkspaceClient, "my-scope", "client-private-key", "client-public-key")
	require.NoError(t, err)
	assert.Equal(t, privKeyContent, privBytes)
	assert.Equal(t, pubKeyContent, pubBytes)
}

func TestCheckAndGenerateSSHKeyPairFromSecrets_GeneratesNewKeys(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)

	secretsAPI := m.GetMockSecretsAPI()
	// First GetSecret fails (no existing private key)
	secretsAPI.EXPECT().GetSecret(ctx, workspace.GetSecretRequest{
		Scope: "my-scope",
		Key:   "client-private-key",
	}).Return(nil, errors.New("not found"))

	// Expect both PutSecret calls for new keys (use mock.MatchedBy since key values are generated)
	secretsAPI.EXPECT().PutSecret(ctx, mock.MatchedBy(func(req workspace.PutSecret) bool {
		return req.Scope == "my-scope" && req.Key == "client-private-key" && req.StringValue != ""
	})).Return(nil)
	secretsAPI.EXPECT().PutSecret(ctx, mock.MatchedBy(func(req workspace.PutSecret) bool {
		return req.Scope == "my-scope" && req.Key == "client-public-key" && req.StringValue != ""
	})).Return(nil)

	privBytes, pubBytes, err := CheckAndGenerateSSHKeyPairFromSecrets(ctx, m.WorkspaceClient, "my-scope", "client-private-key", "client-public-key")
	require.NoError(t, err)

	// Verify the generated keys are valid
	block, _ := pem.Decode(privBytes)
	require.NotNil(t, block)
	assert.Equal(t, "RSA PRIVATE KEY", block.Type)

	assert.Contains(t, string(pubBytes), "ssh-rsa ")
}

func TestCheckAndGenerateSSHKeyPairFromSecrets_PutError(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)

	secretsAPI := m.GetMockSecretsAPI()
	secretsAPI.EXPECT().GetSecret(ctx, workspace.GetSecretRequest{
		Scope: "my-scope",
		Key:   "client-private-key",
	}).Return(nil, errors.New("not found"))

	secretsAPI.EXPECT().PutSecret(ctx, mock.MatchedBy(func(req workspace.PutSecret) bool {
		return req.Scope == "my-scope" && req.Key == "client-private-key"
	})).Return(errors.New("quota exceeded"))

	_, _, err := CheckAndGenerateSSHKeyPairFromSecrets(ctx, m.WorkspaceClient, "my-scope", "client-private-key", "client-public-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to store secret client-private-key")
}
