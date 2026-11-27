package keys_test

import (
	"testing"

	"github.com/databricks/cli/experimental/ssh/internal/keys"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testUserName  = "me@example.test"
	testSessionID = "session-1"
	testScopeName = testUserName + "-" + testSessionID + "-ssh-tunnel-keys"
)

func newScopeMock(t *testing.T) *mocks.MockWorkspaceClient {
	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockCurrentUserAPI().EXPECT().Me(t.Context(), iam.MeRequest{}).Return(&iam.User{UserName: testUserName}, nil)
	return m
}

func expectExistingScope(t *testing.T, m *mocks.MockWorkspaceClient, acls []workspace.AclItem, aclErr error) {
	secrets := m.GetMockSecretsAPI()
	secrets.EXPECT().ListSecretsByScope(t.Context(), testScopeName).Return(&workspace.ListSecretsResponse{}, nil)
	secrets.EXPECT().ListScopesAll(t.Context()).Return([]workspace.SecretScope{
		{Name: "other-scope", BackendType: workspace.ScopeBackendTypeAzureKeyvault},
		{Name: testScopeName, BackendType: workspace.ScopeBackendTypeDatabricks},
	}, nil)
	secrets.EXPECT().ListAclsAll(t.Context(), workspace.ListAclsRequest{Scope: testScopeName}).Return(acls, aclErr)
}

func TestCreateKeysSecretScopeCreatesMissingScope(t *testing.T) {
	m := newScopeMock(t)
	secrets := m.GetMockSecretsAPI()
	secrets.EXPECT().ListSecretsByScope(t.Context(), testScopeName).Return(nil, &apierr.APIError{
		StatusCode: 404,
		ErrorCode:  "RESOURCE_DOES_NOT_EXIST",
	})
	secrets.EXPECT().CreateScope(t.Context(), workspace.CreateScope{Scope: testScopeName}).Return(nil)

	scope, err := keys.CreateKeysSecretScope(t.Context(), m.WorkspaceClient, testSessionID, false)
	require.NoError(t, err)
	assert.Equal(t, testScopeName, scope)
}

func TestCreateKeysSecretScopeReusesOwnScope(t *testing.T) {
	m := newScopeMock(t)
	expectExistingScope(t, m, []workspace.AclItem{
		{Principal: testUserName, Permission: workspace.AclPermissionManage},
	}, nil)

	scope, err := keys.CreateKeysSecretScope(t.Context(), m.WorkspaceClient, testSessionID, false)
	require.NoError(t, err)
	assert.Equal(t, testScopeName, scope)
}

func TestCreateKeysSecretScopeRejectsScopeNotExclusivelyOwned(t *testing.T) {
	tests := []struct {
		name string
		acls []workspace.AclItem
	}{
		{
			name: "another principal also has access",
			acls: []workspace.AclItem{
				{Principal: testUserName, Permission: workspace.AclPermissionManage},
				{Principal: "other@example.test", Permission: workspace.AclPermissionRead},
			},
		},
		{
			name: "a group has access",
			acls: []workspace.AclItem{
				{Principal: testUserName, Permission: workspace.AclPermissionManage},
				{Principal: "users", Permission: workspace.AclPermissionWrite},
			},
		},
		{
			name: "user has only READ",
			acls: []workspace.AclItem{
				{Principal: testUserName, Permission: workspace.AclPermissionRead},
			},
		},
		{
			name: "user has only WRITE",
			acls: []workspace.AclItem{
				{Principal: testUserName, Permission: workspace.AclPermissionWrite},
			},
		},
		{
			name: "only another principal has MANAGE",
			acls: []workspace.AclItem{
				{Principal: "other@example.test", Permission: workspace.AclPermissionManage},
			},
		},
		{
			name: "no ACLs",
			acls: []workspace.AclItem{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newScopeMock(t)
			expectExistingScope(t, m, tc.acls, nil)

			_, err := keys.CreateKeysSecretScope(t.Context(), m.WorkspaceClient, testSessionID, false)
			assertScopeNotOwnedError(t, err)
		})
	}
}

// Listing ACLs requires MANAGE, so a user with only READ or WRITE gets PERMISSION_DENIED.
func TestCreateKeysSecretScopeRejectsScopeWhenACLListIsDenied(t *testing.T) {
	m := newScopeMock(t)
	expectExistingScope(t, m, nil, &apierr.APIError{
		StatusCode: 403,
		ErrorCode:  "PERMISSION_DENIED",
	})

	_, err := keys.CreateKeysSecretScope(t.Context(), m.WorkspaceClient, testSessionID, false)
	assertScopeNotOwnedError(t, err)
}

// A different --name gives a serverless connection a different session and so a different
// scope. On a dedicated cluster the session is the cluster ID, so the hint does not apply.
func TestCreateKeysSecretScopeSuggestsOtherNameOnlyOnServerless(t *testing.T) {
	notOwned := []workspace.AclItem{{Principal: "other@example.test", Permission: workspace.AclPermissionManage}}

	m := newScopeMock(t)
	expectExistingScope(t, m, notOwned, nil)
	_, err := keys.CreateKeysSecretScope(t.Context(), m.WorkspaceClient, testSessionID, true)
	require.Error(t, err)
	assert.Equal(t, scopeNotOwnedMessage+". You can instead connect with a different --name", err.Error())

	m = newScopeMock(t)
	expectExistingScope(t, m, notOwned, nil)
	_, err = keys.CreateKeysSecretScope(t.Context(), m.WorkspaceClient, testSessionID, false)
	assertScopeNotOwnedError(t, err)
}

func TestCreateKeysSecretScopeReturnsOtherACLListErrors(t *testing.T) {
	m := newScopeMock(t)
	expectExistingScope(t, m, nil, &apierr.APIError{
		StatusCode: 500,
		ErrorCode:  "INTERNAL_ERROR",
		Message:    "boom",
	})

	_, err := keys.CreateKeysSecretScope(t.Context(), m.WorkspaceClient, testSessionID, false)
	require.Error(t, err)
	assert.Equal(t, "failed to list ACLs of secret scope "+testScopeName+": boom", err.Error())
}

// scopeNotOwnedMessage is the error for a refused scope outside serverless mode. Deleting the
// scope alone is not enough while a server for the session runs: the CLI reuses that server,
// which still authorizes the old client key, and the new scope has no host key to pin.
const scopeNotOwnedMessage = "secret scope " + testScopeName + " already exists and is not exclusively owned by " + testUserName +
	" (the scope must grant MANAGE to " + testUserName + " and no access to other principals). " +
	"Delete the scope with `databricks secrets delete-scope " + testScopeName + "` (this needs MANAGE permission on the scope; otherwise ask a workspace admin) " +
	"and connect again once no SSH server for this session is running: a running server keeps the keys of the deleted scope " +
	"and shuts down --shutdown-delay after its last client disconnects"

func assertScopeNotOwnedError(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	assert.Equal(t, scopeNotOwnedMessage, err.Error())
}

// The ACL of a scope backed by Azure Key Vault does not govern access through the vault, so
// such a scope is refused before its ACL is read. The mock is strict: an ACL list call fails
// the test.
func TestCreateKeysSecretScopeRejectsScopeNotBackedByDatabricks(t *testing.T) {
	m := newScopeMock(t)
	secrets := m.GetMockSecretsAPI()
	secrets.EXPECT().ListSecretsByScope(t.Context(), testScopeName).Return(&workspace.ListSecretsResponse{}, nil)
	secrets.EXPECT().ListScopesAll(t.Context()).Return([]workspace.SecretScope{
		{Name: testScopeName, BackendType: workspace.ScopeBackendTypeAzureKeyvault},
	}, nil)

	_, err := keys.CreateKeysSecretScope(t.Context(), m.WorkspaceClient, testSessionID, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "secret scope "+testScopeName+" already exists and is backed by AZURE_KEYVAULT (the CLI keeps SSH keys only in a DATABRICKS-backed scope). Delete the scope with")
}

func TestCreateKeysSecretScopeFailsWhenScopeIsNotListed(t *testing.T) {
	m := newScopeMock(t)
	secrets := m.GetMockSecretsAPI()
	secrets.EXPECT().ListSecretsByScope(t.Context(), testScopeName).Return(&workspace.ListSecretsResponse{}, nil)
	secrets.EXPECT().ListScopesAll(t.Context()).Return([]workspace.SecretScope{
		{Name: "other-scope", BackendType: workspace.ScopeBackendTypeDatabricks},
	}, nil)

	_, err := keys.CreateKeysSecretScope(t.Context(), m.WorkspaceClient, testSessionID, false)
	require.Error(t, err)
	assert.Equal(t, "secret scope "+testScopeName+" is missing from the list of secret scopes", err.Error())
}
