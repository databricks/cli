package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
)

// stubFiler is a filer.Filer whose Stat and Write return scripted results per call,
// so tests can drive the retry loops in binaryExists/uploadRelease deterministically.
type stubFiler struct {
	filer.Filer
	statErrs  []error
	writeErrs []error
	statCalls int
	writes    int
}

func (s *stubFiler) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	err := s.statErrs[s.statCalls]
	s.statCalls++
	return nil, err
}

func (s *stubFiler) Write(ctx context.Context, path string, reader io.Reader, mode ...filer.WriteMode) error {
	// Drain the reader as the real filer would, so a retry must supply a fresh one.
	_, _ = io.Copy(io.Discard, reader)
	err := s.writeErrs[s.writes]
	s.writes++
	return err
}

func fakeRelease(ctx context.Context, arch, version, releasesDir string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("archive")), nil
}

func timeoutErr() error {
	return errors.New(`Get "https://example.test/api/2.0/workspace/get-status": request timed out after 1m0s of inactivity`)
}

func TestIsStreamResetError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "typed http2.StreamError wrapped",
			err:  fmt.Errorf(`Post "https://example/api/2.0/workspace-files/import-file/...": %w`, http2.StreamError{StreamID: 15, Code: http2.ErrCodeNo}),
			want: true,
		},
		{
			name: "stringified stream error",
			err:  errors.New("stream error: stream ID 15; NO_ERROR; received from peer"),
			want: true,
		},
		{
			name: "unrelated error",
			err:  errors.New("connection refused"),
			want: false,
		},
		{
			name: "API error message",
			err:  errors.New("RESOURCE_DOES_NOT_EXIST: path does not exist"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isStreamResetError(tt.err))
		})
	}
}

func TestIsRetriableUploadError(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"inactivity timeout string", timeoutErr(), true},
		{"context deadline", context.DeadlineExceeded, true},
		{"retriable API 503", &apierr.APIError{StatusCode: http.StatusServiceUnavailable}, true},
		{"retriable API 429", &apierr.APIError{StatusCode: http.StatusTooManyRequests}, true},
		{"non-retriable API 404", &apierr.APIError{StatusCode: http.StatusNotFound}, false},
		{"plain error", errors.New("connection refused"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isRetriableUploadError(ctx, tt.err))
		})
	}
}

func TestBinaryExistsRetriesTransientStat(t *testing.T) {
	ctx := context.Background()

	// A transient timeout on the first stat, then a clean "exists" on the retry.
	f := &stubFiler{statErrs: []error{timeoutErr(), nil}}
	exists, err := binaryExists(ctx, f, "amd64/databricks")
	require.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, 2, f.statCalls)
}

func TestBinaryExistsNotFoundNoRetry(t *testing.T) {
	ctx := context.Background()

	// A definitive "not found" must resolve immediately (it's the signal to upload), not retry.
	f := &stubFiler{statErrs: []error{fs.ErrNotExist}}
	exists, err := binaryExists(ctx, f, "amd64/databricks")
	require.NoError(t, err)
	assert.False(t, exists)
	assert.Equal(t, 1, f.statCalls)
}

func TestBinaryExistsHaltsOnNonRetriable(t *testing.T) {
	ctx := context.Background()

	f := &stubFiler{statErrs: []error{&apierr.APIError{StatusCode: http.StatusForbidden, Message: "denied"}}}
	_, err := binaryExists(ctx, f, "amd64/databricks")
	require.Error(t, err)
	assert.Equal(t, 1, f.statCalls)
}

func TestUploadReleaseRetriesTransientWrite(t *testing.T) {
	ctx := context.Background()

	// First write fails transiently; the retry must re-fetch a fresh reader and succeed.
	f := &stubFiler{writeErrs: []error{&apierr.APIError{StatusCode: http.StatusServiceUnavailable}, nil}}
	err := uploadRelease(ctx, f, fakeRelease, "amd64", "1.0.0", "", "amd64/databricks.zip")
	require.NoError(t, err)
	assert.Equal(t, 2, f.writes)
}

func TestUploadReleaseStreamResetNoRetry(t *testing.T) {
	ctx := context.Background()

	// A stream reset is a proxy body-size rejection: fail fast with the hint, don't retry.
	f := &stubFiler{writeErrs: []error{http2.StreamError{StreamID: 1, Code: http2.ErrCodeNo}}}
	err := uploadRelease(ctx, f, fakeRelease, "amd64", "1.0.0", "", "amd64/databricks.zip")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "request-body size limit")
	assert.Equal(t, 1, f.writes)
}
