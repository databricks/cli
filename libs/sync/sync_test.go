package sync_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/fileset"
	"github.com/databricks/cli/libs/sync"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncSnapshotDirectoryIdentity(t *testing.T) {
	for _, tc := range []struct {
		name        string
		exists      bool
		legacy      bool
		replaced    bool
		wantUploads int
	}{
		{name: "unchanged", exists: true},
		{name: "legacy snapshot", exists: true, legacy: true, wantUploads: 1},
		{name: "replaced directory", exists: true, replaced: true, wantUploads: 1},
		{name: "deleted directory", wantUploads: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			server := testserver.New(t)
			testserver.AddDefaultHandlers(server)
			client, err := databricks.NewWorkspaceClient(&databricks.Config{
				Host:  server.URL,
				Token: "test-token",
			})
			require.NoError(t, err)

			root := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(root, "file.txt"), []byte("hello"), 0o600))
			opts := sync.SyncOptions{
				WorktreeRoot:     vfs.MustNew(root),
				LocalRoot:        vfs.MustNew(root),
				Paths:            []string{"."},
				RemotePath:       "/test-sync",
				SnapshotBasePath: t.TempDir(),
				Host:             server.URL,
				WorkspaceClient:  client,
				CurrentUser:      &iam.User{UserName: "test-user"},
				DryRun:           true,
			}
			var remoteID int64
			if tc.exists {
				require.NoError(t, client.Workspace.MkdirsByPath(ctx, opts.RemotePath))
				info, err := client.Workspace.GetStatusByPath(ctx, opts.RemotePath)
				require.NoError(t, err)
				remoteID = info.ObjectId
				require.NotZero(t, remoteID)
			}

			files, err := fileset.New(opts.LocalRoot).Files()
			require.NoError(t, err)
			snapshot, err := sync.NewSnapshot(files, &opts)
			require.NoError(t, err)
			for _, file := range files {
				snapshot.LastModifiedTimes[file.Relative] = file.Modified()
			}
			switch {
			case tc.legacy:
				snapshot.RemoteObjectID = 0
			case tc.replaced || !tc.exists:
				snapshot.RemoteObjectID = remoteID + 1
			default:
				snapshot.RemoteObjectID = remoteID
			}
			require.NoError(t, snapshot.Save(ctx))
			snapshotPath, err := sync.SnapshotPath(&opts)
			require.NoError(t, err)
			before, err := os.ReadFile(snapshotPath)
			require.NoError(t, err)

			require.NoError(t, sync.EnsureRemotePathIsUsable(ctx, client, opts.RemotePath, opts.CurrentUser, true))
			s, err := sync.New(ctx, opts)
			require.NoError(t, err)
			defer s.Close()
			_, err = s.RunOnce(ctx)
			require.NoError(t, err)
			assert.Equal(t, sync.FileCounts{Uploaded: tc.wantUploads}, s.FileCounts())
			after, err := os.ReadFile(snapshotPath)
			require.NoError(t, err)
			assert.Equal(t, before, after, "a dry run must not persist snapshot invalidation")
			if !tc.exists {
				_, err = client.Workspace.GetStatusByPath(ctx, opts.RemotePath)
				assert.True(t, apierr.IsMissing(err), "a dry run must not recreate the remote directory")
			}
		})
	}
}
