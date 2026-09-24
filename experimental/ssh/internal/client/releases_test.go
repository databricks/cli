package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"syscall"
	"testing"
	"testing/synctest"
	"time"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
)

func TestIsProxyUploadError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "413 request entity too large",
			err:  &apierr.APIError{StatusCode: http.StatusRequestEntityTooLarge, Message: "request too large"},
			want: true,
		},
		{
			name: "connection reset mid-body",
			err:  fmt.Errorf(`Post "https://example/...": write tcp: %w`, syscall.ECONNRESET),
			want: true,
		},
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
			name: "non-413 API error",
			err:  &apierr.APIError{StatusCode: http.StatusForbidden, Message: "permission denied"},
			want: false,
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
			assert.Equal(t, tt.want, isProxyUploadError(tt.err))
		})
	}
}

func TestNewHTTP11TransportDisablesHTTP2(t *testing.T) {
	tr := newHTTP11Transport(&config.Config{})
	assert.False(t, tr.ForceAttemptHTTP2)
	assert.NotNil(t, tr.TLSNextProto)
	assert.Empty(t, tr.TLSNextProto)
}

func TestNewHTTP11WorkspaceClient(t *testing.T) {
	src := &config.Config{
		Host:               "https://test.databricks.test",
		Token:              "dapi-test",
		InsecureSkipVerify: true,
	}

	w, err := newHTTP11WorkspaceClient(src)
	require.NoError(t, err)

	// Auth-relevant attributes are carried over to the reconstructed config.
	assert.Equal(t, src.Host, w.Config.Host)
	assert.Equal(t, src.Token, w.Config.Token)
	assert.True(t, w.Config.InsecureSkipVerify)

	// The reconstructed client forces HTTP/1.1 via its transport.
	tr, ok := w.Config.HTTPTransport.(*http.Transport)
	require.True(t, ok)
	assert.False(t, tr.ForceAttemptHTTP2)
	assert.Empty(t, tr.TLSNextProto)

	// The source config is not mutated: it keeps its own (nil) transport.
	assert.Nil(t, src.HTTPTransport)
}

type releaseRoundTripFunc func(*http.Request) (*http.Response, error)

func (f releaseRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (r *trackingReadCloser) Close() error {
	r.closed = true
	return nil
}

func TestReleaseDownloadHTTPClientTimeout(t *testing.T) {
	assert.Equal(t, 10*time.Minute, releaseDownloadHTTPClient.Timeout)
}

func TestGetGithubReleaseWithClientRequest(t *testing.T) {
	body := &trackingReadCloser{Reader: strings.NewReader("release")}
	client := &http.Client{Transport: releaseRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, http.MethodGet, req.Method)
		assert.Equal(t, "https://github.com/databricks/cli/releases/download/v1.2.3/databricks_cli_1.2.3_linux_arm64.zip", req.URL.String())
		assert.Equal(t, t.Context(), req.Context())
		return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
	})}

	release, err := getGithubReleaseWithClient(t.Context(), "arm64", "1.2.3", "", client)
	require.NoError(t, err)
	assert.Same(t, body, release)
}

func TestGetGithubReleaseWithClientClosesNonOKBody(t *testing.T) {
	body := &trackingReadCloser{Reader: strings.NewReader("not found")}
	client := &http.Client{Transport: releaseRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusNotFound, Body: body}, nil
	})}

	release, err := getGithubReleaseWithClient(t.Context(), "amd64", "1.2.3", "", client)
	require.Error(t, err)
	assert.Nil(t, release)
	assert.True(t, body.closed)
}

func TestGetGithubReleaseWithClientCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		entered := make(chan struct{})
		client := &http.Client{Transport: releaseRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			close(entered)
			<-req.Context().Done()
			return nil, req.Context().Err()
		})}
		errCh := make(chan error, 1)
		go func() {
			_, err := getGithubReleaseWithClient(ctx, "amd64", "1.2.3", "", client)
			errCh <- err
		}()

		<-entered
		cancel()
		err := <-errCh
		require.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
	})
}
