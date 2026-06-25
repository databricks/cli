package generate

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newGenieSpaceTestBundle(t *testing.T, m *mocks.MockWorkspaceClient, filePath string) *bundle.Bundle {
	t.Helper()
	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Resources: config.Resources{
				GenieSpaces: map[string]*resources.GenieSpace{
					"my_space": {
						GenieSpaceConfig: resources.GenieSpaceConfig{
							Title: "My Space",
						},
						FilePath: filePath,
					},
				},
			},
		},
	}
	b.Config.Resources.GenieSpaces["my_space"].ID = "space-id-1"
	b.SetWorkpaceClient(m.WorkspaceClient)
	return b
}

func TestGenieSpace_UpdateForResource_WritesFileWhenNotWatching(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "my_space.geniespace.json")

	m := mocks.NewMockWorkspaceClient(t)
	m.GetMockGenieAPI().EXPECT().
		GetSpace(mock.Anything, dashboards.GenieGetSpaceRequest{
			SpaceId:                "space-id-1",
			IncludeSerializedSpace: true,
		}).
		Return(&dashboards.GenieSpace{
			SpaceId:         "space-id-1",
			Title:           "My Space",
			SerializedSpace: `{"version":1}`,
		}, nil).
		Once()

	g := &genieSpace{
		resource: "my_space",
		force:    true,
	}
	b := newGenieSpaceTestBundle(t, m, filePath)

	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())
	ctx = logdiag.InitContext(ctx)
	logdiag.SetCollect(ctx, true)
	require.NoError(t, g.updateGenieSpaceForResource(ctx, b))

	require.Empty(t, logdiag.FlushCollected(ctx))

	contents, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Contains(t, string(contents), `"version"`)
}

func TestGenieSpace_UpdateForResource_WatchExitsOnCancel(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "my_space.geniespace.json")

	m := mocks.NewMockWorkspaceClient(t)
	// Allow any number of GetSpace calls; we don't know how many fire before cancel.
	m.GetMockGenieAPI().EXPECT().
		GetSpace(mock.Anything, dashboards.GenieGetSpaceRequest{
			SpaceId:                "space-id-1",
			IncludeSerializedSpace: true,
		}).
		Return(&dashboards.GenieSpace{
			SpaceId:         "space-id-1",
			Title:           "My Space",
			SerializedSpace: `{"version":1}`,
		}, nil).
		Maybe()

	g := &genieSpace{
		resource: "my_space",
		force:    true,
		watch:    true,
	}
	b := newGenieSpaceTestBundle(t, m, filePath)

	base, _ := cmdio.NewTestContextWithStdout(t.Context())
	ctx, cancel := context.WithCancel(logdiag.InitContext(base))
	logdiag.SetCollect(ctx, true)

	done := make(chan struct{})
	go func() {
		// Returns nil once the context is cancelled below; the test asserts the
		// initial save landed and that this goroutine exits promptly.
		_ = g.updateGenieSpaceForResource(ctx, b)
		close(done)
	}()

	// First iteration always saves. Wait until the file lands, then cancel.
	require.Eventually(t, func() bool {
		_, err := os.Stat(filePath)
		return err == nil
	}, 2*time.Second, 10*time.Millisecond, "expected initial save to write file")

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("watch loop did not exit promptly after ctx cancel")
	}
}
