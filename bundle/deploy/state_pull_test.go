package deploy

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deploy/files"
	mockfiler "github.com/databricks/cli/internal/mocks/libs/filer"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/sync"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type snapshortStateExpectations struct {
	localToRemoteNames map[string]string
	remoteToLocalNames map[string]string
}

type statePullExpectations struct {
	seq                     int
	filesInDevelopmentState []File
	snapshotState           *snapshortStateExpectations
}

type statePullOpts struct {
	files                []File
	seq                  int
	localFiles           []string
	localNotebooks       []string
	expects              statePullExpectations
	withExistingSnapshot bool
	localState           *DeploymentState
}

func testStatePull(t *testing.T, opts statePullOpts) {
	s := &statePull{func(b *bundle.Bundle) (filer.Filer, error) {
		f := mockfiler.NewMockFiler(t)

		deploymentStateData, err := json.Marshal(DeploymentState{
			Version: DeploymentStateVersion,
			Seq:     int64(opts.seq),
			Files:   opts.files,
		})
		require.NoError(t, err)

		f.EXPECT().Read(mock.Anything, DeploymentStateFileName).Return(io.NopCloser(bytes.NewReader(deploymentStateData)), nil)

		return f, nil
	}}

	tmpDir := t.TempDir()
	b := &bundle.Bundle{
		BundleRootPath: tmpDir,
		BundleRoot:     vfs.MustNew(tmpDir),

		SyncRootPath: tmpDir,
		SyncRoot:     vfs.MustNew(tmpDir),

		Config: config.Root{
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

	for _, file := range opts.localFiles {
		testutil.Touch(t, b.SyncRootPath, "bar", file)
	}

	for _, file := range opts.localNotebooks {
		testutil.TouchNotebook(t, b.SyncRootPath, "bar", file)
	}

	if opts.withExistingSnapshot {
		opts, err := files.GetSyncOptions(ctx, b)
		require.NoError(t, err)

		snapshotPath, err := sync.SnapshotPath(opts)
		require.NoError(t, err)

		err = os.WriteFile(snapshotPath, []byte("snapshot"), 0o644)
		require.NoError(t, err)
	}

	if opts.localState != nil {
		statePath, err := getPathToStateFile(ctx, b)
		require.NoError(t, err)

		data, err := json.Marshal(opts.localState)
		require.NoError(t, err)

		err = os.WriteFile(statePath, data, 0o644)
		require.NoError(t, err)
	}

	diags := bundle.Apply(ctx, b, s)
	require.NoError(t, diags.Error())

	// Check that deployment state was written
	statePath, err := getPathToStateFile(ctx, b)
	require.NoError(t, err)

	data, err := os.ReadFile(statePath)
	require.NoError(t, err)

	var state DeploymentState
	err = json.Unmarshal(data, &state)
	require.NoError(t, err)

	require.Equal(t, int64(opts.expects.seq), state.Seq)
	require.Len(t, state.Files, len(opts.expects.filesInDevelopmentState))
	for i, file := range opts.expects.filesInDevelopmentState {
		require.Equal(t, file.LocalPath, state.Files[i].LocalPath)
	}

	if opts.expects.snapshotState != nil {
		syncOpts, err := files.GetSyncOptions(ctx, b)
		require.NoError(t, err)

		snapshotPath, err := sync.SnapshotPath(syncOpts)
		require.NoError(t, err)

		_, err = os.Stat(snapshotPath)
		require.NoError(t, err)

		data, err = os.ReadFile(snapshotPath)
		require.NoError(t, err)

		var snapshot sync.Snapshot
		err = json.Unmarshal(data, &snapshot)
		require.NoError(t, err)

		snapshotState := snapshot.SnapshotState
		require.Len(t, snapshotState.LocalToRemoteNames, len(opts.expects.snapshotState.localToRemoteNames))
		for local, remote := range opts.expects.snapshotState.localToRemoteNames {
			require.Equal(t, remote, snapshotState.LocalToRemoteNames[local])
		}

		require.Len(t, snapshotState.RemoteToLocalNames, len(opts.expects.snapshotState.remoteToLocalNames))
		for remote, local := range opts.expects.snapshotState.remoteToLocalNames {
			require.Equal(t, local, snapshotState.RemoteToLocalNames[remote])
		}
	}
}

var stateFiles []File = []File{
	{
		LocalPath:  "bar/t1.py",
		IsNotebook: false,
	},
	{
		LocalPath:  "bar/t2.py",
		IsNotebook: false,
	},
	{
		LocalPath:  "bar/notebook.py",
		IsNotebook: true,
	},
}

func TestStatePull(t *testing.T) {
	testStatePull(t, statePullOpts{
		seq:            1,
		files:          stateFiles,
		localFiles:     []string{"t1.py", "t2.py"},
		localNotebooks: []string{"notebook.py"},
		expects: statePullExpectations{
			seq: 1,
			filesInDevelopmentState: []File{
				{
					LocalPath: "bar/t1.py",
				},
				{
					LocalPath: "bar/t2.py",
				},
				{
					LocalPath: "bar/notebook.py",
				},
			},
			snapshotState: &snapshortStateExpectations{
				localToRemoteNames: map[string]string{
					"bar/t1.py":       "bar/t1.py",
					"bar/t2.py":       "bar/t2.py",
					"bar/notebook.py": "bar/notebook",
				},
				remoteToLocalNames: map[string]string{
					"bar/t1.py":    "bar/t1.py",
					"bar/t2.py":    "bar/t2.py",
					"bar/notebook": "bar/notebook.py",
				},
			},
		},
	})
}

func TestStatePullSnapshotExists(t *testing.T) {
	testStatePull(t, statePullOpts{
		withExistingSnapshot: true,
		seq:                  1,
		files:                stateFiles,
		localFiles:           []string{"t1.py", "t2.py"},
		expects: statePullExpectations{
			seq: 1,
			filesInDevelopmentState: []File{
				{
					LocalPath: "bar/t1.py",
				},
				{
					LocalPath: "bar/t2.py",
				},
				{
					LocalPath: "bar/notebook.py",
				},
			},
			snapshotState: &snapshortStateExpectations{
				localToRemoteNames: map[string]string{
					"bar/t1.py":       "bar/t1.py",
					"bar/t2.py":       "bar/t2.py",
					"bar/notebook.py": "bar/notebook",
				},
				remoteToLocalNames: map[string]string{
					"bar/t1.py":    "bar/t1.py",
					"bar/t2.py":    "bar/t2.py",
					"bar/notebook": "bar/notebook.py",
				},
			},
		},
	})
}

func TestStatePullNoState(t *testing.T) {
	s := &statePull{func(b *bundle.Bundle) (filer.Filer, error) {
		f := mockfiler.NewMockFiler(t)

		f.EXPECT().Read(mock.Anything, DeploymentStateFileName).Return(nil, os.ErrNotExist)

		return f, nil
	}}

	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Bundle: config.Bundle{
				Target: "default",
			},
			Workspace: config.Workspace{
				StatePath: "/state",
			},
		},
	}
	ctx := context.Background()

	diags := bundle.Apply(ctx, b, s)
	require.NoError(t, diags.Error())

	// Check that deployment state was not written
	statePath, err := getPathToStateFile(ctx, b)
	require.NoError(t, err)

	_, err = os.Stat(statePath)
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestStatePullOlderState(t *testing.T) {
	testStatePull(t, statePullOpts{
		seq:            1,
		files:          stateFiles,
		localFiles:     []string{"t1.py", "t2.py"},
		localNotebooks: []string{"notebook.py"},
		localState: &DeploymentState{
			Version: DeploymentStateVersion,
			Seq:     2,
			Files: []File{
				{
					LocalPath: "bar/t1.py",
				},
			},
		},
		expects: statePullExpectations{
			seq: 2,
			filesInDevelopmentState: []File{
				{
					LocalPath: "bar/t1.py",
				},
			},
		},
	})
}

func TestStatePullNewerState(t *testing.T) {
	testStatePull(t, statePullOpts{
		seq:            1,
		files:          stateFiles,
		localFiles:     []string{"t1.py", "t2.py"},
		localNotebooks: []string{"notebook.py"},
		localState: &DeploymentState{
			Version: DeploymentStateVersion,
			Seq:     0,
			Files: []File{
				{
					LocalPath: "bar/t1.py",
				},
			},
		},
		expects: statePullExpectations{
			seq: 1,
			filesInDevelopmentState: []File{
				{
					LocalPath: "bar/t1.py",
				},
				{
					LocalPath: "bar/t2.py",
				},
				{
					LocalPath: "bar/notebook.py",
				},
			},
			snapshotState: &snapshortStateExpectations{
				localToRemoteNames: map[string]string{
					"bar/t1.py":       "bar/t1.py",
					"bar/t2.py":       "bar/t2.py",
					"bar/notebook.py": "bar/notebook",
				},
				remoteToLocalNames: map[string]string{
					"bar/t1.py":    "bar/t1.py",
					"bar/t2.py":    "bar/t2.py",
					"bar/notebook": "bar/notebook.py",
				},
			},
		},
	})
}

func TestStatePullAndFileIsRemovedLocally(t *testing.T) {
	testStatePull(t, statePullOpts{
		seq:            1,
		files:          stateFiles,
		localFiles:     []string{"t2.py"}, // t1.py is removed locally
		localNotebooks: []string{"notebook.py"},
		expects: statePullExpectations{
			seq: 1,
			filesInDevelopmentState: []File{
				{
					LocalPath: "bar/t1.py",
				},
				{
					LocalPath: "bar/t2.py",
				},
				{
					LocalPath: "bar/notebook.py",
				},
			},
			snapshotState: &snapshortStateExpectations{
				localToRemoteNames: map[string]string{
					"bar/t1.py":       "bar/t1.py",
					"bar/t2.py":       "bar/t2.py",
					"bar/notebook.py": "bar/notebook",
				},
				remoteToLocalNames: map[string]string{
					"bar/t1.py":    "bar/t1.py",
					"bar/t2.py":    "bar/t2.py",
					"bar/notebook": "bar/notebook.py",
				},
			},
		},
	})
}

func TestStatePullAndNotebookIsRemovedLocally(t *testing.T) {
	testStatePull(t, statePullOpts{
		seq:            1,
		files:          stateFiles,
		localFiles:     []string{"t1.py", "t2.py"},
		localNotebooks: []string{}, // notebook.py is removed locally
		expects: statePullExpectations{
			seq: 1,
			filesInDevelopmentState: []File{
				{
					LocalPath: "bar/t1.py",
				},
				{
					LocalPath: "bar/t2.py",
				},
				{
					LocalPath: "bar/notebook.py",
				},
			},
			snapshotState: &snapshortStateExpectations{
				localToRemoteNames: map[string]string{
					"bar/t1.py":       "bar/t1.py",
					"bar/t2.py":       "bar/t2.py",
					"bar/notebook.py": "bar/notebook",
				},
				remoteToLocalNames: map[string]string{
					"bar/t1.py":    "bar/t1.py",
					"bar/t2.py":    "bar/t2.py",
					"bar/notebook": "bar/notebook.py",
				},
			},
		},
	})
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
					LocalPath: "bar/t1.py",
				},
				{
					LocalPath: "bar/t2.py",
				},
			},
		})
		require.NoError(t, err)

		f.EXPECT().Read(mock.Anything, DeploymentStateFileName).Return(io.NopCloser(bytes.NewReader(deploymentStateData)), nil)

		return f, nil
	}}

	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Bundle: config.Bundle{
				Target: "default",
			},
			Workspace: config.Workspace{
				StatePath: "/state",
			},
		},
	}
	ctx := context.Background()

	diags := bundle.Apply(ctx, b, s)
	require.True(t, diags.HasError())
	require.ErrorContains(t, diags.Error(), "remote deployment state is incompatible with the current version of the CLI, please upgrade to at least 1.2.3")
}
