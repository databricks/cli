package deploy

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/internal/testutil"
	databrickscfg "github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestStateUpdate(t *testing.T) {
	s := &stateUpdate{}

	b := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Bundle: config.Bundle{
				Target: "default",
			},
			Workspace: config.Workspace{
				StatePath: "/state",
				FilePath:  "/files",
				CurrentUser: &config.User{
					User: &iam.User{
						UserName: "test-user",
					},
				},
			},
		},
	}

	testutil.Touch(t, b.Config.Path, "test1.py")
	testutil.Touch(t, b.Config.Path, "test2.py")

	m := mocks.NewMockWorkspaceClient(t)
	m.WorkspaceClient.Config = &databrickscfg.Config{
		Host: "https://test.com",
	}
	b.SetWorkpaceClient(m.WorkspaceClient)

	wsApi := m.GetMockWorkspaceAPI()
	wsApi.EXPECT().GetStatusByPath(mock.Anything, "/files").Return(&workspace.ObjectInfo{
		ObjectType: "DIRECTORY",
	}, nil)

	ctx := context.Background()

	err := bundle.Apply(ctx, b, s)
	require.NoError(t, err)

	// Check that the state file was updated.
	state, err := load(ctx, b)
	require.NoError(t, err)

	require.Equal(t, int64(1), state.Seq)
	require.Len(t, state.Files, 3)
	require.Equal(t, build.GetInfo().Version, state.CliVersion)

	err = bundle.Apply(ctx, b, s)
	require.NoError(t, err)

	// Check that the state file was updated again.
	state, err = load(ctx, b)
	require.NoError(t, err)

	require.Equal(t, int64(2), state.Seq)
	require.Len(t, state.Files, 3)
	require.Equal(t, build.GetInfo().Version, state.CliVersion)
}

func TestStateUpdateWithExistingState(t *testing.T) {
	s := &stateUpdate{}

	b := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Bundle: config.Bundle{
				Target: "default",
			},
			Workspace: config.Workspace{
				StatePath: "/state",
				FilePath:  "/files",
				CurrentUser: &config.User{
					User: &iam.User{
						UserName: "test-user",
					},
				},
			},
		},
	}

	testutil.Touch(t, b.Config.Path, "test1.py")
	testutil.Touch(t, b.Config.Path, "test2.py")

	m := mocks.NewMockWorkspaceClient(t)
	m.WorkspaceClient.Config = &databrickscfg.Config{
		Host: "https://test.com",
	}
	b.SetWorkpaceClient(m.WorkspaceClient)

	wsApi := m.GetMockWorkspaceAPI()
	wsApi.EXPECT().GetStatusByPath(mock.Anything, "/files").Return(&workspace.ObjectInfo{
		ObjectType: "DIRECTORY",
	}, nil)

	ctx := context.Background()

	// Create an existing state file.
	statePath, err := getPathToStateFile(ctx, b)
	require.NoError(t, err)

	state := &DeploymentState{
		Version:    DeploymentStateVersion,
		Seq:        10,
		CliVersion: build.GetInfo().Version,
		Files: []File{
			{
				LocalPath: "bar/t1.py",
			},
		},
	}

	data, err := json.Marshal(state)
	require.NoError(t, err)

	err = os.WriteFile(statePath, data, 0644)
	require.NoError(t, err)

	err = bundle.Apply(ctx, b, s)
	require.NoError(t, err)

	// Check that the state file was updated.
	state, err = load(ctx, b)
	require.NoError(t, err)

	require.Equal(t, int64(11), state.Seq)
	require.Len(t, state.Files, 3)
	require.Equal(t, build.GetInfo().Version, state.CliVersion)
}
