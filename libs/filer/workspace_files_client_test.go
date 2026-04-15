package filer

import (
	"encoding/json"
	"io/fs"
	"testing"
	"time"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceFilesDirEntry(t *testing.T) {
	entries := wsfsDirEntriesFromObjectInfos([]workspace.ObjectInfo{
		{
			Path:       "/dir",
			ObjectType: workspace.ObjectTypeDirectory,
		},
		{
			Path:       "/file",
			ObjectType: workspace.ObjectTypeFile,
			Size:       42,
		},
		{
			Path:       "/repo",
			ObjectType: workspace.ObjectTypeRepo,
		},
	})

	// Confirm the path is passed through correctly.
	assert.Equal(t, "dir", entries[0].Name())
	assert.Equal(t, "file", entries[1].Name())
	assert.Equal(t, "repo", entries[2].Name())

	// Confirm the type is passed through correctly.
	assert.Equal(t, fs.ModeDir, entries[0].Type())
	assert.Equal(t, fs.ModePerm, entries[1].Type())
	assert.Equal(t, fs.ModeDir, entries[2].Type())

	// Get [fs.FileInfo] from directory entry.
	i0, err := entries[0].Info()
	require.NoError(t, err)
	i1, err := entries[1].Info()
	require.NoError(t, err)
	i2, err := entries[2].Info()
	require.NoError(t, err)

	// Confirm size.
	assert.Equal(t, int64(0), i0.Size())
	assert.Equal(t, int64(42), i1.Size())
	assert.Equal(t, int64(0), i2.Size())

	// Confirm IsDir.
	assert.True(t, i0.IsDir())
	assert.False(t, i1.IsDir())
	assert.True(t, i2.IsDir())
}

func TestWorkspaceFilesClientOrgIDHeaders(t *testing.T) {
	tests := []struct {
		name        string
		workspaceID string
		expect      map[string]string
	}{
		{
			name:        "with workspace ID",
			workspaceID: "7474644166319138",
			expect:      map[string]string{"X-Databricks-Org-Id": "7474644166319138"},
		},
		{
			name:        "without workspace ID",
			workspaceID: "",
			expect:      nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := &WorkspaceFilesClient{
				workspaceClient: &databricks.WorkspaceClient{
					Config: &config.Config{
						WorkspaceID: tc.workspaceID,
					},
				},
			}
			assert.Equal(t, tc.expect, w.orgIDHeaders())
		})
	}

	t.Run("nil workspace client", func(t *testing.T) {
		w := &WorkspaceFilesClient{}
		assert.Nil(t, w.orgIDHeaders())
	})
}

func TestWorkspaceFilesClient_wsfsUnmarshal(t *testing.T) {
	payload := `
		{
			"created_at": 1671030805916,
			"language": "PYTHON",
			"modified_at": 1671032235392,
			"object_id": 795822750063438,
			"object_type": "NOTEBOOK",
			"path": "/some/path/to/a/notebook",
			"repos_export_format": "SOURCE",
			"resource_id": "795822750063438"
		}
	`

	var info wsfsFileInfo
	err := json.Unmarshal([]byte(payload), &info)
	require.NoError(t, err)

	// Fields in the object info.
	assert.Equal(t, int64(1671030805916), info.CreatedAt)
	assert.Equal(t, workspace.LanguagePython, info.Language)
	assert.Equal(t, int64(1671032235392), info.ModifiedAt)
	assert.Equal(t, int64(795822750063438), info.ObjectId)
	assert.Equal(t, workspace.ObjectTypeNotebook, info.ObjectType)
	assert.Equal(t, "/some/path/to/a/notebook", info.Path)
	assert.Equal(t, workspace.ExportFormatSource, info.ReposExportFormat)
	assert.Equal(t, "795822750063438", info.ResourceId)

	// Functions for fs.FileInfo.
	assert.Equal(t, "notebook", info.Name())
	assert.Equal(t, int64(0), info.Size())
	assert.Equal(t, fs.ModePerm, info.Mode())
	assert.Equal(t, time.UnixMilli(1671032235392), info.ModTime())
	assert.False(t, info.IsDir())
	assert.NotNil(t, info.Sys())
}

func TestWorkspaceFilesClientStatReturnsAPIErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		errorCode  string
	}{
		{"forbidden", 403, "PERMISSION_DENIED"},
		{"internal_error", 500, "INTERNAL_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testserver.New(t)

			server.Handle("GET", "/api/2.0/workspace/get-status", func(req testserver.Request) any {
				return testserver.Response{
					StatusCode: tt.statusCode,
					Body: map[string]string{
						"error_code": tt.errorCode,
						"message":    "test error",
					},
				}
			})

			testserver.AddDefaultHandlers(server)

			client, err := databricks.NewWorkspaceClient(&databricks.Config{
				Host:  server.URL,
				Token: "testtoken",
			})
			require.NoError(t, err)

			f, err := NewWorkspaceFilesClient(client, "/test")
			require.NoError(t, err)

			_, err = f.Stat(t.Context(), "file")
			require.Error(t, err)

			var apiErr *apierr.APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tt.statusCode, apiErr.StatusCode)
		})
	}
}

func TestWorkspaceFilesClientStatReturnsNotFoundAsFileDoesNotExist(t *testing.T) {
	server := testserver.New(t)

	server.Handle("GET", "/api/2.0/workspace/get-status", func(req testserver.Request) any {
		return testserver.Response{
			StatusCode: 404,
			Body: map[string]string{
				"error_code": "RESOURCE_DOES_NOT_EXIST",
				"message":    "not found",
			},
		}
	})

	testserver.AddDefaultHandlers(server)

	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)

	f, err := NewWorkspaceFilesClient(client, "/test")
	require.NoError(t, err)

	_, err = f.Stat(t.Context(), "file")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}
