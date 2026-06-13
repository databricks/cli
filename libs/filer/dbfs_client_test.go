package filer

import (
	"io/fs"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/files"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func dbfsReadWithGetStatusResponse(t *testing.T, response any) error {
	t.Helper()

	server := testserver.New(t)
	server.Handle("GET", "/api/2.0/dbfs/get-status", func(req testserver.Request) any {
		return response
	})
	testserver.AddDefaultHandlers(server)

	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)

	f, err := NewDbfsClient(client, "/test")
	require.NoError(t, err)

	_, err = f.Read(t.Context(), "file")
	require.Error(t, err)
	return err
}

func TestDbfsClientReadDirectory(t *testing.T) {
	err := dbfsReadWithGetStatusResponse(t, files.FileInfo{
		Path:  "/test/file",
		IsDir: true,
	})
	assert.ErrorIs(t, err, fs.ErrInvalid)
}

func TestDbfsClientReadFileDoesNotExist(t *testing.T) {
	err := dbfsReadWithGetStatusResponse(t, testserver.Response{
		StatusCode: 404,
		Body: map[string]string{
			"error_code": "RESOURCE_DOES_NOT_EXIST",
			"message":    "test error",
		},
	})
	assert.ErrorIs(t, err, fs.ErrNotExist)
}
