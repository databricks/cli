package server

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	mockWorkspace "github.com/databricks/databricks-sdk-go/experimental/mocks/service/workspace"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testSetup struct {
	ctx          context.Context
	authKeysPath string
	mockClient   *mocks.MockWorkspaceClient
	secretsAPI   *mockWorkspace.MockSecretsInterface
	manager      *AuthorizedKeysManager
}

func setupTest(t *testing.T) *testSetup {
	ctx := context.Background()
	tempDir := t.TempDir()
	authKeysPath := filepath.Join(tempDir, "authorized_keys")

	m := mocks.NewMockWorkspaceClient(t)
	secretsAPI := m.GetMockSecretsAPI()
	manager := NewAuthorizedKeysManager(m.WorkspaceClient, authKeysPath, "test-scope")

	return &testSetup{
		ctx:          ctx,
		authKeysPath: authKeysPath,
		mockClient:   m,
		secretsAPI:   secretsAPI,
		manager:      manager,
	}
}

func (s *testSetup) mockGetSecret(keyName, publicKey string) {
	encodedKey := base64.StdEncoding.EncodeToString([]byte(publicKey))
	s.secretsAPI.EXPECT().GetSecret(s.ctx, workspace.GetSecretRequest{
		Scope: "test-scope",
		Key:   keyName,
	}).Return(&workspace.GetSecretResponse{
		Value: encodedKey,
	}, nil)
}

func (s *testSetup) mockGetSecretOnce(keyName, publicKey string) {
	encodedKey := base64.StdEncoding.EncodeToString([]byte(publicKey))
	s.secretsAPI.EXPECT().GetSecret(s.ctx, workspace.GetSecretRequest{
		Scope: "test-scope",
		Key:   keyName,
	}).Return(&workspace.GetSecretResponse{
		Value: encodedKey,
	}, nil).Once()
}

func (s *testSetup) mockGetSecretError(keyName string, err error) {
	s.secretsAPI.EXPECT().GetSecret(s.ctx, workspace.GetSecretRequest{
		Scope: "test-scope",
		Key:   keyName,
	}).Return(nil, err)
}

func TestAuthorizedKeysManager_AddKey_Success(t *testing.T) {
	s := setupTest(t)
	publicKey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC... test@example.com"

	s.mockGetSecret("test-key", publicKey)

	err := s.manager.AddKey(s.ctx, "test-key")
	require.NoError(t, err)

	content, err := os.ReadFile(s.authKeysPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), publicKey)
	assert.True(t, s.manager.addedKeys["test-key"])
}

func TestAuthorizedKeysManager_AddKey_Deduplication(t *testing.T) {
	s := setupTest(t)
	publicKey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC... test@example.com"

	s.mockGetSecretOnce("test-key", publicKey)

	err := s.manager.AddKey(s.ctx, "test-key")
	require.NoError(t, err)

	contentAfterFirst, err := os.ReadFile(s.authKeysPath)
	require.NoError(t, err)

	err = s.manager.AddKey(s.ctx, "test-key")
	require.NoError(t, err)

	contentAfterSecond, err := os.ReadFile(s.authKeysPath)
	require.NoError(t, err)

	assert.Equal(t, string(contentAfterFirst), string(contentAfterSecond))
	occurrences := strings.Count(string(contentAfterSecond), publicKey)
	assert.Equal(t, 1, occurrences)
}

func TestAuthorizedKeysManager_AddKey_GetSecretError(t *testing.T) {
	s := setupTest(t)

	s.mockGetSecretError("missing-key", errors.New("secret not found"))

	err := s.manager.AddKey(s.ctx, "missing-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get client public key")
	assert.False(t, s.manager.addedKeys["missing-key"])
}

func TestAuthorizedKeysManager_AddKey_FileWriteError(t *testing.T) {
	s := setupTest(t)
	publicKey := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC... test@example.com"

	s.mockGetSecret("test-key", publicKey)
	s.manager = NewAuthorizedKeysManager(s.mockClient.WorkspaceClient, "/nonexistent/directory/authorized_keys", "test-scope")

	err := s.manager.AddKey(s.ctx, "test-key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open authorized keys file")
	assert.False(t, s.manager.addedKeys["test-key"])
}

func TestAuthorizedKeysManager_AddKey_ThreadSafety(t *testing.T) {
	s := setupTest(t)
	const numGoroutines = 10
	const numKeysPerGoroutine = 10

	expectedKeys := make([]string, 0, numGoroutines*numKeysPerGoroutine)
	for i := range numGoroutines {
		for j := range numKeysPerGoroutine {
			keyName := fmt.Sprintf("key-%d-%d", i, j)
			publicKey := fmt.Sprintf("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC%d%d test@example.com", i, j)
			s.mockGetSecret(keyName, publicKey)
			expectedKeys = append(expectedKeys, publicKey)
		}
	}

	var wg sync.WaitGroup
	for i := range numGoroutines {
		wg.Go(func() {
			for j := range numKeysPerGoroutine {
				keyName := fmt.Sprintf("key-%d-%d", i, j)
				err := s.manager.AddKey(s.ctx, keyName)
				assert.NoError(t, err)
			}
		})
	}

	wg.Wait()

	assert.Equal(t, numGoroutines*numKeysPerGoroutine, len(s.manager.addedKeys))

	content, err := os.ReadFile(s.authKeysPath)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	for _, publicKey := range expectedKeys {
		assert.Contains(t, lines, publicKey)
	}
}

func TestAuthorizedKeysManager_AddKey_MultipleKeys(t *testing.T) {
	s := setupTest(t)
	keys := map[string]string{
		"key1": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC1 user1@example.com",
		"key2": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC2 user2@example.com",
		"key3": "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC3 user3@example.com",
	}

	for keyName, publicKey := range keys {
		s.mockGetSecret(keyName, publicKey)
	}

	for keyName := range keys {
		err := s.manager.AddKey(s.ctx, keyName)
		require.NoError(t, err)
	}

	content, err := os.ReadFile(s.authKeysPath)
	require.NoError(t, err)

	for _, publicKey := range keys {
		assert.Contains(t, string(content), publicKey)
	}

	assert.Equal(t, len(keys), len(s.manager.addedKeys))
}
