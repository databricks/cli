package generate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewGenerateStorageCredentialCommand_Help(t *testing.T) {
	cmd := NewGenerateStorageCredentialCommand()
	assert.Contains(t, cmd.Long, "storage credential")
	assert.NotNil(t, cmd.Flag("existing-storage-credential-name"))
}

func TestGenerateStorageCredential_AwsIamRole(t *testing.T) {
	work := t.TempDir()
	w := mocks.NewMockWorkspaceClient(t)
	w.GetMockStorageCredentialsAPI().EXPECT().
		GetByName(mock.Anything, "prod_cred").
		Return(&catalog.StorageCredentialInfo{
			Name: "prod_cred",
			AwsIamRole: &catalog.AwsIamRoleResponse{
				RoleArn: "arn:aws:iam::123:role/prod",
			},
		}, nil)

	_, err := runSubcmd(t, w,
		"storage-credential",
		"--existing-storage-credential-name", "prod_cred",
		"--output-dir", work,
	)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(work, "storage_credentials_prod_cred.yml"))
	require.NoError(t, err)
	contents := string(data)
	assert.Contains(t, contents, "storage_credentials:")
	assert.Contains(t, contents, "role_arn: arn:aws:iam::123:role/prod")
}

func TestGenerateStorageCredential_AzureSPWarning(t *testing.T) {
	work := t.TempDir()
	w := mocks.NewMockWorkspaceClient(t)
	w.GetMockStorageCredentialsAPI().EXPECT().
		GetByName(mock.Anything, "azure_sp").
		Return(&catalog.StorageCredentialInfo{
			Name: "azure_sp",
			AzureServicePrincipal: &catalog.AzureServicePrincipal{
				DirectoryId:   "dir-id",
				ApplicationId: "app-id",
			},
		}, nil)

	stderr, err := runSubcmd(t, w,
		"storage-credential",
		"--existing-storage-credential-name", "azure_sp",
		"--output-dir", work,
	)
	require.NoError(t, err)
	assert.Contains(t, stderr, "client_secret not available")

	data, err := os.ReadFile(filepath.Join(work, "storage_credentials_azure_sp.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "directory_id: dir-id")
}

func TestGenerateStorageCredential_UnsupportedIdentityErrors(t *testing.T) {
	work := t.TempDir()
	w := mocks.NewMockWorkspaceClient(t)
	w.GetMockStorageCredentialsAPI().EXPECT().
		GetByName(mock.Anything, "unknown").
		Return(&catalog.StorageCredentialInfo{Name: "unknown"}, nil)

	_, err := runSubcmd(t, w,
		"storage-credential",
		"--existing-storage-credential-name", "unknown",
		"--output-dir", work,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported identity type")
}
