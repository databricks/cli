package testserver_test

import (
	"encoding/base64"
	"errors"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/files"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newDbfsTestClient(t *testing.T) *databricks.WorkspaceClient {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)
	return w
}

func dbfsErrorCode(err error) string {
	if aerr, ok := errors.AsType[*apierr.APIError](err); ok {
		return aerr.ErrorCode
	}
	return ""
}

func TestDbfsEnforcesBlockAndReadLimits(t *testing.T) {
	ctx := t.Context()
	w := newDbfsTestClient(t)

	created, err := w.Dbfs.Create(ctx, files.Create{Path: "/tmp/f"})
	require.NoError(t, err)
	err = w.Dbfs.AddBlock(ctx, files.AddBlock{Handle: created.Handle, Data: base64.StdEncoding.EncodeToString(make([]byte, 1<<20+1))})
	assert.Equal(t, "MAX_BLOCK_SIZE_EXCEEDED", dbfsErrorCode(err))
	require.NoError(t, w.Dbfs.AddBlock(ctx, files.AddBlock{Handle: created.Handle, Data: base64.StdEncoding.EncodeToString([]byte("abc"))}))
	require.NoError(t, w.Dbfs.CloseByHandle(ctx, created.Handle))

	_, err = w.Dbfs.Read(ctx, files.ReadDbfsRequest{Path: "/tmp/f", Length: 1<<20 + 1})
	assert.Equal(t, "MAX_READ_SIZE_EXCEEDED", dbfsErrorCode(err))
	_, err = w.Dbfs.Read(ctx, files.ReadDbfsRequest{Path: "/tmp/f", Offset: -1, Length: 1})
	assert.Equal(t, "INVALID_PARAMETER_VALUE", dbfsErrorCode(err))

	res, err := w.Dbfs.Read(ctx, files.ReadDbfsRequest{Path: "/tmp/f", Offset: 1, Length: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(2), res.BytesRead)
}

func TestDbfsRecursiveDeleteOfRootKeepsRoot(t *testing.T) {
	ctx := t.Context()
	w := newDbfsTestClient(t)

	require.NoError(t, w.Dbfs.MkdirsByPath(ctx, "/a/b"))
	require.NoError(t, w.Dbfs.Delete(ctx, files.Delete{Path: "/", Recursive: true}))

	info, err := w.Dbfs.GetStatusByPath(ctx, "/")
	require.NoError(t, err)
	assert.True(t, info.IsDir)
	_, err = w.Dbfs.GetStatusByPath(ctx, "/a")
	assert.Equal(t, "RESOURCE_DOES_NOT_EXIST", dbfsErrorCode(err))

	// Used to loop forever once the root itself had been deleted.
	require.NoError(t, w.Dbfs.MkdirsByPath(ctx, "/c"))
}
