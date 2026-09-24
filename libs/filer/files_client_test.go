package filer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	cliauth "github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	files "github.com/databricks/sdk-go/files/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const filesAPITestTimeout = time.Second

func TestNewFilesAPIClientDoesNotResolveAmbientConfig(t *testing.T) {
	server := testserver.New(t)
	server.Handle("HEAD", "/api/2.0/fs/files/{path...}", func(req testserver.Request) any {
		assert.Equal(t, "Bearer resolved-token", req.Headers.Get("Authorization"))
		assert.Empty(t, req.Headers.Get(cliauth.WorkspaceIDHeader))
		return ""
	})

	workspaceClient, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:        server.URL,
		Token:       "resolved-token",
		WorkspaceID: cliauth.WorkspaceIDNone,
	})
	require.NoError(t, err)

	configFile := filepath.Join(t.TempDir(), ".databrickscfg")
	require.NoError(t, os.WriteFile(configFile, []byte(`
[DEFAULT]
host = https://ambient.test
token = ambient-token
workspace_id = ambient-profile-workspace
`), 0o600))

	testCases := []struct {
		name        string
		configFile  string
		workspaceID string
	}{
		{name: "config file", configFile: configFile},
		{name: "environment", workspaceID: "ambient-env-workspace"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DATABRICKS_CONFIG_FILE", tc.configFile)
			t.Setenv("DATABRICKS_CONFIG_PROFILE", "")
			t.Setenv("DATABRICKS_WORKSPACE_ID", tc.workspaceID)

			client, err := newFilesAPIClient(t.Context(), workspaceClient.Config)
			require.NoError(t, err)

			filePath := "/Volumes/main/schema/volume/file"
			_, err = client.GetFileMetadata(t.Context(), files.GetFileMetadataRequest{FilePath: &filePath})
			require.NoError(t, err)
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestFilesAPIClient(t *testing.T, cfg *databricks.Config) *files.Client {
	t.Helper()

	workspaceClient, err := databricks.NewWorkspaceClient(cfg)
	require.NoError(t, err)
	client, err := newFilesAPIClient(t.Context(), workspaceClient.Config)
	require.NoError(t, err)
	return client
}

func TestNewFilesAPIClientTimesOutStalledResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
		<-req.Context().Done()
	}))
	defer server.Close()

	client := newTestFilesAPIClient(t, &databricks.Config{
		Host:                server.URL,
		Token:               "test-token",
		HTTPTimeoutSeconds:  int(filesAPITestTimeout.Seconds()),
		RetryTimeoutSeconds: 2,
	})
	filePath := "/Volumes/main/schema/volume/file"

	start := time.Now()
	_, err := client.GetFileMetadata(t.Context(), files.GetFileMetadataRequest{FilePath: &filePath})
	require.ErrorContains(t, err, "1s of inactivity")
	assert.Less(t, time.Since(start), 3*time.Second)
}

func TestNewFilesAPIClientExtendsTimeoutForProgressingResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		flusher := w.(http.Flusher)
		for _, chunk := range []string{`{`, `"contents":[]`, `}`} {
			_, err := w.Write([]byte(chunk))
			if err != nil {
				return
			}
			flusher.Flush()
			time.Sleep(500 * time.Millisecond)
		}
	}))
	defer server.Close()

	client := newTestFilesAPIClient(t, &databricks.Config{
		Host:                server.URL,
		Token:               "test-token",
		HTTPTimeoutSeconds:  int(filesAPITestTimeout.Seconds()),
		RetryTimeoutSeconds: 5,
	})
	directoryPath := "/Volumes/main/schema/volume"

	start := time.Now()
	_, err := client.ListDirectoryContents(t.Context(), files.ListDirectoryContentsRequest{DirectoryPath: &directoryPath})
	require.NoError(t, err)
	assert.Greater(t, time.Since(start), filesAPITestTimeout)
}

func TestNewFilesAPIClientUsesCustomTransport(t *testing.T) {
	var requests atomic.Int32
	transport := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/.well-known/databricks-config" {
			return &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found", Header: make(http.Header), Body: http.NoBody, Request: req}, nil
		}
		requests.Add(1)
		assert.Equal(t, http.MethodHead, req.Method)
		assert.Equal(t, "/api/2.0/fs/files/Volumes/main/schema/volume/file", req.URL.Path)
		assert.Equal(t, "Bearer test-token", req.Header.Get("Authorization"))
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Header: make(http.Header), Body: http.NoBody, Request: req}, nil
	})

	client := newTestFilesAPIClient(t, &databricks.Config{
		Host:          "https://workspace.test",
		Token:         "test-token",
		HTTPTransport: transport,
	})
	filePath := "/Volumes/main/schema/volume/file"
	_, err := client.GetFileMetadata(t.Context(), files.GetFileMetadataRequest{FilePath: &filePath})
	require.NoError(t, err)
	assert.Equal(t, int32(1), requests.Load())
}

func TestNewFilesClientAllowsDelayedFilesAPIResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		select {
		case <-time.After(500 * time.Millisecond):
			_, _ = io.WriteString(w, `{}`)
		case <-t.Context().Done():
		}
	}))
	defer server.Close()

	workspaceClient, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:                server.URL,
		Token:               "test-token",
		HTTPTimeoutSeconds:  int(filesAPITestTimeout.Seconds()),
		RetryTimeoutSeconds: 2,
	})
	require.NoError(t, err)
	client, err := NewFilesClient(t.Context(), workspaceClient, "/")
	require.NoError(t, err)

	entries, err := client.ReadDir(t.Context(), "/")
	require.NoError(t, err)
	assert.Empty(t, entries)
}

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

	f, err := NewFilesClient(t.Context(), client, "/test")
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

func newTestFilesClient(t *testing.T) Filer {
	t.Helper()

	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)

	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)

	f, err := NewFilesClient(t.Context(), client, "/")
	require.NoError(t, err)
	return f
}

func TestFilesClientMkdirWhenFileExists(t *testing.T) {
	// The Files API reports "a file already exists at this path" as a 409, which
	// the SDK maps to codes.Aborted (not codes.AlreadyExists); the filer keys off
	// the HTTP status so it still surfaces as fs.ErrExist.
	ctx := t.Context()
	f := newTestFilesClient(t)

	require.NoError(t, f.Mkdir(ctx, "/Volumes/main/schema/vol"))
	require.NoError(t, f.Write(ctx, "/Volumes/main/schema/vol/hello", bytes.NewReader([]byte("abc"))))

	err := f.Mkdir(ctx, "/Volumes/main/schema/vol/hello")
	assert.ErrorIs(t, err, fs.ErrExist)
}

// onlyReader hides the Seek method of an underlying reader, modelling a
// non-seekable stream (e.g. a remote download body).
type onlyReader struct{ io.Reader }

func TestIsSeekable(t *testing.T) {
	in := []byte("hello, files API")

	if !isSeekable(bytes.NewReader(in)) {
		t.Fatal("bytes.Reader should be seekable")
	}

	// The position must be left at the start so a subsequent read covers every byte.
	r := bytes.NewReader(in)
	if !isSeekable(r) {
		t.Fatal("bytes.Reader should be seekable")
	}
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, in) {
		t.Errorf("read %q after isSeekable, want %q (position not left at start)", b, in)
	}

	if isSeekable(onlyReader{bytes.NewReader(in)}) {
		t.Error("a non-seekable reader should report false")
	}
}

func TestMultipartUploadEnabled(t *testing.T) {
	ctx := t.Context()
	if MultipartUploadEnabled(ctx) {
		t.Error("multipart upload must be disabled by default")
	}
	for _, on := range []string{"true", "1", "yes", "on"} {
		if !MultipartUploadEnabled(env.Set(ctx, multipartUploadEnvVar, on)) {
			t.Errorf("value %q should enable multipart upload", on)
		}
	}
	for _, off := range []string{"false", "0", "", "nonsense"} {
		if MultipartUploadEnabled(env.Set(ctx, multipartUploadEnvVar, off)) {
			t.Errorf("value %q should not enable multipart upload", off)
		}
	}
}

func TestMapUploadError(t *testing.T) {
	const p = "/Volumes/c/s/v/f.bin"

	if err := mapUploadError(nil, p); err != nil {
		t.Errorf("nil error should pass through, got %v", err)
	}

	// The engine's already-exists sentinel (even wrapped) must surface as fs.ErrExist
	// so skip-if-exists keeps working.
	for _, in := range []error{
		files.ErrAlreadyExists,
		fmt.Errorf("upload failed: %w", files.ErrAlreadyExists),
	} {
		got := mapUploadError(in, p)
		if !errors.Is(got, fs.ErrExist) {
			t.Errorf("mapUploadError(%v) = %v, want errors.Is fs.ErrExist", in, got)
		}
	}

	// Other errors pass through unchanged.
	other := errors.New("boom")
	if got := mapUploadError(other, p); got != other {
		t.Errorf("mapUploadError(other) = %v, want it unchanged", got)
	}
}
