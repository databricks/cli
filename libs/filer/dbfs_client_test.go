package filer

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/files"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockDbfsApiClient struct {
	t        testutil.TestingT
	isCalled bool
}

func (m *mockDbfsApiClient) Do(ctx context.Context, method, path string,
	headers map[string]string, request, response any,
	visitors ...func(*http.Request) error,
) error {
	m.isCalled = true

	require.Equal(m.t, "POST", method)
	require.Equal(m.t, "/api/2.0/dbfs/put", path)
	require.Contains(m.t, headers["Content-Type"], "multipart/form-data; boundary=")
	require.Contains(m.t, string(request.([]byte)), "hello world")
	return nil
}

func TestDbfsClientForSmallFiles(t *testing.T) {
	// write file to local disk
	tmp := t.TempDir()
	localPath := filepath.Join(tmp, "hello.txt")
	err := os.WriteFile(localPath, []byte("hello world"), 0o644)
	require.NoError(t, err)

	// setup DBFS client with mocks
	m := mocks.NewMockWorkspaceClient(t)
	mockApiClient := &mockDbfsApiClient{t: t}
	dbfsClient := DbfsClient{
		apiClient:       mockApiClient,
		workspaceClient: m.WorkspaceClient,
		root:            NewWorkspaceRootPath("dbfs:/a/b/c"),
	}

	m.GetMockDbfsAPI().EXPECT().GetStatusByPath(mock.Anything, "dbfs:/a/b/c").Return(nil, nil)

	// write file to DBFS
	fd, err := os.Open(localPath)
	require.NoError(t, err)
	err = dbfsClient.Write(context.Background(), "hello.txt", fd)
	require.NoError(t, err)

	// verify mock API client is called
	require.True(t, mockApiClient.isCalled)
}

type mockDbfsHandle struct {
	builder strings.Builder
}

func (h *mockDbfsHandle) Read(data []byte) (n int, err error)      { return 0, nil }
func (h *mockDbfsHandle) Close() error                             { return nil }
func (h *mockDbfsHandle) WriteTo(w io.Writer) (n int64, err error) { return 0, nil }

func (h *mockDbfsHandle) ReadFrom(r io.Reader) (n int64, err error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return 0, err
	}
	num, err := h.builder.Write(b)
	return int64(num), err
}

func (h *mockDbfsHandle) Write(data []byte) (n int, err error) {
	return h.builder.Write(data)
}

func TestDbfsClientForLargerFiles(t *testing.T) {
	// write file to local disk
	tmp := t.TempDir()
	localPath := filepath.Join(tmp, "hello.txt")
	err := os.WriteFile(localPath, []byte("hello world"), 0o644)
	require.NoError(t, err)

	// Modify the max file size to 1 byte to simulate
	// a large file that needs to be uploaded in chunks.
	oldV := MaxDbfsPutFileSize
	MaxDbfsPutFileSize = 1
	t.Cleanup(func() {
		MaxDbfsPutFileSize = oldV
	})

	// setup DBFS client with mocks
	m := mocks.NewMockWorkspaceClient(t)
	mockApiClient := &mockDbfsApiClient{t: t}
	dbfsClient := DbfsClient{
		apiClient:       mockApiClient,
		workspaceClient: m.WorkspaceClient,
		root:            NewWorkspaceRootPath("dbfs:/a/b/c"),
	}

	h := &mockDbfsHandle{}
	m.GetMockDbfsAPI().EXPECT().GetStatusByPath(mock.Anything, "dbfs:/a/b/c").Return(nil, nil)
	m.GetMockDbfsAPI().EXPECT().Open(mock.Anything, "dbfs:/a/b/c/hello.txt", files.FileModeWrite).Return(h, nil)

	// write file to DBFS
	fd, err := os.Open(localPath)
	require.NoError(t, err)
	err = dbfsClient.Write(context.Background(), "hello.txt", fd)
	require.NoError(t, err)

	// verify mock API client is NOT called
	require.False(t, mockApiClient.isCalled)

	// verify the file content was written to the mock handle
	assert.Equal(t, "hello world", h.builder.String())
}

func TestDbfsClientForNonLocalFiles(t *testing.T) {
	// setup DBFS client with mocks
	m := mocks.NewMockWorkspaceClient(t)
	mockApiClient := &mockDbfsApiClient{t: t}
	dbfsClient := DbfsClient{
		apiClient:       mockApiClient,
		workspaceClient: m.WorkspaceClient,
		root:            NewWorkspaceRootPath("dbfs:/a/b/c"),
	}

	h := &mockDbfsHandle{}
	m.GetMockDbfsAPI().EXPECT().GetStatusByPath(mock.Anything, "dbfs:/a/b/c").Return(nil, nil)
	m.GetMockDbfsAPI().EXPECT().Open(mock.Anything, "dbfs:/a/b/c/hello.txt", files.FileModeWrite).Return(h, nil)

	// write file to DBFS
	err := dbfsClient.Write(context.Background(), "hello.txt", strings.NewReader("hello world"))
	require.NoError(t, err)

	// verify mock API client is NOT called
	require.False(t, mockApiClient.isCalled)

	// verify the file content was written to the mock handle
	assert.Equal(t, "hello world", h.builder.String())
}
