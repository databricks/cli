package client

import (
	"encoding/base64"
	"errors"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	testPrivateKeyName = "client-private-key"
	testPublicKeyName  = "client-public-key"
)

// mockSecretsForRun sets up secrets mocks (scope exists, keys exist) for a Run() test.
// Run() wraps the context with WithCancel, so all mocks must use mock.Anything for context.
func mockSecretsForRun(m *mocks.MockWorkspaceClient, sessionID string) {
	scopeName := "testuser@example.com-" + sessionID + "-ssh-tunnel-keys"

	m.GetMockCurrentUserAPI().EXPECT().Me(mock.Anything).Return(&iam.User{
		UserName: "testuser@example.com",
	}, nil)

	m.GetMockSecretsAPI().EXPECT().ListSecretsByScope(mock.Anything, scopeName).
		Return(&workspace.ListSecretsResponse{}, nil)

	privKey := base64.StdEncoding.EncodeToString([]byte("fake-private-key"))
	pubKey := base64.StdEncoding.EncodeToString([]byte("fake-public-key"))

	m.GetMockSecretsAPI().EXPECT().GetSecret(mock.Anything, workspace.GetSecretRequest{
		Scope: scopeName,
		Key:   testPrivateKeyName,
	}).Return(&workspace.GetSecretResponse{Value: privKey}, nil)

	m.GetMockSecretsAPI().EXPECT().GetSecret(mock.Anything, workspace.GetSecretRequest{
		Scope: scopeName,
		Key:   testPublicKeyName,
	}).Return(&workspace.GetSecretResponse{Value: pubKey}, nil)
}

func TestRun_EmptySessionID(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)

	err := Run(ctx, m.WorkspaceClient, ClientOptions{
		ProxyMode: true,
	})
	assert.EqualError(t, err, "either --cluster or --name must be provided")
}

// In Run(), the order for classic clusters is: cluster check -> secrets -> keys -> metadata -> SSH.
// Tests below mock only the calls that happen before the expected failure point.

func TestRun_ClassicCluster_ClusterCheckFails(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)

	// Cluster check happens before secrets. No secrets mock needed.
	m.GetMockClustersAPI().EXPECT().GetByClusterId(mock.Anything, "cluster-123").
		Return(nil, errors.New("cluster not found"))

	err := Run(ctx, m.WorkspaceClient, ClientOptions{
		ClusterID:      "cluster-123",
		ServerMetadata: "root,2222,cluster-123",
		SSHKeysDir:     t.TempDir(),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get cluster info")
}

func TestRun_ClassicCluster_ClusterNotRunning(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)

	m.GetMockClustersAPI().EXPECT().GetByClusterId(mock.Anything, "cluster-123").
		Return(&compute.ClusterDetails{State: compute.StateTerminated}, nil)

	err := Run(ctx, m.WorkspaceClient, ClientOptions{
		ClusterID:      "cluster-123",
		ServerMetadata: "root,2222,cluster-123",
		SSHKeysDir:     t.TempDir(),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not running")
	assert.Contains(t, err.Error(), "--auto-start-cluster")
}

func TestRun_SecretScopeCreationFails(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)

	// Use serverless mode to skip cluster check, making this test purely about secrets
	scopeName := "testuser@example.com-my-conn-ssh-tunnel-keys"
	m.GetMockCurrentUserAPI().EXPECT().Me(mock.Anything).Return(&iam.User{
		UserName: "testuser@example.com",
	}, nil)

	m.GetMockSecretsAPI().EXPECT().ListSecretsByScope(mock.Anything, scopeName).
		Return(nil, databricks.ErrResourceDoesNotExist)
	m.GetMockSecretsAPI().EXPECT().CreateScope(mock.Anything, workspace.CreateScope{Scope: scopeName}).
		Return(errors.New("limit exceeded"))

	err := Run(ctx, m.WorkspaceClient, ClientOptions{
		ConnectionName: "my-conn",
		ServerMetadata: "root,2222,cluster-123",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create secret scope")
}

func TestRun_ServerlessSkipsClusterCheck(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)
	mockSecretsForRun(m, "my-conn")

	// No cluster mock — if Run tries to check cluster state, the mock will panic.
	// Using metadata without cluster ID: serverless requires it, so Run fails after secrets.
	err := Run(ctx, m.WorkspaceClient, ClientOptions{
		ConnectionName:       "my-conn",
		ClientPrivateKeyName: testPrivateKeyName,
		ClientPublicKeyName:  testPublicKeyName,
		ServerMetadata:       "root,2222",
		SSHKeysDir:           t.TempDir(),
	})
	assert.Error(t, err)
	// Fails at serverless cluster ID check, proving cluster check was skipped
	assert.Contains(t, err.Error(), "cluster ID is required for serverless connections")
	assert.NotContains(t, err.Error(), "failed to get cluster")
}

func TestRun_InvalidMetadataFormat(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)

	// Cluster check passes, secrets pass, then metadata parsing fails
	m.GetMockClustersAPI().EXPECT().GetByClusterId(mock.Anything, "cluster-123").
		Return(&compute.ClusterDetails{State: compute.StateRunning}, nil)
	mockSecretsForRun(m, "cluster-123")

	err := Run(ctx, m.WorkspaceClient, ClientOptions{
		ClusterID:            "cluster-123",
		ClientPrivateKeyName: testPrivateKeyName,
		ClientPublicKeyName:  testPublicKeyName,
		ServerMetadata:       "badformat",
		SSHKeysDir:           t.TempDir(),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid metadata")
}

func TestRun_EmptyUserNameInMetadata(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)

	m.GetMockClustersAPI().EXPECT().GetByClusterId(mock.Anything, "cluster-123").
		Return(&compute.ClusterDetails{State: compute.StateRunning}, nil)
	mockSecretsForRun(m, "cluster-123")

	err := Run(ctx, m.WorkspaceClient, ClientOptions{
		ClusterID:            "cluster-123",
		ClientPrivateKeyName: testPrivateKeyName,
		ClientPublicKeyName:  testPublicKeyName,
		ServerMetadata:       ",2222,cluster-123",
		SSHKeysDir:           t.TempDir(),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "remote user name is empty")
}

func TestRun_ServerlessMissingClusterID(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)
	mockSecretsForRun(m, "my-conn")

	err := Run(ctx, m.WorkspaceClient, ClientOptions{
		ConnectionName:       "my-conn",
		ClientPrivateKeyName: testPrivateKeyName,
		ClientPublicKeyName:  testPublicKeyName,
		ServerMetadata:       "root,2222",
		SSHKeysDir:           t.TempDir(),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cluster ID is required for serverless connections")
}
