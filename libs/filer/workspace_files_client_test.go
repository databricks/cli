package filer

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func TestWorkspaceFilesClientWorkspaceIDHeaders(t *testing.T) {
	tests := []struct {
		name        string
		workspaceID string
		expect      map[string]string
	}{
		{
			name:        "with workspace ID",
			workspaceID: "7474644166319138",
			expect:      map[string]string{"X-Databricks-Workspace-Id": "7474644166319138"},
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
			assert.Equal(t, tc.expect, w.workspaceIDHeaders())
		})
	}

	t.Run("nil workspace client", func(t *testing.T) {
		w := &WorkspaceFilesClient{}
		assert.Nil(t, w.workspaceIDHeaders())
	})
}

func TestWorkspaceFilesClientWriteSuccess(t *testing.T) {
	tests := []struct {
		name           string
		modes          []WriteMode
		expectOverride bool
	}{
		{
			name:           "no overwrite",
			modes:          nil,
			expectOverride: false,
		},
		{
			name:           "overwrite",
			modes:          []WriteMode{OverwriteIfExists},
			expectOverride: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mw := mocks.NewMockWorkspaceClient(t)
			workspaceApi := mw.GetMockWorkspaceAPI()

			workspaceApi.EXPECT().Upload(
				mock.Anything,
				"/dir/file.txt",
				mock.Anything,
				mock.Anything,
				mock.Anything,
			).RunAndReturn(func(_ context.Context, _ string, r io.Reader, opts ...func(*workspace.Import)) error {
				body, err := io.ReadAll(r)
				require.NoError(t, err)
				assert.Equal(t, "hello", string(body))

				i := &workspace.Import{}
				for _, opt := range opts {
					opt(i)
				}
				assert.Equal(t, workspace.ImportFormatAuto, i.Format)
				assert.Equal(t, tc.expectOverride, i.Overwrite)
				return nil
			}).Once()

			c := WorkspaceFilesClient{
				workspaceClient: mw.WorkspaceClient,
				root:            NewWorkspaceRootPath("/dir"),
			}
			err := c.Write(t.Context(), "file.txt", strings.NewReader("hello"), tc.modes...)
			require.NoError(t, err)
		})
	}
}

func TestWorkspaceFilesClientWriteErrorMapping(t *testing.T) {
	tests := []struct {
		name            string
		mode            []WriteMode
		apiErr          *apierr.APIError
		expectErrTarget any
	}{
		{
			name:            "404 without create-parent maps to noSuchDirectoryError",
			apiErr:          &apierr.APIError{StatusCode: http.StatusNotFound, Message: "not found"},
			expectErrTarget: noSuchDirectoryError{},
		},
		{
			name: "400 RESOURCE_ALREADY_EXISTS maps to fileAlreadyExistsError",
			apiErr: &apierr.APIError{
				StatusCode: http.StatusBadRequest,
				ErrorCode:  "RESOURCE_ALREADY_EXISTS",
				Message:    "/dir/file.txt already exists. Please pass overwrite=true to overwrite it.",
			},
			expectErrTarget: fileAlreadyExistsError{},
		},
		{
			name: "409 ALREADY_EXISTS (concurrent contention) maps to fileAlreadyExistsError",
			apiErr: &apierr.APIError{
				StatusCode: http.StatusConflict,
				ErrorCode:  "ALREADY_EXISTS",
				Message:    "Node with name /dir/file.txt already exists. Please pass overwrite=true to update it.",
			},
			expectErrTarget: fileAlreadyExistsError{},
		},
		{
			name: "400 INVALID_PARAMETER_VALUE 'type mismatch' (overwrite=true) maps to fileAlreadyExistsError",
			apiErr: &apierr.APIError{
				StatusCode: http.StatusBadRequest,
				ErrorCode:  "INVALID_PARAMETER_VALUE",
				Message:    "Cannot overwrite the asset at /dir/foo due to type mismatch (asked: FILE, actual: NOTEBOOK).",
			},
			expectErrTarget: fileAlreadyExistsError{},
		},
		{
			name: "400 INVALID_PARAMETER_VALUE 'Requested node type' (overwrite=true) maps to fileAlreadyExistsError",
			apiErr: &apierr.APIError{
				StatusCode: http.StatusBadRequest,
				ErrorCode:  "INVALID_PARAMETER_VALUE",
				Message:    "Requested node type [FILE] is different from the existing node type [NOTEBOOK]",
			},
			expectErrTarget: fileAlreadyExistsError{},
		},
		{
			name: "400 INVALID_PARAMETER_VALUE other message passes through",
			apiErr: &apierr.APIError{
				StatusCode: http.StatusBadRequest,
				ErrorCode:  "INVALID_PARAMETER_VALUE",
				Message:    "some other validation failure",
			},
			expectErrTarget: nil,
		},
		{
			name:            "403 maps to permissionError",
			apiErr:          &apierr.APIError{StatusCode: http.StatusForbidden, Message: "denied"},
			expectErrTarget: permissionError{},
		},
		{
			name:            "500 passes through",
			apiErr:          &apierr.APIError{StatusCode: http.StatusInternalServerError, Message: "boom"},
			expectErrTarget: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mw := mocks.NewMockWorkspaceClient(t)
			workspaceApi := mw.GetMockWorkspaceAPI()
			workspaceApi.EXPECT().Upload(
				mock.Anything, "/dir/file.txt", mock.Anything, mock.Anything, mock.Anything,
			).Return(tc.apiErr).Once()

			c := WorkspaceFilesClient{
				workspaceClient: mw.WorkspaceClient,
				root:            NewWorkspaceRootPath("/dir"),
			}
			err := c.Write(t.Context(), "file.txt", bytes.NewReader([]byte("data")), tc.mode...)
			require.Error(t, err)
			switch target := tc.expectErrTarget.(type) {
			case noSuchDirectoryError:
				assert.ErrorAs(t, err, &target)
			case fileAlreadyExistsError:
				assert.ErrorAs(t, err, &target)
			case permissionError:
				assert.ErrorAs(t, err, &target)
			case nil:
				// passthrough — same APIError pointer
				var aerr *apierr.APIError
				require.ErrorAs(t, err, &aerr)
				assert.Equal(t, tc.apiErr.StatusCode, aerr.StatusCode)
			}
		})
	}
}

// writeWithImportError exercises Write through a real HTTP roundtrip so the
// SDK parses AIP-193 error details from the response body (the errorDetails
// field on apierr.APIError is unexported and only populated during response
// parsing, so it cannot be set on a directly constructed APIError).
func writeWithImportError(t *testing.T, body map[string]any) error {
	t.Helper()

	server := testserver.New(t)
	server.Handle("POST", "/api/2.0/workspace/import", func(req testserver.Request) any {
		return testserver.Response{
			StatusCode: http.StatusBadRequest,
			Body:       body,
		}
	})
	testserver.AddDefaultHandlers(server)

	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)

	f, err := NewWorkspaceFilesClient(client, "/dir")
	require.NoError(t, err)

	err = f.Write(t.Context(), "file.txt", strings.NewReader("data"), OverwriteIfExists)
	require.Error(t, err)
	return err
}

func TestWorkspaceFilesClientWriteTypeMismatchReason(t *testing.T) {
	// The message is deliberately one the fallback string match does not
	// recognize, to prove the branch fires on the structured reason alone.
	err := writeWithImportError(t, map[string]any{
		"error_code": "INVALID_PARAMETER_VALUE",
		"message":    "some future wording for the same condition",
		"details": []map[string]any{
			{
				"@type":    "type.googleapis.com/google.rpc.ErrorInfo",
				"reason":   workspaceObjectTypeMismatchReason,
				"domain":   "workspace.databricks.com",
				"metadata": map[string]string{"existing_type": "NOTEBOOK"},
			},
		},
	})
	var target fileAlreadyExistsError
	assert.ErrorAs(t, err, &target)
}

func TestWorkspaceFilesClientWriteUnrelatedReasonPassesThrough(t *testing.T) {
	err := writeWithImportError(t, map[string]any{
		"error_code": "INVALID_PARAMETER_VALUE",
		"message":    "some other validation failure",
		"details": []map[string]any{
			{
				"@type":  "type.googleapis.com/google.rpc.ErrorInfo",
				"reason": "SOME_OTHER_REASON",
				"domain": "workspace.databricks.com",
			},
		},
	})
	var aerr *apierr.APIError
	require.ErrorAs(t, err, &aerr)
	assert.Equal(t, http.StatusBadRequest, aerr.StatusCode)
}

func TestWorkspaceFilesClientWriteCreatesParentDirectories(t *testing.T) {
	mw := mocks.NewMockWorkspaceClient(t)
	workspaceApi := mw.GetMockWorkspaceAPI()

	// First Upload returns 404, second returns success after MkdirsByPath.
	workspaceApi.EXPECT().Upload(
		mock.Anything, "/dir/sub/file.txt", mock.Anything, mock.Anything, mock.Anything,
	).Return(&apierr.APIError{StatusCode: http.StatusNotFound, Message: "not found"}).Once()

	workspaceApi.EXPECT().MkdirsByPath(mock.Anything, "/dir/sub").Return(nil).Once()

	workspaceApi.EXPECT().Upload(
		mock.Anything, "/dir/sub/file.txt", mock.Anything, mock.Anything, mock.Anything,
	).Return(nil).Once()

	c := WorkspaceFilesClient{
		workspaceClient: mw.WorkspaceClient,
		root:            NewWorkspaceRootPath("/dir"),
	}
	err := c.Write(t.Context(), "sub/file.txt", strings.NewReader("data"), CreateParentDirectories)
	require.NoError(t, err)
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

func statWithError(t *testing.T, statusCode int, errorCode string) error {
	t.Helper()

	server := testserver.New(t)
	server.Handle("GET", "/api/2.0/workspace/get-status", func(req testserver.Request) any {
		return testserver.Response{
			StatusCode: statusCode,
			Body: map[string]string{
				"error_code": errorCode,
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
	return err
}

func TestWorkspaceFilesClientStatForbidden(t *testing.T) {
	err := statWithError(t, 403, "PERMISSION_DENIED")
	var apiErr *apierr.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 403, apiErr.StatusCode)
}

func TestWorkspaceFilesClientStatInternalError(t *testing.T) {
	err := statWithError(t, 500, "INTERNAL_ERROR")
	var apiErr *apierr.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 500, apiErr.StatusCode)
}

func TestWorkspaceFilesClientStatNotFound(t *testing.T) {
	err := statWithError(t, 404, "RESOURCE_DOES_NOT_EXIST")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}
