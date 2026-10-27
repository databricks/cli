package filer

import (
	"strings"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/useragent"
	files "github.com/databricks/sdk-go/files/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilesAPIClientCarriesCommandUserAgent(t *testing.T) {
	var gotUserAgent string
	server := testserver.New(t)
	server.Handle("HEAD", "/api/2.0/fs/files/{path...}", func(req testserver.Request) any {
		gotUserAgent = req.Headers.Get("User-Agent")
		return ""
	})

	workspaceClient, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "token",
	})
	require.NoError(t, err)

	// The registered segments are process-wide, so the values are unique to
	// this test to keep it independent of other tests in the package.
	ctx := useragent.InContext(t.Context(), "cmd", "test_files-user-agent")
	ctx = useragent.InContext(ctx, "engine", "direct")

	client, err := newFilesAPIClient(ctx, workspaceClient.Config)
	require.NoError(t, err)

	filePath := "/Volumes/main/schema/volume/file"
	_, err = client.GetFileMetadata(ctx, files.GetFileMetadataRequest{FilePath: &filePath})
	require.NoError(t, err)

	assert.Contains(t, gotUserAgent, " cmd/test_files-user-agent engine/direct")
	assert.NotContains(t, gotUserAgent, "databricks-sdk-go/")
	assert.Equal(t, 1, strings.Count(gotUserAgent, " auth/"), gotUserAgent)
	assert.Equal(t, 1, strings.Count(gotUserAgent, " go/"), gotUserAgent)
}
