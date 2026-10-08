package bundle

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	rootcmd "github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitWithVolumeConfigFile(t *testing.T) {
	server := testserver.New(t)
	var requests int
	server.Handle("GET", "/api/2.0/fs/files/{path...}", func(req testserver.Request) any {
		requests++
		assert.Equal(t, "/api/2.0/fs/files/Volumes/main/schema/volume/config.json", req.URL.Path)
		return `{"project_name":"volume_project"}`
	})
	server.Handle("GET", "/api/2.1/unity-catalog/current-metastore-assignment", func(req testserver.Request) any {
		return testserver.Response{
			StatusCode: 404,
			Body: map[string]string{
				"error_code": "FEATURE_DISABLED",
				"message":    "Unity Catalog is not available",
			},
		}
	})
	server.Handle("GET", "/api/2.0/preview/scim/v2/Me", func(req testserver.Request) any {
		return map[string]string{
			"userName":    "test@example.com",
			"displayName": "Test User",
		}
	})

	workspaceClient, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "test-token",
	})
	require.NoError(t, err)

	ctx := t.Context()
	root := rootcmd.New(ctx)
	root.AddGroup(&cobra.Group{ID: "development"})
	root.AddCommand(New())
	cmd, _, err := root.Find([]string{"bundle", "init"})
	require.NoError(t, err)
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		cmd.SetContext(cmdctx.SetWorkspaceClient(cmd.Context(), workspaceClient))
		return nil
	}
	outputDir := t.TempDir()
	root.SetIn(strings.NewReader(""))
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SetArgs([]string{
		"bundle",
		"init",
		"default-minimal",
		"--config-file", "dbfs:/Volumes/main/schema/volume/config.json",
		"--output-dir", outputDir,
	})

	require.NoError(t, rootcmd.Execute(ctx, root))
	assert.Equal(t, 1, requests)

	config, err := os.ReadFile(filepath.Join(outputDir, "volume_project", "databricks.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(config), "name: volume_project")
}
