package client

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"sync"
	"syscall"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
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

func TestUploadReleasesWithExistingBinary(t *testing.T) {
	tests := []struct {
		name         string
		version      string
		wantUploaded []string
	}{
		{
			name:         "release version skips the upload",
			version:      "1.12.0",
			wantUploaded: nil,
		},
		{
			name:         "dev version overwrites the binary",
			version:      "1.12.1-dev+abcdef123456",
			wantUploaded: []string{"amd64", "arm64"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			workspaceFiler, err := filer.NewLocalClient(t.TempDir())
			require.NoError(t, err)
			for _, arch := range []string{"amd64", "arm64"} {
				remoteBinaryPath := strings.TrimSuffix(getReleaseName(arch, tt.version), ".zip") + "/databricks"
				err := workspaceFiler.Write(ctx, remoteBinaryPath, strings.NewReader("old"), filer.CreateParentDirectories)
				require.NoError(t, err)
			}

			var uploaded []string
			getRelease := func(ctx context.Context, architecture, version, releasesDir string) (io.ReadCloser, error) {
				uploaded = append(uploaded, architecture)
				return io.NopCloser(strings.NewReader("new")), nil
			}

			err = uploadReleases(ctx, workspaceFiler, getRelease, tt.version, "")
			require.NoError(t, err)
			assert.Equal(t, tt.wantUploaded, uploaded)
		})
	}
}

// releaseArchive returns a zip holding the databricks binary. The import-file
// endpoint unzips uploaded archives, so it has to be valid.
func releaseArchive(t *testing.T) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	_, err := zw.Create("databricks")
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func TestUploadReleasesSendsEachArchiveOnce(t *testing.T) {
	tests := []struct {
		name    string
		version string
		uploads int
		want    []string
	}{
		{
			name:    "release version into new folders",
			version: "1.12.0",
			uploads: 1,
			want: []string{
				"databricks_cli_1.12.0_linux_amd64 200",
				"databricks_cli_1.12.0_linux_arm64 200",
			},
		},
		{
			name:    "dev version again into existing folders",
			version: "1.12.1-dev+abcdef123456",
			uploads: 2,
			want: []string{
				"databricks_cli_linux_amd64 200",
				"databricks_cli_linux_arm64 200",
				"databricks_cli_linux_amd64 200",
				"databricks_cli_linux_arm64 200",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testserver.New(t)
			testserver.AddDefaultHandlers(server)
			var mu sync.Mutex
			var imports []string
			server.ResponseCallback = func(req *testserver.Request, resp *testserver.EncodedResponse) {
				if !strings.HasPrefix(req.URL.Path, "/api/2.0/workspace-files/import-file/") {
					return
				}
				mu.Lock()
				defer mu.Unlock()
				imports = append(imports, fmt.Sprintf("%s %d", path.Base(path.Dir(req.URL.Path)), resp.StatusCode))
			}
			client, err := databricks.NewWorkspaceClient(&databricks.Config{
				Host:  server.URL,
				Token: "testtoken",
			})
			require.NoError(t, err)
			workspaceFiler, err := filer.NewWorkspaceFilesClient(client, "/Workspace/Users/tester@databricks.test/.databricks/ssh-tunnel/"+tt.version)
			require.NoError(t, err)

			archive := releaseArchive(t)
			getRelease := func(ctx context.Context, architecture, version, releasesDir string) (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(archive)), nil
			}
			for range tt.uploads {
				err = uploadReleases(t.Context(), workspaceFiler, getRelease, tt.version, "")
				require.NoError(t, err)
			}

			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, tt.want, imports)
		})
	}
}
