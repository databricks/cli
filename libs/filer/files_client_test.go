package filer

import (
	"io/fs"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func deleteDirectoryWithError(t *testing.T, statusCode int, errorCode, reason string) error {
	t.Helper()

	server := testserver.New(t)
	server.Handle("DELETE", "/api/2.0/fs/directories/{path...}", func(req testserver.Request) any {
		return testserver.Response{
			StatusCode: statusCode,
			Body: map[string]any{
				"error_code": errorCode,
				"message":    "test error",
				"details": []map[string]any{
					{
						"@type":  "type.googleapis.com/google.rpc.ErrorInfo",
						"reason": reason,
					},
				},
			},
		}
	})
	testserver.AddDefaultHandlers(server)

	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)

	f, err := NewFilesClient(client, "/test")
	require.NoError(t, err)

	return f.(*FilesClient).deleteDirectory(t.Context(), "dir")
}

func TestFilesClientDeleteDirectoryNotFound(t *testing.T) {
	// A GCS-backed implicit directory can vanish once its last child is deleted,
	// so the delete API returns 404 FILES_API_DIRECTORY_IS_NOT_FOUND. It must
	// map to a not-found error so recursive delete can tolerate it.
	err := deleteDirectoryWithError(t, 404, "NOT_FOUND", "FILES_API_DIRECTORY_IS_NOT_FOUND")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestFilesClientDeleteDirectoryNotEmpty(t *testing.T) {
	err := deleteDirectoryWithError(t, 400, "INVALID_PARAMETER_VALUE", "FILES_API_DIRECTORY_IS_NOT_EMPTY")
	assert.ErrorIs(t, err, fs.ErrInvalid)
}
