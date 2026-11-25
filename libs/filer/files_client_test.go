package filer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	cliauth "github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/sdk-go/core/apierr"
	"github.com/databricks/sdk-go/core/ops"
	files "github.com/databricks/sdk-go/files/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		WorkspaceID: profile.WorkspaceIDNone,
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
	if got := mapUploadError(other, p); got != other { //nolint:errorlint // must be the same error, not a wrapper
		t.Errorf("mapUploadError(other) = %v, want it unchanged", got)
	}
}

// throttledResponse is how the Files API reports throttled cloud storage.
var throttledResponse = testserver.Response{
	StatusCode: 429,
	Body: map[string]any{
		"error_code": "RESOURCE_EXHAUSTED",
		"message":    "AWS S3 is throttling requests; try again later.",
	},
}

// throttled wraps a handler so that its first n requests are throttled.
func throttled(n int, handler func(req testserver.Request) any) func(req testserver.Request) any {
	return failFirst(n, throttledResponse, handler)
}

// failFirst wraps a handler so that its first n requests get resp.
func failFirst(n int, resp testserver.Response, handler func(req testserver.Request) any) func(req testserver.Request) any {
	var mu sync.Mutex
	count := 0
	return func(req testserver.Request) any {
		mu.Lock()
		count++
		fail := count <= n
		mu.Unlock()
		if fail {
			return resp
		}
		return handler(req)
	}
}

func newFilesClientWithFastRetries(t *testing.T, server *testserver.Server) *FilesClient {
	t.Helper()

	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)

	f, err := NewFilesClient(t.Context(), client, "/Volumes/main/schema/vol")
	require.NoError(t, err)
	fc := f.(*FilesClient)
	fc.retryBackoff = ops.BackoffPolicy{Initial: time.Millisecond, Maximum: time.Millisecond}
	return fc
}

func ok(testserver.Request) any { return testserver.Response{} }

func TestFilesClientRetriesThrottledCalls(t *testing.T) {
	ctx := t.Context()
	server := testserver.New(t)
	var uploads []string
	server.Handle("HEAD", "/api/2.0/fs/directories/{path...}", throttled(2, ok))
	server.Handle("PUT", "/api/2.0/fs/files/{path...}", func(req testserver.Request) any {
		uploads = append(uploads, string(req.Body))
		return testserver.Response{}
	})
	server.Handle("GET", "/api/2.0/fs/files/{path...}", throttled(2, func(testserver.Request) any {
		return testserver.Response{Body: []byte("abc")}
	}))
	server.Handle("DELETE", "/api/2.0/fs/files/{path...}", throttled(2, ok))
	f := newFilesClientWithFastRetries(t, server)

	require.NoError(t, f.Write(ctx, "hello", strings.NewReader("abc")))

	r, err := f.Read(ctx, "hello")
	require.NoError(t, err)
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, "abc", string(data))
	require.NoError(t, r.Close())

	require.NoError(t, f.deleteFile(ctx, "hello"))
	assert.Equal(t, []string{"abc"}, uploads)
}

func TestFilesClientRetriesThrottledUploadFromStartOffset(t *testing.T) {
	ctx := t.Context()
	server := testserver.New(t)
	var uploads []string
	put := throttled(2, ok)
	server.Handle("PUT", "/api/2.0/fs/files/{path...}", func(req testserver.Request) any {
		uploads = append(uploads, string(req.Body))
		return put(req)
	})
	f := newFilesClientWithFastRetries(t, server)

	// Each attempt resends the body from where the reader was when Write was
	// called, not from where the failed attempt left it.
	reader := strings.NewReader("skip:abc")
	_, err := reader.Seek(5, io.SeekStart)
	require.NoError(t, err)
	require.NoError(t, f.Write(ctx, "hello", reader, CreateParentDirectories))
	assert.Equal(t, []string{"abc", "abc", "abc"}, uploads)
}

// uploadsWithOneThrottle records each PUT body; the first PUT is throttled.
func uploadsWithOneThrottle(server *testserver.Server) *[]string {
	var uploads []string
	put := throttled(1, ok)
	server.Handle("PUT", "/api/2.0/fs/files/{path...}", func(req testserver.Request) any {
		uploads = append(uploads, string(req.Body))
		return put(req)
	})
	return &uploads
}

func TestFilesClientRetriesSmallNonSeekableUpload(t *testing.T) {
	ctx := t.Context()
	server := testserver.New(t)
	uploads := uploadsWithOneThrottle(server)
	f := newFilesClientWithFastRetries(t, server)
	setReplayBufferSize(t, 3)

	// Exactly filesReplayBufferSize bytes still fit in the buffer.
	require.NoError(t, f.Write(ctx, "hello", onlyReader{strings.NewReader("abc")}, CreateParentDirectories))
	assert.Equal(t, []string{"abc", "abc"}, *uploads)
}

func TestFilesClientRetriesSmallPipeUpload(t *testing.T) {
	ctx := t.Context()
	server := testserver.New(t)
	uploads := uploadsWithOneThrottle(server)
	f := newFilesClientWithFastRetries(t, server)

	// A pipe is an *os.File whose Seek fails.
	pr, pw, err := os.Pipe()
	require.NoError(t, err)
	_, err = pw.WriteString("abc")
	require.NoError(t, err)
	require.NoError(t, pw.Close())
	defer pr.Close()

	require.NoError(t, f.Write(ctx, "hello", pr, CreateParentDirectories))
	assert.Equal(t, []string{"abc", "abc"}, *uploads)
}

func TestFilesClientDoesNotRetryLargeNonSeekableUpload(t *testing.T) {
	ctx := t.Context()
	server := testserver.New(t)
	uploads := uploadsWithOneThrottle(server)
	f := newFilesClientWithFastRetries(t, server)
	setReplayBufferSize(t, 2)

	err := f.Write(ctx, "hello", onlyReader{strings.NewReader("abc")}, CreateParentDirectories)
	assert.ErrorContains(t, err, "AWS S3 is throttling requests")
	assert.Equal(t, []string{"abc"}, *uploads)
}

// countingReader counts Read calls on a non-seekable reader, modelling a DBFS
// read handle that makes one API call per Read.
type countingReader struct {
	r     io.Reader
	calls int
}

func (c *countingReader) Read(p []byte) (int, error) {
	c.calls++
	return c.r.Read(p)
}

func TestFilesClientBuffersUploadInLargeReads(t *testing.T) {
	ctx := t.Context()
	server := testserver.New(t)
	uploads := uploadsWithOneThrottle(server)
	f := newFilesClientWithFastRetries(t, server)

	content := strings.Repeat("x", filesReplayReadSize)
	src := &countingReader{r: strings.NewReader(content)}
	require.NoError(t, f.Write(ctx, "hello", src, CreateParentDirectories))
	assert.Len(t, *uploads, 2)
	// One Read fills the buffer and one more returns EOF; io.ReadAll's own
	// growing chunks would take about 20.
	assert.LessOrEqual(t, src.calls, 3)
}

func setReplayBufferSize(t *testing.T, n int) {
	old := filesReplayBufferSize
	filesReplayBufferSize = n
	t.Cleanup(func() { filesReplayBufferSize = old })
}

func TestFilesClientStopsRetryingAfterLimit(t *testing.T) {
	ctx := t.Context()
	server := testserver.New(t)
	requests := 0
	del := throttled(filesMaxRetries+1, ok)
	server.Handle("DELETE", "/api/2.0/fs/files/{path...}", func(req testserver.Request) any {
		requests++
		return del(req)
	})
	f := newFilesClientWithFastRetries(t, server)

	err := f.deleteFile(ctx, "hello")
	assert.ErrorContains(t, err, "AWS S3 is throttling requests")
	assert.Equal(t, filesMaxRetries+1, requests)
}

// failing wraps a handler so that its first n requests fail with status and no
// body, as a failed HEAD (or a proxy error page) does.
func failing(n, status int, handler func(req testserver.Request) any) func(req testserver.Request) any {
	return failFirst(n, testserver.Response{StatusCode: status, Body: []byte{}}, handler)
}

func TestFilesClientRetriesReadOnlyCallsOn500(t *testing.T) {
	ctx := t.Context()
	server := testserver.New(t)
	server.Handle("HEAD", "/api/2.0/fs/directories/{path...}", failing(2, 500, ok))
	server.Handle("HEAD", "/api/2.0/fs/files/{path...}", failing(2, 500, ok))
	server.Handle("PUT", "/api/2.0/fs/files/{path...}", ok)
	server.Handle("GET", "/api/2.0/fs/files/{path...}", failing(2, 500, func(testserver.Request) any {
		return testserver.Response{Body: []byte("abc")}
	}))
	f := newFilesClientWithFastRetries(t, server)

	// Write checks that the parent directory exists with a HEAD.
	require.NoError(t, f.Write(ctx, "hello", strings.NewReader("abc")))

	_, err := f.statFile(ctx, "hello")
	require.NoError(t, err)

	r, err := f.Read(ctx, "hello")
	require.NoError(t, err)
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, "abc", string(data))
	require.NoError(t, r.Close())
}

func TestFilesClientDoesNotRetryDeleteOn500(t *testing.T) {
	ctx := t.Context()
	server := testserver.New(t)
	requests := 0
	server.Handle("DELETE", "/api/2.0/fs/files/{path...}", func(req testserver.Request) any {
		requests++
		return testserver.Response{StatusCode: 500, Body: []byte{}}
	})
	f := newFilesClientWithFastRetries(t, server)

	err := f.deleteFile(ctx, "hello")
	assert.EqualError(t, err, "delete /Volumes/main/schema/vol/hello: Files API returned HTTP 500 Internal Server Error")
	assert.Equal(t, 1, requests)
}

func TestFilesClientDescribesErrorWithoutMessage(t *testing.T) {
	ctx := t.Context()
	server := testserver.New(t)
	server.Handle("HEAD", "/api/2.0/fs/directories/{path...}", failing(filesMaxRetries+1, 403, ok))
	f := newFilesClientWithFastRetries(t, server)

	_, err := f.Stat(ctx, "dir")
	assert.EqualError(t, err, "stat /Volumes/main/schema/vol/dir: Files API returned HTTP 403 Forbidden")
	// The status stays available to callers that map errors by it.
	assert.Equal(t, http.StatusForbidden, httpStatus(err))
}

func TestDescribeFilesErrorLeavesOtherErrors(t *testing.T) {
	withMessage := apierr.FromHTTPError(500, nil, []byte(`{"error_code":"INTERNAL_ERROR","message":"boom"}`))
	assert.Equal(t, "databricks-api error: boom", describeFilesError(withMessage, "stat", "/p").Error())

	// A wrapped error already has context from its wrapper.
	wrapped := fmt.Errorf("context: %w", apierr.FromHTTPError(500, nil, nil))
	assert.Equal(t, wrapped, describeFilesError(wrapped, "stat", "/p"))

	assert.NoError(t, describeFilesError(nil, "stat", "/p"))
}

func TestFilesClientStopsRetrying500(t *testing.T) {
	ctx := t.Context()
	server := testserver.New(t)
	requests := 0
	head := failing(filesMaxServerErrorRetries+1, 500, ok)
	server.Handle("HEAD", "/api/2.0/fs/files/{path...}", func(req testserver.Request) any {
		requests++
		return head(req)
	})
	f := newFilesClientWithFastRetries(t, server)

	_, err := f.statFile(ctx, "hello")
	assert.EqualError(t, err, "stat /Volumes/main/schema/vol/hello: Files API returned HTTP 500 Internal Server Error")
	assert.Equal(t, filesMaxServerErrorRetries+1, requests)
}

func TestFilesAPIErrorText(t *testing.T) {
	errorCodeOnly := apierr.FromHTTPError(403, nil, []byte(`{"error_code":"PERMISSION_DENIED"}`))
	assert.EqualError(t, describeFilesError(errorCodeOnly, "read", "/p"), "read /p: Files API returned HTTP 403 Forbidden (PERMISSION_DENIED)")

	nonStandard := apierr.FromHTTPError(520, nil, []byte("<html>oops</html>"))
	assert.EqualError(t, describeFilesError(nonStandard, "read", "/p"), "read /p: Files API returned HTTP 520")
}

// failThenNotFound answers the first request with status and every later one
// with 404, as when a delete is retried after its first attempt was or was not
// applied.
func failThenNotFound(status int) testserver.HandlerFunc {
	calls := 0
	return func(testserver.Request) any {
		calls++
		if calls == 1 {
			return testserver.Response{StatusCode: status, Body: map[string]string{"error_code": "INJECTED", "message": "injected"}}
		}
		return testserver.Response{StatusCode: http.StatusNotFound, Body: map[string]string{"error_code": "NOT_FOUND", "message": "not found"}}
	}
}

func TestFilesClientDeleteNotFoundAfterRetry(t *testing.T) {
	for _, tc := range []struct {
		status  int
		deleted bool
	}{
		// The first attempt may have been applied, so the 404 means it deleted the path.
		{http.StatusGatewayTimeout, true},
		{http.StatusBadGateway, true},
		{http.StatusServiceUnavailable, true},
		{http.StatusRequestTimeout, true},
		// A throttled attempt was not applied, so the 404 means the path is missing.
		{http.StatusTooManyRequests, false},
	} {
		t.Run(strconv.Itoa(tc.status), func(t *testing.T) {
			server := testserver.New(t)
			server.Handle("DELETE", "/api/2.0/fs/files/{path...}", failThenNotFound(tc.status))
			server.Handle("DELETE", "/api/2.0/fs/directories/{path...}", failThenNotFound(tc.status))
			f := newFilesClientWithFastRetries(t, server)

			fileErr := f.deleteFile(t.Context(), "file")
			dirErr := f.deleteDirectory(t.Context(), "dir")
			if tc.deleted {
				assert.NoError(t, fileErr)
				assert.NoError(t, dirErr)
			} else {
				assert.ErrorIs(t, fileErr, fs.ErrNotExist)
				assert.ErrorIs(t, dirErr, fs.ErrNotExist)
			}
		})
	}
}
