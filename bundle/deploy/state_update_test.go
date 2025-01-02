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
	"github.com/databricks/cli/libs/fileset"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func setupBundleForStateUpdate(t *testing.T) *bundle.Bundle {
	tmpDir := t.TempDir()

	testutil.Touch(t, tmpDir, "test1.py")
	testutil.TouchNotebook(t, tmpDir, "test2.py")

	files, err := fileset.New(vfs.MustNew(tmpDir)).Files()
	require.NoError(t, err)

	return &bundle.Bundle{
		BundleRootPath: tmpDir,
		Config: config.Root{
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
		Files: files,
	}
}

func TestStateUpdate(t *testing.T) {
	s := &stateUpdate{}

	b := setupBundleForStateUpdate(t)
	ctx := context.Background()

	diags := bundle.Apply(ctx, b, s)
	require.NoError(t, diags.Error())

	// Check that the state file was updated.
	state, err := load(ctx, b)
	require.NoError(t, err)

	require.Equal(t, int64(1), state.Seq)
	require.Equal(t, Filelist{
		{
			LocalPath: "test1.py",
		},
		{
			LocalPath:  "test2.py",
			IsNotebook: true,
		},
	}, state.Files)
	require.Equal(t, build.GetInfo().Version, state.CliVersion)

	diags = bundle.Apply(ctx, b, s)
	require.NoError(t, diags.Error())

	// Check that the state file was updated again.
	state, err = load(ctx, b)
	require.NoError(t, err)

	require.Equal(t, int64(2), state.Seq)
	require.Equal(t, Filelist{
		{
			LocalPath: "test1.py",
		},
		{
			LocalPath:  "test2.py",
			IsNotebook: true,
		},
	}, state.Files)
	require.Equal(t, build.GetInfo().Version, state.CliVersion)

	// Valid non-empty UUID is generated.
	require.NotEqual(t, uuid.Nil, state.ID)
}

func TestStateUpdateWithExistingState(t *testing.T) {
	s := &stateUpdate{}

	b := setupBundleForStateUpdate(t)
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
		ID: uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
	}

	data, err := json.Marshal(state)
	require.NoError(t, err)

	err = os.WriteFile(statePath, data, 0o644)
	require.NoError(t, err)

	diags := bundle.Apply(ctx, b, s)
	require.NoError(t, diags.Error())

	// Check that the state file was updated.
	state, err = load(ctx, b)
	require.NoError(t, err)

	require.Equal(t, int64(11), state.Seq)
	require.Equal(t, Filelist{
		{
			LocalPath: "test1.py",
		},
		{
			LocalPath:  "test2.py",
			IsNotebook: true,
		},
	}, state.Files)
	require.Equal(t, build.GetInfo().Version, state.CliVersion)

	// Existing UUID is not overwritten.
	require.Equal(t, uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), state.ID)
}
