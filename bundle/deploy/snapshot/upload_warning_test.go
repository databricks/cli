package snapshot

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/snapshot"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestClient(t *testing.T) *databricks.WorkspaceClient {
	t.Helper()
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:               server.URL,
		Token:              "testtoken",
		RateLimitPerSecond: math.MaxInt,
	})
	require.NoError(t, err)
	return client
}

func makeBundle(t *testing.T, nFiles int) *bundle.Bundle {
	t.Helper()
	dir := t.TempDir()
	for i := range nFiles {
		p := filepath.Join(dir, fmt.Sprintf("f%d.py", i))
		require.NoError(t, os.WriteFile(p, []byte("x"), 0o644))
	}
	root := vfs.MustNew(dir)
	b := &bundle.Bundle{
		BundleRootPath: dir,
		SyncRoot:       root,
		WorktreeRoot:   root,
		Config: config.Root{
			Bundle: config.Bundle{Target: "default"},
			// The SyncDefaultPath mutator sets this to ["."] during initialize;
			// set it here since these tests bypass the mutator pipeline. Empty
			// sync paths select no files.
			Sync: config.Sync{Paths: []string{"."}},
			Workspace: config.Workspace{
				CurrentUser: &config.User{
					User: &iam.User{UserName: "test@example.test"},
				},
			},
		},
	}
	return b
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	return logdiag.InitContext(cmdio.MockDiscard(t.Context()))
}

func TestUploadWarnsAboveFileLimit(t *testing.T) {
	b := makeBundle(t, fileLimitWarning+1)
	b.SetWorkpaceClient(setupTestClient(t))
	m := &snapshotUpload{clean: true}

	diags := m.Apply(testContext(t), b)

	require.Len(t, diags, 1)
	assert.Equal(t, diag.Warning, diags[0].Severity)
	assert.Contains(t, diags[0].Summary, fmt.Sprintf("%d files", fileLimitWarning+1))
}

func TestUploadNoWarningBelowFileLimit(t *testing.T) {
	b := makeBundle(t, 5)
	b.SetWorkpaceClient(setupTestClient(t))
	m := &snapshotUpload{clean: true}

	diags := m.Apply(testContext(t), b)

	assert.True(t, diags.HasError() == false && len(diags) == 0, "expected no diagnostics")
}

// canManageBundle returns a bundle deployed by deployer@example.test with the given permissions.
func canManageBundle(permissions []resources.Permission) *bundle.Bundle {
	return &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				CurrentUser: &config.User{
					User: &iam.User{UserName: "deployer@example.test"},
				},
			},
			Permissions: permissions,
		},
	}
}

func TestBuildCanManage(t *testing.T) {
	b := canManageBundle([]resources.Permission{
		{Level: "CAN_MANAGE", UserName: "manager@example.test"},
		{Level: "CAN_VIEW", UserName: "viewer@example.test"},
		{Level: "CAN_MANAGE", GroupName: "admins"},
		{Level: "CAN_MANAGE", ServicePrincipalName: "sp-1"},
	})

	// The deploying identity, then only the CAN_MANAGE principals.
	assert.Equal(t, []snapshot.ManagePrincipal{
		{UserName: "deployer@example.test"},
		{UserName: "manager@example.test"},
		{GroupName: "admins"},
		{ServicePrincipalName: "sp-1"},
	}, BuildCanManage(b))
}

func TestBuildCanManageWithoutManagers(t *testing.T) {
	b := canManageBundle([]resources.Permission{
		{Level: "CAN_VIEW", UserName: "viewer@example.test"},
	})

	// The API rejects an empty list, so the deploying identity is always there.
	assert.Equal(t, []snapshot.ManagePrincipal{
		{UserName: "deployer@example.test"},
	}, BuildCanManage(b))
}

func TestUploadReusesStagedZipWhenNotClean(t *testing.T) {
	// clean=false is the deploy --plan path: the zip staged when the plan was produced is
	// registered on the config as is, without rebuilding or warning again, even for a
	// bundle that would otherwise exceed the file limit.
	ctx := testContext(t)
	b := makeBundle(t, fileLimitWarning+1)
	b.SetWorkpaceClient(setupTestClient(t))

	require.Len(t, (&snapshotUpload{clean: true}).Apply(ctx, b), 1)
	staged := b.Config.Resources.Snapshots[resources.SnapshotResourceKey]
	require.NotNil(t, staged)

	b.Config.Resources.Snapshots = nil
	diags := (&snapshotUpload{clean: false}).Apply(ctx, b)

	require.Empty(t, diags)
	reused := b.Config.Resources.Snapshots[resources.SnapshotResourceKey]
	require.NotNil(t, reused)
	assert.Equal(t, staged.ZipPath, reused.ZipPath)
	assert.Equal(t, staged.FullPath(), reused.FullPath())
}

func TestUploadNotCleanWithoutStagedZip(t *testing.T) {
	b := makeBundle(t, 5)
	b.SetWorkpaceClient(setupTestClient(t))

	diags := (&snapshotUpload{clean: false}).Apply(testContext(t), b)

	require.True(t, diags.HasError())
	assert.Contains(t, diags[0].Summary, "run 'bundle plan' again")
}

func TestSnapshotName(t *testing.T) {
	const hash = "e3b0c4"
	const prevName = hash + "-0199e2c1-7b1a-7c2d-8e3f-4a5b6c7d8e9f"
	snap := &resources.Snapshot{
		BundleID:   "bundle-id",
		ACL:        []snapshot.ACLEntry{{UserName: "deployer@example.test", PermissionLevel: "CAN_READ"}},
		CanManage:  []snapshot.ManagePrincipal{{UserName: "deployer@example.test"}},
		RemoteRoot: "/Workspace/Users/sp/.snapshots",
	}
	unchanged := &dresources.SnapshotState{
		RelativePath: "bundle-id/" + prevName,
		FullPath:     "/Workspace/Users/sp/.snapshots/bundle-id/" + prevName + "/snapshot",
		BundleID:     "bundle-id",
		ACL:          []snapshot.ACLEntry{{UserName: "deployer@example.test", PermissionLevel: "CAN_READ"}},
		CanManage:    []snapshot.ManagePrincipal{{UserName: "deployer@example.test"}},
		ZipPath:      "/stage/" + prevName + ".zip",
	}
	otherManagers := *unchanged
	otherManagers.CanManage = []snapshot.ManagePrincipal{{GroupName: "admins"}}

	tests := []struct {
		name   string
		prev   *dresources.SnapshotState
		dir    string
		hash   string
		reused bool
	}{
		{name: "no previous snapshot", prev: nil, dir: "/stage", hash: hash, reused: false},
		{name: "unchanged", prev: unchanged, dir: "/stage", hash: hash, reused: true},
		{name: "content changed", prev: unchanged, dir: "/stage", hash: "a1b2c3", reused: false},
		{name: "staged elsewhere", prev: unchanged, dir: "/other", hash: hash, reused: false},
		{name: "managers changed", prev: &otherManagers, dir: "/stage", hash: hash, reused: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			name := snapshotName(snap, tc.prev, tc.dir, tc.hash)
			if tc.reused {
				assert.Equal(t, prevName, name)
				return
			}
			assert.NotEqual(t, prevName, name)
			assert.True(t, strings.HasPrefix(name, tc.hash+"-"), name)
		})
	}
}

func TestSnapshotNameIsUniquePerUpload(t *testing.T) {
	snap := &resources.Snapshot{BundleID: "bundle-id"}

	first := snapshotName(snap, nil, "/stage", "e3b0c4")
	second := snapshotName(snap, nil, "/stage", "e3b0c4")

	assert.NotEqual(t, first, second)
}
