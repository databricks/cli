package deploy

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deploy/files"
	mockfiler "github.com/databricks/cli/internal/mocks/libs/filer"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/sync"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func touchNotebook(t *testing.T, path, file string) {
	os.MkdirAll(path, 0755)
	f, err := os.Create(filepath.Join(path, file))
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(path, file), []byte("# Databricks notebook source"), 0644)
	require.NoError(t, err)
	f.Close()
}

func TestStatePull(t *testing.T) {
	s := &statePull{func(b *bundle.Bundle) (filer.Filer, error) {
		f := mockfiler.NewMockFiler(t)

		deploymentStateData, err := json.Marshal(DeploymentState{
			Version: DeploymentStateVersion,
			Seq:     1,
			Files: []File{
				{
					Path:       "bar/t1.py",
					IsNotebook: false,
				},
				{
					Path:       "bar/t2.py",
					IsNotebook: false,
				},
				{
					Path:       "bar/notebook.py",
					IsNotebook: true,
				},
			},
		})
		require.NoError(t, err)

		f.EXPECT().Read(mock.Anything, DeploymentStateFileName).Return(io.NopCloser(bytes.NewReader(deploymentStateData)), nil)

		return f, nil
	}}

	b := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Bundle: config.Bundle{
				Target: "default",
			},
			Workspace: config.Workspace{
				StatePath: "/state",
				CurrentUser: &config.User{
					User: &iam.User{
						UserName: "test-user",
					},
				},
			},
		},
	}
	ctx := context.Background()

	touch(t, filepath.Join(b.Config.Path, "bar"), "t1.py")
	touch(t, filepath.Join(b.Config.Path, "bar"), "t2.py")
	touchNotebook(t, filepath.Join(b.Config.Path, "bar"), "notebook.py")

	err := bundle.Apply(ctx, b, s)
	require.NoError(t, err)

	// Check that deployment state was written
	statePath, err := getPathToStateFile(ctx, b)
	require.NoError(t, err)

	data, err := os.ReadFile(statePath)
	require.NoError(t, err)

	var state DeploymentState
	err = json.Unmarshal(data, &state)
	require.NoError(t, err)

	require.Equal(t, int64(1), state.Seq)
	require.Len(t, state.Files, 3)
	require.Equal(t, "bar/t1.py", state.Files[0].Path)
	require.Equal(t, "bar/t2.py", state.Files[1].Path)
	require.Equal(t, "bar/notebook.py", state.Files[2].Path)

	opts, err := files.GetSyncOptions(ctx, b)
	require.NoError(t, err)

	snapshotPath, err := sync.SnapshotPath(opts)
	require.NoError(t, err)

	_, err = os.Stat(snapshotPath)
	require.NoError(t, err)

	data, err = os.ReadFile(snapshotPath)
	require.NoError(t, err)

	var snapshot sync.Snapshot
	err = json.Unmarshal(data, &snapshot)
	require.NoError(t, err)

	snapshotState := snapshot.SnapshotState
	require.Len(t, snapshotState.LocalToRemoteNames, 3)
	require.Equal(t, "bar/t1.py", snapshotState.LocalToRemoteNames["bar/t1.py"])
	require.Equal(t, "bar/t2.py", snapshotState.LocalToRemoteNames["bar/t2.py"])
	require.Equal(t, "bar/notebook", snapshotState.LocalToRemoteNames["bar/notebook.py"])

	require.Len(t, snapshotState.RemoteToLocalNames, 3)
	require.Equal(t, "bar/t1.py", snapshotState.RemoteToLocalNames["bar/t1.py"])
	require.Equal(t, "bar/t2.py", snapshotState.RemoteToLocalNames["bar/t2.py"])
	require.Equal(t, "bar/notebook.py", snapshotState.RemoteToLocalNames["bar/notebook"])
}

func TestStatePullSnapshotExists(t *testing.T) {
	s := &statePull{func(b *bundle.Bundle) (filer.Filer, error) {
		f := mockfiler.NewMockFiler(t)

		deploymentStateData, err := json.Marshal(DeploymentState{
			Version: DeploymentStateVersion,
			Seq:     1,
			Files: []File{
				{
					Path: "bar/t1.py",
				},
				{
					Path: "bar/t2.py",
				},
			},
		})
		require.NoError(t, err)

		f.EXPECT().Read(mock.Anything, DeploymentStateFileName).Return(io.NopCloser(bytes.NewReader(deploymentStateData)), nil)

		return f, nil
	}}

	b := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Bundle: config.Bundle{
				Target: "default",
			},
			Workspace: config.Workspace{
				StatePath: "/state",
				CurrentUser: &config.User{
					User: &iam.User{
						UserName: "test-user",
					},
				},
			},
		},
	}
	ctx := context.Background()

	touch(t, filepath.Join(b.Config.Path, "bar"), "t1.py")
	touch(t, filepath.Join(b.Config.Path, "bar"), "t2.py")

	opts, err := files.GetSyncOptions(ctx, b)
	require.NoError(t, err)

	snapshotPath, err := sync.SnapshotPath(opts)
	require.NoError(t, err)

	// Create a snapshot file
	err = os.WriteFile(snapshotPath, []byte("snapshot"), 0644)
	require.NoError(t, err)

	err = bundle.Apply(ctx, b, s)
	require.NoError(t, err)

	// Check that deployment state was written
	statePath, err := getPathToStateFile(ctx, b)
	require.NoError(t, err)

	data, err := os.ReadFile(statePath)
	require.NoError(t, err)

	var state DeploymentState
	err = json.Unmarshal(data, &state)
	require.NoError(t, err)

	require.Equal(t, int64(1), state.Seq)
	require.Len(t, state.Files, 2)
	require.Equal(t, "bar/t1.py", state.Files[0].Path)
	require.Equal(t, "bar/t2.py", state.Files[1].Path)

	// Check that snapshot is overriden anyway because deployment state is newer
	data, err = os.ReadFile(snapshotPath)
	require.NoError(t, err)

	var snapshot sync.Snapshot
	err = json.Unmarshal(data, &snapshot)
	require.NoError(t, err)

	snapshotState := snapshot.SnapshotState
	require.Len(t, snapshotState.LocalToRemoteNames, 2)
	require.Equal(t, "bar/t1.py", snapshotState.LocalToRemoteNames["bar/t1.py"])
	require.Equal(t, "bar/t2.py", snapshotState.LocalToRemoteNames["bar/t2.py"])

	require.Len(t, snapshotState.RemoteToLocalNames, 2)
	require.Equal(t, "bar/t1.py", snapshotState.RemoteToLocalNames["bar/t1.py"])
	require.Equal(t, "bar/t2.py", snapshotState.RemoteToLocalNames["bar/t2.py"])
}

func TestStatePullNoState(t *testing.T) {
	s := &statePull{func(b *bundle.Bundle) (filer.Filer, error) {
		f := mockfiler.NewMockFiler(t)

		f.EXPECT().Read(mock.Anything, DeploymentStateFileName).Return(nil, os.ErrNotExist)

		return f, nil
	}}

	b := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Bundle: config.Bundle{
				Target: "default",
			},
			Workspace: config.Workspace{
				StatePath: "/state",
			},
		},
	}
	ctx := context.Background()

	err := bundle.Apply(ctx, b, s)
	require.NoError(t, err)

	// Check that deployment state was not written
	statePath, err := getPathToStateFile(ctx, b)
	require.NoError(t, err)

	_, err = os.Stat(statePath)
	require.True(t, os.IsNotExist(err))
}

func TestStatePullOlderState(t *testing.T) {
	s := &statePull{func(b *bundle.Bundle) (filer.Filer, error) {
		f := mockfiler.NewMockFiler(t)

		deploymentStateData, err := json.Marshal(DeploymentState{
			Version: DeploymentStateVersion,
			Seq:     1,
			Files: []File{
				{
					Path: "bar/t1.py",
				},
				{
					Path: "bar/t2.py",
				},
			},
		})
		require.NoError(t, err)

		f.EXPECT().Read(mock.Anything, DeploymentStateFileName).Return(io.NopCloser(bytes.NewReader(deploymentStateData)), nil)

		return f, nil
	}}

	b := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Bundle: config.Bundle{
				Target: "default",
			},
			Workspace: config.Workspace{
				StatePath: "/state",
			},
		},
	}
	ctx := context.Background()

	// Create an newer local deployment state file
	statePath, err := getPathToStateFile(ctx, b)
	require.NoError(t, err)

	newerState := DeploymentState{
		Version: DeploymentStateVersion,
		Seq:     2,
		Files: []File{
			{
				Path: "bar/t1.py",
			},
		},
	}

	data, err := json.Marshal(newerState)
	require.NoError(t, err)

	err = os.WriteFile(statePath, data, 0644)
	require.NoError(t, err)

	err = bundle.Apply(ctx, b, s)
	require.NoError(t, err)

	// Check that deployment state was not written
	data, err = os.ReadFile(statePath)
	require.NoError(t, err)

	var state DeploymentState
	err = json.Unmarshal(data, &state)
	require.NoError(t, err)

	require.Equal(t, int64(2), state.Seq)
	require.Len(t, state.Files, 1)
	require.Equal(t, "bar/t1.py", state.Files[0].Path)
}

func TestStatePullNewerState(t *testing.T) {
	s := &statePull{func(b *bundle.Bundle) (filer.Filer, error) {
		f := mockfiler.NewMockFiler(t)

		deploymentStateData, err := json.Marshal(DeploymentState{
			Version: DeploymentStateVersion,
			Seq:     1,
			Files: []File{
				{
					Path: "bar/t1.py",
				},
				{
					Path: "bar/t2.py",
				},
			},
		})
		require.NoError(t, err)

		f.EXPECT().Read(mock.Anything, DeploymentStateFileName).Return(io.NopCloser(bytes.NewReader(deploymentStateData)), nil)

		return f, nil
	}}

	b := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Bundle: config.Bundle{
				Target: "default",
			},
			Workspace: config.Workspace{
				StatePath: "/state",
				CurrentUser: &config.User{
					User: &iam.User{
						UserName: "test-user",
					},
				},
			},
		},
	}
	ctx := context.Background()

	// Create an older local deployment state file
	statePath, err := getPathToStateFile(ctx, b)
	require.NoError(t, err)

	olderState := DeploymentState{
		Version: DeploymentStateVersion,
		Seq:     0,
		Files: []File{
			{
				Path: "bar/t1.py",
			},
		},
	}

	data, err := json.Marshal(olderState)
	require.NoError(t, err)

	err = os.WriteFile(statePath, data, 0644)
	require.NoError(t, err)

	err = bundle.Apply(ctx, b, s)
	require.NoError(t, err)

	// Check that deployment state was written
	data, err = os.ReadFile(statePath)
	require.NoError(t, err)

	var state DeploymentState
	err = json.Unmarshal(data, &state)
	require.NoError(t, err)

	require.Equal(t, int64(1), state.Seq)
	require.Len(t, state.Files, 2)
	require.Equal(t, "bar/t1.py", state.Files[0].Path)
	require.Equal(t, "bar/t2.py", state.Files[1].Path)
}

func TestStatePullAndFileIsRemovedLocally(t *testing.T) {
	s := &statePull{func(b *bundle.Bundle) (filer.Filer, error) {
		f := mockfiler.NewMockFiler(t)

		deploymentStateData, err := json.Marshal(DeploymentState{
			Version: DeploymentStateVersion,
			Seq:     1,
			Files: []File{
				{
					Path:       "bar/t1.py",
					IsNotebook: false,
				},
				{
					Path:       "bar/t2.py",
					IsNotebook: false,
				},
				{
					Path:       "bar/notebook.py",
					IsNotebook: true,
				},
			},
		})
		require.NoError(t, err)

		f.EXPECT().Read(mock.Anything, DeploymentStateFileName).Return(io.NopCloser(bytes.NewReader(deploymentStateData)), nil)

		return f, nil
	}}

	b := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Bundle: config.Bundle{
				Target: "default",
			},
			Workspace: config.Workspace{
				StatePath: "/state",
				CurrentUser: &config.User{
					User: &iam.User{
						UserName: "test-user",
					},
				},
			},
		},
	}
	ctx := context.Background()

	touch(t, filepath.Join(b.Config.Path, "bar"), "t2.py")
	touchNotebook(t, filepath.Join(b.Config.Path, "bar"), "notebook.py")

	err := bundle.Apply(ctx, b, s)
	require.NoError(t, err)

	// Check that deployment state was written
	statePath, err := getPathToStateFile(ctx, b)
	require.NoError(t, err)

	data, err := os.ReadFile(statePath)
	require.NoError(t, err)

	var state DeploymentState
	err = json.Unmarshal(data, &state)
	require.NoError(t, err)

	require.Equal(t, int64(1), state.Seq)
	require.Len(t, state.Files, 3)
	require.Equal(t, "bar/t1.py", state.Files[0].Path)
	require.Equal(t, "bar/t2.py", state.Files[1].Path)
	require.Equal(t, "bar/notebook.py", state.Files[2].Path)

	opts, err := files.GetSyncOptions(ctx, b)
	require.NoError(t, err)

	snapshotPath, err := sync.SnapshotPath(opts)
	require.NoError(t, err)

	_, err = os.Stat(snapshotPath)
	require.NoError(t, err)

	data, err = os.ReadFile(snapshotPath)
	require.NoError(t, err)

	var snapshot sync.Snapshot
	err = json.Unmarshal(data, &snapshot)
	require.NoError(t, err)

	snapshotState := snapshot.SnapshotState

	// Check that even though file is removed locally, it is still in the snapshot restored from deployment state
	require.Len(t, snapshotState.LocalToRemoteNames, 3)
	require.Equal(t, "bar/t1.py", snapshotState.LocalToRemoteNames["bar/t1.py"])
	require.Equal(t, "bar/t2.py", snapshotState.LocalToRemoteNames["bar/t2.py"])
	require.Equal(t, "bar/notebook", snapshotState.LocalToRemoteNames["bar/notebook.py"])

	require.Len(t, snapshotState.RemoteToLocalNames, 3)
	require.Equal(t, "bar/t1.py", snapshotState.RemoteToLocalNames["bar/t1.py"])
	require.Equal(t, "bar/t2.py", snapshotState.RemoteToLocalNames["bar/t2.py"])
	require.Equal(t, "bar/notebook.py", snapshotState.RemoteToLocalNames["bar/notebook"])
}

func TestStatePullAndNotebookIsRemovedLocally(t *testing.T) {
	s := &statePull{func(b *bundle.Bundle) (filer.Filer, error) {
		f := mockfiler.NewMockFiler(t)

		deploymentStateData, err := json.Marshal(DeploymentState{
			Version: DeploymentStateVersion,
			Seq:     1,
			Files: []File{
				{
					Path:       "bar/t1.py",
					IsNotebook: false,
				},
				{
					Path:       "bar/t2.py",
					IsNotebook: false,
				},
				{
					Path:       "bar/notebook.py",
					IsNotebook: true,
				},
			},
		})
		require.NoError(t, err)

		f.EXPECT().Read(mock.Anything, DeploymentStateFileName).Return(io.NopCloser(bytes.NewReader(deploymentStateData)), nil)

		return f, nil
	}}

	b := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Bundle: config.Bundle{
				Target: "default",
			},
			Workspace: config.Workspace{
				StatePath: "/state",
				CurrentUser: &config.User{
					User: &iam.User{
						UserName: "test-user",
					},
				},
			},
		},
	}
	ctx := context.Background()

	touch(t, filepath.Join(b.Config.Path, "bar"), "t1.py")
	touch(t, filepath.Join(b.Config.Path, "bar"), "t2.py")

	err := bundle.Apply(ctx, b, s)
	require.NoError(t, err)

	// Check that deployment state was written
	statePath, err := getPathToStateFile(ctx, b)
	require.NoError(t, err)

	data, err := os.ReadFile(statePath)
	require.NoError(t, err)

	var state DeploymentState
	err = json.Unmarshal(data, &state)
	require.NoError(t, err)

	require.Equal(t, int64(1), state.Seq)
	require.Len(t, state.Files, 3)
	require.Equal(t, "bar/t1.py", state.Files[0].Path)
	require.Equal(t, "bar/t2.py", state.Files[1].Path)
	require.Equal(t, "bar/notebook.py", state.Files[2].Path)

	opts, err := files.GetSyncOptions(ctx, b)
	require.NoError(t, err)

	snapshotPath, err := sync.SnapshotPath(opts)
	require.NoError(t, err)

	_, err = os.Stat(snapshotPath)
	require.NoError(t, err)

	data, err = os.ReadFile(snapshotPath)
	require.NoError(t, err)

	var snapshot sync.Snapshot
	err = json.Unmarshal(data, &snapshot)
	require.NoError(t, err)

	snapshotState := snapshot.SnapshotState

	// Check that even though notebook is removed locally, it is still in the snapshot restored from deployment state
	require.Len(t, snapshotState.LocalToRemoteNames, 3)
	require.Equal(t, "bar/t1.py", snapshotState.LocalToRemoteNames["bar/t1.py"])
	require.Equal(t, "bar/t2.py", snapshotState.LocalToRemoteNames["bar/t2.py"])
	require.Equal(t, "bar/notebook", snapshotState.LocalToRemoteNames["bar/notebook.py"])

	require.Len(t, snapshotState.RemoteToLocalNames, 3)
	require.Equal(t, "bar/t1.py", snapshotState.RemoteToLocalNames["bar/t1.py"])
	require.Equal(t, "bar/t2.py", snapshotState.RemoteToLocalNames["bar/t2.py"])
	require.Equal(t, "bar/notebook.py", snapshotState.RemoteToLocalNames["bar/notebook"])
}

func TestStatePullNewerDeploymentStateVersion(t *testing.T) {
	s := &statePull{func(b *bundle.Bundle) (filer.Filer, error) {
		f := mockfiler.NewMockFiler(t)

		deploymentStateData, err := json.Marshal(DeploymentState{
			Version:    DeploymentStateVersion + 1,
			Seq:        1,
			CliVersion: "1.2.3",
			Files: []File{
				{
					Path: "bar/t1.py",
				},
				{
					Path: "bar/t2.py",
				},
			},
		})
		require.NoError(t, err)

		f.EXPECT().Read(mock.Anything, DeploymentStateFileName).Return(io.NopCloser(bytes.NewReader(deploymentStateData)), nil)

		return f, nil
	}}

	b := &bundle.Bundle{
		Config: config.Root{
			Path: t.TempDir(),
			Bundle: config.Bundle{
				Target: "default",
			},
			Workspace: config.Workspace{
				StatePath: "/state",
			},
		},
	}
	ctx := context.Background()

	err := bundle.Apply(ctx, b, s)
	require.Error(t, err)
	require.Contains(t, err.Error(), "remote deployment state is incompatible with current version of CLI, please upgarde to at least 1.2.3")
}
