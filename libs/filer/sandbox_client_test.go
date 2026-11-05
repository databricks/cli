package filer_test

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/databricks/databricks-sdk-go/service/sandbox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sandboxFilesRoute = "/api/2.0/sandbox-files/sandboxes/test-sandbox"

func newSandboxFiler(t *testing.T, server *testserver.Server, root, workspaceID string) filer.Filer {
	t.Helper()
	w, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host: server.URL, Token: "testtoken", WorkspaceID: workspaceID,
	})
	require.NoError(t, err)
	f, err := filer.NewSandboxClient(w, "test-sandbox", root)
	require.NoError(t, err)
	return f
}

func TestSandboxWriteOverwrite(t *testing.T) {
	for _, overwrite := range []bool{false, true} {
		t.Run(strconv.FormatBool(overwrite), func(t *testing.T) {
			server := testserver.New(t)
			server.Handle("PUT", sandboxFilesRoute+"/content", func(req testserver.Request) any {
				assert.Equal(t, "/root/a ?#雪.txt", req.URL.Query().Get("path"))
				assert.Equal(t, []string{strconv.FormatBool(overwrite)}, req.URL.Query()["overwrite"])
				assert.Equal(t, "application/octet-stream", req.Headers.Get("Content-Type"))
				assert.Equal(t, "123", req.Headers.Get(auth.WorkspaceIDHeader))
				assert.Equal(t, "contents", string(req.Body))
				if !overwrite {
					return testserver.Response{StatusCode: 412, Body: map[string]string{
						"error_code": "RESOURCE_ALREADY_EXISTS", "message": "already exists",
					}}
				}
				return testserver.Response{StatusCode: 204}
			})
			f := newSandboxFiler(t, server, "/root", "123")
			mode := []filer.WriteMode{filer.CreateParentDirectories}
			if overwrite {
				mode = append(mode, filer.OverwriteIfExists)
			}
			err := f.Write(t.Context(), "a ?#雪.txt", strings.NewReader("contents"), mode...)
			if overwrite {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, fs.ErrExist)
			}
		})
	}
}

func TestSandboxWriteParent(t *testing.T) {
	for _, fileType := range []sandbox.SandboxFileType{
		sandbox.SandboxFileTypeSandboxFileTypeDirectory,
		sandbox.SandboxFileTypeSandboxFileTypeFile,
		sandbox.SandboxFileTypeSandboxFileTypeSymlink,
	} {
		t.Run(string(fileType), func(t *testing.T) {
			server := testserver.New(t)
			server.Handle("GET", sandboxFilesRoute+"/paths:describe", func(req testserver.Request) any {
				assert.Equal(t, "/root/sub", req.URL.Query().Get("path"))
				return sandbox.GetMetadataResponse{File: &sandbox.SandboxFileInfo{FileType: fileType}}
			})
			uploaded := false
			server.Handle("PUT", sandboxFilesRoute+"/content", func(req testserver.Request) any {
				uploaded = true
				assert.Empty(t, req.Headers.Get(auth.WorkspaceIDHeader))
				return testserver.Response{StatusCode: 204}
			})
			f := newSandboxFiler(t, server, "/root", "none")
			err := f.Write(t.Context(), "sub/file", strings.NewReader("content"))
			if fileType != sandbox.SandboxFileTypeSandboxFileTypeFile {
				require.NoError(t, err)
				assert.True(t, uploaded)
			} else {
				require.ErrorIs(t, err, fs.ErrInvalid)
				assert.False(t, uploaded)
			}
		})
	}
}

func TestSandboxReadAndMetadata(t *testing.T) {
	server := testserver.New(t)
	modified := time.Date(2026, 11, 5, 1, 2, 3, 123456789, time.UTC)
	server.Handle("GET", sandboxFilesRoute+"/content", func(req testserver.Request) any {
		assert.Equal(t, "/tmp/file", req.URL.Query().Get("path"))
		return testserver.Response{Body: []byte("contents"), Headers: http.Header{"Content-Type": {"application/octet-stream"}}}
	})
	server.Handle("GET", sandboxFilesRoute+"/paths:describe", func(req testserver.Request) any {
		return sandbox.GetMetadataResponse{File: &sandbox.SandboxFileInfo{
			Path: "/tmp/file", Size: 8, Mode: 0o640, ModifyTime: sdktime.New(modified),
			FileType: sandbox.SandboxFileTypeSandboxFileTypeFile,
		}}
	})
	f := newSandboxFiler(t, server, "/", "")
	r, err := f.Read(t.Context(), "/tmp/file")
	require.NoError(t, err)
	defer r.Close()
	contents, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, "contents", string(contents))
	info, err := f.Stat(t.Context(), "/tmp/file")
	require.NoError(t, err)
	assert.Equal(t, "file", info.Name())
	assert.Equal(t, int64(8), info.Size())
	assert.Equal(t, fs.FileMode(0o640), info.Mode())
	assert.Equal(t, modified, info.ModTime())
	assert.False(t, info.IsDir())
}

func TestSandboxReadDirPagination(t *testing.T) {
	server := testserver.New(t)
	var tokens []string
	server.Handle("GET", sandboxFilesRoute+"/directory-entries", func(req testserver.Request) any {
		assert.Equal(t, "/tmp", req.URL.Query().Get("path"))
		token := req.URL.Query().Get("page_token")
		tokens = append(tokens, token)
		if token == "" {
			return sandbox.ListDirectoryContentsResponse{
				Contents:      []sandbox.SandboxFileInfo{{Path: "/tmp/z", FileType: sandbox.SandboxFileTypeSandboxFileTypeSymlink}},
				NextPageToken: "page2",
			}
		}
		return sandbox.ListDirectoryContentsResponse{Contents: []sandbox.SandboxFileInfo{
			{Path: "/tmp/a", FileType: sandbox.SandboxFileTypeSandboxFileTypeDirectory},
		}}
	})
	f := newSandboxFiler(t, server, "/", "")
	entries, err := f.ReadDir(t.Context(), "/tmp")
	require.NoError(t, err)
	require.Len(t, entries, 2)
	assert.Equal(t, []string{"", "page2"}, tokens)
	assert.Equal(t, "a", entries[0].Name())
	assert.True(t, entries[0].IsDir())
	assert.Equal(t, "z", entries[1].Name())
	assert.Equal(t, fs.ModeSymlink, entries[1].Type())
}

func TestSandboxMkdirAndDelete(t *testing.T) {
	server := testserver.New(t)
	server.Handle("GET", sandboxFilesRoute+"/paths:describe", func(req testserver.Request) any {
		return sandbox.GetMetadataResponse{File: &sandbox.SandboxFileInfo{
			Path: "/root/sub", FileType: sandbox.SandboxFileTypeSandboxFileTypeDirectory,
		}}
	})
	server.Handle("POST", sandboxFilesRoute+"/directories:make", func(req testserver.Request) any {
		var body sandbox.CreateDirectoryRequest
		require.NoError(t, json.Unmarshal(req.Body, &body))
		assert.Equal(t, "/root/sub", body.Path)
		assert.True(t, body.Parents)
		return sandbox.CreateDirectoryResponse{}
	})
	server.Handle("POST", sandboxFilesRoute+"/paths:remove", func(req testserver.Request) any {
		var body sandbox.DeletePathRequest
		require.NoError(t, json.Unmarshal(req.Body, &body))
		assert.Equal(t, "/root/sub", body.Path)
		assert.True(t, body.Recursive)
		return sandbox.DeletePathResponse{}
	})
	f := newSandboxFiler(t, server, "/root", "")
	require.NoError(t, f.Mkdir(t.Context(), "sub"))
	require.NoError(t, f.Delete(t.Context(), "sub", filer.DeleteRecursively))
	require.ErrorIs(t, f.Delete(t.Context(), ".", filer.DeleteRecursively), fs.ErrInvalid)
	require.Error(t, f.Mkdir(t.Context(), "../escape"))
}

func TestSandboxPathErrors(t *testing.T) {
	for _, tc := range []struct {
		status int
		code   string
		want   error
	}{
		{404, "NOT_FOUND", fs.ErrNotExist},
		{403, "PERMISSION_DENIED", fs.ErrPermission},
		{400, "INVALID_PARAMETER_VALUE", apierr.ErrInvalidParameterValue},
	} {
		t.Run(tc.code, func(t *testing.T) {
			server := testserver.New(t)
			server.Handle("GET", sandboxFilesRoute+"/paths:describe", func(req testserver.Request) any {
				return testserver.Response{StatusCode: tc.status, Body: map[string]string{"error_code": tc.code, "message": "failed"}}
			})
			f := newSandboxFiler(t, server, "/", "")
			_, err := f.Stat(t.Context(), "missing")
			require.ErrorIs(t, err, tc.want)
		})
	}
}
