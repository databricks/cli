package workspaceapi

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/apiclient/apiclienttest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUploadSendsMultipart(t *testing.T) {
	var fields map[string]string
	c := New(apiclienttest.New(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/2.0/workspace/import", r.URL.Path)
		assert.Equal(t, "123", r.Header.Get("X-Databricks-Workspace-Id"))
		assert.Equal(t, "Bearer token", r.Header.Get("Authorization"))
		require.NoError(t, r.ParseMultipartForm(1<<20))
		fields = map[string]string{}
		for k, v := range r.MultipartForm.Value {
			fields[k] = v[0]
		}
		f, _, err := r.FormFile("content")
		require.NoError(t, err)
		data, _ := io.ReadAll(f)
		fields["content"] = string(data)
	}))
	err := c.Upload(t.Context(), "/a/b.py", strings.NewReader("print(1)"), UploadOverwrite())
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"path":      "/a/b.py",
		"language":  "PYTHON",
		"overwrite": "true",
		"content":   "print(1)",
	}, fields)
}

func TestDownload(t *testing.T) {
	c := New(apiclienttest.New(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/2.0/workspace/export", r.URL.Path)
		assert.Equal(t, "/a/b", r.URL.Query().Get("path"))
		assert.Equal(t, "true", r.URL.Query().Get("direct_download"))
		assert.Equal(t, "JUPYTER", r.URL.Query().Get("format"))
		_, _ = w.Write([]byte("raw bytes"))
	}))
	data, err := c.Download(t.Context(), "/a/b", DownloadFormat(ExportFormatJupyter))
	require.NoError(t, err)
	b, _ := io.ReadAll(data)
	assert.Equal(t, "raw bytes", string(b))
}

func TestGetStatusAndListUseQuery(t *testing.T) {
	c := New(apiclienttest.New(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/workspace/get-status":
			assert.Equal(t, "path=%2Fa%20b", r.URL.RawQuery)
			_ = json.NewEncoder(w).Encode(ObjectInfo{Path: "/a b", ObjectType: ObjectTypeDirectory, ObjectId: 7})
		case "/api/2.0/workspace/list":
			assert.Equal(t, "/a b", r.URL.Query().Get("path"))
			_ = json.NewEncoder(w).Encode(ListResponse{Objects: []ObjectInfo{{Path: "/a b/x", ObjectType: ObjectTypeFile}}})
		default:
			t.Errorf("unexpected %s", r.URL.Path)
		}
	}))
	info, err := c.GetStatusByPath(t.Context(), "/a b")
	require.NoError(t, err)
	assert.Equal(t, int64(7), info.ObjectId)

	objs, err := c.RecursiveList(t.Context(), "/a b")
	require.NoError(t, err)
	assert.Equal(t, []ObjectInfo{{Path: "/a b/x", ObjectType: ObjectTypeFile}}, objs)
}
