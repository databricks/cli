package filer

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Confirms directory entries derived from workspace object infos report the
// name, type, and size of each object type the workspace can return.
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

// Confirms the workspace ID routing header is set only when the client config
// carries a workspace ID, and that a nil workspace client yields no headers.
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

// Confirms NewWorkspaceFilesClient clears the CLI-only "none" sentinel so the
// SDK's Workspace.Upload/Download never route on it, while a real workspace ID
// is left untouched.
func TestNewWorkspaceFilesClientNormalizesWorkspaceID(t *testing.T) {
	server := testserver.New(t)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"none sentinel is cleared", auth.WorkspaceIDNone, ""},
		{"real workspace ID is kept", "7474644166319138", "7474644166319138"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, err := databricks.NewWorkspaceClient(&databricks.Config{
				Host:        server.URL,
				Token:       "testtoken",
				WorkspaceID: tc.input,
			})
			require.NoError(t, err)

			_, err = NewWorkspaceFilesClient(client, "/dir")
			require.NoError(t, err)
			assert.Equal(t, tc.want, client.Config.WorkspaceID)
		})
	}
}

// importFormFields parses the multipart body of a recorded /workspace/import
// request into a field name -> value map.
func importFormFields(t *testing.T, contentType string, body []byte) map[string]string {
	t.Helper()

	_, params, err := mime.ParseMediaType(contentType)
	require.NoError(t, err)
	mr := multipart.NewReader(bytes.NewReader(body), params["boundary"])

	fields := map[string]string{}
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		data, err := io.ReadAll(part)
		require.NoError(t, err)
		fields[part.FormName()] = string(data)
	}
	return fields
}

// Confirms Write posts a multipart /workspace/import body with format=AUTO, and
// that the overwrite field is present only when OverwriteIfExists is passed.
// AUTO is what lets the server decide between storing content as a file or a
// notebook, so the format is asserted explicitly rather than left to the
// endpoint default (SOURCE), which would import every file as a notebook.
func TestWorkspaceFilesClientWriteSuccess(t *testing.T) {
	tests := []struct {
		name          string
		modes         []WriteMode
		expectPresent bool
	}{
		{
			name:          "no overwrite",
			modes:         nil,
			expectPresent: false,
		},
		{
			name:          "overwrite",
			modes:         []WriteMode{OverwriteIfExists},
			expectPresent: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotForm map[string]string
			server := testserver.New(t)
			server.Handle("POST", "/api/2.0/workspace/import", func(req testserver.Request) any {
				gotForm = importFormFields(t, req.Headers.Get("Content-Type"), req.Body)
				return testserver.Response{StatusCode: http.StatusOK}
			})
			testserver.AddDefaultHandlers(server)

			client, err := databricks.NewWorkspaceClient(&databricks.Config{
				Host:  server.URL,
				Token: "testtoken",
			})
			require.NoError(t, err)

			f, err := NewWorkspaceFilesClient(client, "/dir")
			require.NoError(t, err)

			require.NoError(t, f.Write(t.Context(), "file.txt", strings.NewReader("hello"), tc.modes...))

			assert.Equal(t, "/dir/file.txt", gotForm["path"])
			assert.Equal(t, "hello", gotForm["content"])
			assert.Equal(t, string(workspace.ImportFormatAuto), gotForm["format"])
			overwrite, present := gotForm["overwrite"]
			assert.Equal(t, tc.expectPresent, present)
			if present {
				assert.Equal(t, "true", overwrite)
			}
		})
	}
}

// Confirms the workspace routing header is sent when the config carries a real
// workspace ID, and that the CLI-only "none" sentinel is never forwarded as a
// literal routing identifier. The sentinel is written to .databrickscfg by
// `auth login --skip-workspace`; the platform has no workspace named "none",
// so sending it would misroute the upload.
func TestWorkspaceFilesClientWriteWorkspaceIDHeader(t *testing.T) {
	tests := []struct {
		name        string
		workspaceID string
		expect      string
	}{
		{
			name:        "real workspace ID is forwarded",
			workspaceID: "7474644166319138",
			expect:      "7474644166319138",
		},
		{
			name:        "none sentinel is not forwarded",
			workspaceID: auth.WorkspaceIDNone,
			expect:      "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotHeader string
			server := testserver.New(t)
			server.Handle("POST", "/api/2.0/workspace/import", func(req testserver.Request) any {
				gotHeader = req.Headers.Get(auth.WorkspaceIDHeader)
				return testserver.Response{StatusCode: http.StatusOK}
			})
			testserver.AddDefaultHandlers(server)

			client, err := databricks.NewWorkspaceClient(&databricks.Config{
				Host:        server.URL,
				Token:       "testtoken",
				WorkspaceID: tc.workspaceID,
			})
			require.NoError(t, err)

			f, err := NewWorkspaceFilesClient(client, "/dir")
			require.NoError(t, err)

			require.NoError(t, f.Write(t.Context(), "file.txt", strings.NewReader("hello")))
			assert.Equal(t, tc.expect, gotHeader)
		})
	}
}

// Confirms each error /workspace/import returns is translated to the filer's
// own error type, and that unrecognized errors pass through unchanged. The
// endpoint signals an occupied path with three different status/error_code
// combinations (see the comment on Write), so all three are covered here to
// pin the mapping the rest of the CLI relies on.
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
			// A 409 carrying no error_code unwraps to ErrResourceConflict only,
			// not to ErrAlreadyExists; the testserver returns exactly this shape.
			name: "409 without error_code maps to fileAlreadyExistsError",
			apiErr: &apierr.APIError{
				StatusCode: http.StatusConflict,
				Message:    "Node with name /dir/file.txt already exists.",
			},
			expectErrTarget: fileAlreadyExistsError{},
		},
		{
			// Notebook conflicts arrive as a 400 with an empty error_code, so
			// they match neither ErrResourceAlreadyExists nor
			// ErrInvalidParameterValue and are detected by message marker.
			name: "400 without error_code and 'already exists.' maps to fileAlreadyExistsError",
			apiErr: &apierr.APIError{
				StatusCode: http.StatusBadRequest,
				Message:    "Path (/dir/notebook) already exists.",
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
			server := testserver.New(t)
			server.Handle("POST", "/api/2.0/workspace/import", func(req testserver.Request) any {
				return testserver.Response{
					StatusCode: tc.apiErr.StatusCode,
					Body: map[string]string{
						"error_code": tc.apiErr.ErrorCode,
						"message":    tc.apiErr.Message,
					},
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

			err = f.Write(t.Context(), "file.txt", bytes.NewReader([]byte("data")), tc.mode...)
			require.Error(t, err)
			switch target := tc.expectErrTarget.(type) {
			case noSuchDirectoryError:
				assert.ErrorAs(t, err, &target)
			case fileAlreadyExistsError:
				assert.ErrorAs(t, err, &target)
			case permissionError:
				assert.ErrorAs(t, err, &target)
			case nil:
				// passthrough — the underlying APIError stays inspectable,
				// and the message names the file that failed to upload.
				var aerr *apierr.APIError
				require.ErrorAs(t, err, &aerr)
				assert.Equal(t, tc.apiErr.StatusCode, aerr.StatusCode)
				assert.Contains(t, err.Error(), "/dir/file.txt")
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

// Confirms a type-mismatch collision is recognized from the structured
// AIP-193 ErrorInfo reason, independent of the error message wording.
func TestWorkspaceFilesClientWriteTypeMismatchReason(t *testing.T) {
	// The message is deliberately one the fallback string match does not
	// recognize, to prove the branch fires on the structured reason alone.
	err := writeWithImportError(t, map[string]any{
		"error_code": "INVALID_PARAMETER_VALUE",
		"message":    "some future wording for the same condition",
		"details": []map[string]any{
			{
				"@type":    "type.googleapis.com/google.rpc.ErrorInfo",
				"reason":   "WORKSPACE_OBJECT_TYPE_MISMATCH",
				"domain":   "workspace.databricks.com",
				"metadata": map[string]string{"existing_type": "NOTEBOOK"},
			},
		},
	})
	var target fileAlreadyExistsError
	assert.ErrorAs(t, err, &target)
}

// Confirms an ErrorInfo carrying some other reason is not mistaken for a path
// collision, so the reason check above cannot swallow unrelated failures.
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

// Confirms a 404 from a missing parent directory triggers a mkdirs call and a
// retry of the upload when CreateParentDirectories is passed. The retry re-reads
// the buffered body, so this also covers that the content survives the second
// attempt.
func TestWorkspaceFilesClientWriteCreatesParentDirectories(t *testing.T) {
	var uploads int
	var mkdirs []string
	var lastContent string

	server := testserver.New(t)
	server.Handle("POST", "/api/2.0/workspace/import", func(req testserver.Request) any {
		uploads++
		// The first attempt 404s as it would against a missing parent; the retry
		// after mkdirs succeeds.
		if uploads == 1 {
			return testserver.Response{
				StatusCode: http.StatusNotFound,
				Body:       map[string]string{"message": "The parent folder (/dir/sub) does not exist."},
			}
		}
		form := importFormFields(t, req.Headers.Get("Content-Type"), req.Body)
		lastContent = form["content"]
		return testserver.Response{StatusCode: http.StatusOK}
	})
	server.Handle("POST", "/api/2.0/workspace/mkdirs", func(req testserver.Request) any {
		var request workspace.Mkdirs
		require.NoError(t, json.Unmarshal(req.Body, &request))
		mkdirs = append(mkdirs, request.Path)
		return testserver.Response{StatusCode: http.StatusOK}
	})
	testserver.AddDefaultHandlers(server)

	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)

	f, err := NewWorkspaceFilesClient(client, "/dir")
	require.NoError(t, err)

	require.NoError(t, f.Write(t.Context(), "sub/file.txt", strings.NewReader("data"), CreateParentDirectories))

	assert.Equal(t, 2, uploads)
	assert.Equal(t, []string{"/dir/sub"}, mkdirs)
	// The retry re-reads the buffered body, so the content must survive.
	assert.Equal(t, "data", lastContent)
}

// Confirms a workspace get-status payload unmarshals into wsfsFileInfo and that
// the fs.FileInfo methods derived from it report the expected values.
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

// Confirms a 403 from get-status surfaces as an APIError rather than being
// remapped, so callers can distinguish it from a missing file.
func TestWorkspaceFilesClientStatForbidden(t *testing.T) {
	err := statWithError(t, 403, "PERMISSION_DENIED")
	var apiErr *apierr.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 403, apiErr.StatusCode)
}

// Confirms a 500 from get-status surfaces as an APIError and is not treated as
// a missing file.
func TestWorkspaceFilesClientStatInternalError(t *testing.T) {
	err := statWithError(t, 500, "INTERNAL_ERROR")
	var apiErr *apierr.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, 500, apiErr.StatusCode)
}

// Confirms RESOURCE_DOES_NOT_EXIST maps to fs.ErrNotExist so the filer plugs
// into the io/fs error conventions callers check against.
func TestWorkspaceFilesClientStatNotFound(t *testing.T) {
	err := statWithError(t, 404, "RESOURCE_DOES_NOT_EXIST")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}
