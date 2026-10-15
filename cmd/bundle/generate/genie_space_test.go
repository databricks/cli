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
	"github.com/databricks/cli/libs/httpfixtures"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newGenieSpaceTestBundle(t *testing.T, m *sdkclients.Clients, filePath string) *bundle.Bundle {
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
	b.SetWorkpaceClient(m)
	return b
}

func TestGenieSpace_UpdateForResource_WritesFileWhenNotWatching(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "my_space.geniespace.json")

	m := genieSpaceClients(t)

	g := &genieSpace{
		resource: "my_space",
		force:    true,
	}
	b := newGenieSpaceTestBundle(t, m, filePath)

	ctx, _ := cmdio.NewTestContextWithStdout(t.Context())
	ctx = logdiag.InitContext(ctx)
	logdiag.SetCollect(ctx, true)
	g.updateGenieSpaceForResource(ctx, b)

	require.Empty(t, logdiag.FlushCollected(ctx))

	contents, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Contains(t, string(contents), `"version"`)
}

func TestGenieSpace_UpdateForResource_WatchExitsOnCancel(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "my_space.geniespace.json")

	m := genieSpaceClients(t)

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
		g.updateGenieSpaceForResource(ctx, b)
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

// genieSpaceClients serves the Genie space of newGenieSpaceTestBundle.
func genieSpaceClients(t *testing.T) *sdkclients.Clients {
	t.Helper()
	cfg, _ := httpfixtures.Fixtures{{
		Method:       "GET",
		Resource:     "/api/2.0/genie/spaces/space-id-1?include_serialized_space=true",
		Response:     `{"space_id":"space-id-1","title":"My Space","serialized_space":"{\"version\":1}"}`,
		ReuseRequest: true,
	}}.Config(t)
	w, err := sdkclients.New(t.Context(), cfg)
	require.NoError(t, err)
	return w
}
