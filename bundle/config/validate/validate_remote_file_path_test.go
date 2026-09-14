package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const testRemoteFilePath = "/Workspace/Users/me/.bundle/files"

func setupBundleForRemoteFilePathTest(t *testing.T, objectType workspace.ObjectType) *bundle.Bundle {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				FilePath: testRemoteFilePath,
				CurrentUser: &config.User{
					User: &iam.User{UserName: "me@example.com"},
				},
			},
			Sync: config.Sync{
				Paths: []string{"."},
			},
		},
	}

	m := mocks.NewMockWorkspaceClient(t)
	m.WorkspaceClient.Config = &sdkconfig.Config{
		Host: "https://foo.test",
	}

	// No MkdirsByPath expectation: the dry-run check must not mutate remote state,
	// so an unexpected mkdir call fails the test.
	m.GetMockWorkspaceAPI().EXPECT().GetStatusByPath(mock.Anything, testRemoteFilePath).Return(&workspace.ObjectInfo{
		ObjectType: objectType,
		Path:       testRemoteFilePath,
	}, nil)

	b.SetWorkpaceClient(m.WorkspaceClient)
	return b
}

func TestValidateRemoteFilePath_Directory(t *testing.T) {
	b := setupBundleForRemoteFilePathTest(t, workspace.ObjectTypeDirectory)

	ctx := t.Context()
	diags := ValidateRemoteFilePath().Apply(ctx, b)
	assert.Empty(t, diags)
}

func TestValidateRemoteFilePath_NotADirectory(t *testing.T) {
	b := setupBundleForRemoteFilePathTest(t, workspace.ObjectTypeNotebook)

	ctx := t.Context()
	diags := ValidateRemoteFilePath().Apply(ctx, b)
	require.Len(t, diags, 1)
	assert.Equal(t, diag.Error, diags[0].Severity)
	assert.Contains(t, diags[0].Summary, testRemoteFilePath)
}

func TestValidateRemoteFilePath_NoSyncPaths(t *testing.T) {
	// No sync paths: nothing to check, so no workspace client is even set.
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				FilePath: testRemoteFilePath,
			},
			Sync: config.Sync{
				Paths: []string{},
			},
		},
	}

	ctx := t.Context()
	diags := ValidateRemoteFilePath().Apply(ctx, b)
	assert.Empty(t, diags)
}

func TestValidateRemoteFilePath_ImmutableFolder(t *testing.T) {
	// Under immutable_folder, file_path is not a sync destination, so it is not checked.
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				FilePath: testRemoteFilePath,
			},
			Sync: config.Sync{
				Paths: []string{"."},
			},
			Experimental: &config.Experimental{
				ImmutableFolder: true,
			},
		},
	}

	// Inject a mock client with no expectations: any workspace call (which would
	// happen if the check were not skipped) fails the test.
	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)

	ctx := t.Context()
	diags := ValidateRemoteFilePath().Apply(ctx, b)
	assert.Empty(t, diags)
}
