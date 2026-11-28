package client

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"sync"
	"syscall"
	"testing"

	"github.com/databricks/cli/experimental/ssh/internal/workspace"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/iam"
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

	w, err := newHTTP11WorkspaceClient(t.Context(), src)
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

func TestWithPyPIFallback(t *testing.T) {
	errGithub := errors.New("dial tcp: lookup github.com: no such host")
	errPyPI := errors.New("failed to fetch the databricks package from pypi.org: HTTP 403")

	provider := func(name string, err error, calls *[]string) releaseProvider {
		return func(ctx context.Context, architecture, version, releasesDir string) (io.ReadCloser, error) {
			*calls = append(*calls, name)
			if err != nil {
				return nil, err
			}
			return io.NopCloser(strings.NewReader(name)), nil
		}
	}

	tests := []struct {
		name      string
		version   string
		githubErr error
		pypiErr   error
		// cancelled cancels the command before the download.
		cancelled bool
		wantCalls []string
		want      string
		wantErrs  []error
	}{
		{
			name:      "GitHub works",
			version:   testWheelVersion,
			wantCalls: []string{"github"},
			want:      "github",
		},
		{
			name:      "GitHub fails",
			version:   testWheelVersion,
			githubErr: errGithub,
			wantCalls: []string{"github", "pypi"},
			want:      "pypi",
		},
		{
			name:      "both fail",
			version:   testWheelVersion,
			githubErr: errGithub,
			pypiErr:   errPyPI,
			wantCalls: []string{"github", "pypi"},
			wantErrs:  []error{errGithub, errPyPI},
		},
		{
			name:      "command cancelled",
			version:   testWheelVersion,
			githubErr: context.Canceled,
			cancelled: true,
			wantCalls: []string{"github"},
			wantErrs:  []error{context.Canceled},
		},
		{
			name:      "development versions are not on PyPI",
			version:   "1.27.1-dev+abcdef123456",
			githubErr: errGithub,
			wantCalls: []string{"github"},
			wantErrs:  []error{errGithub},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls []string
			getRelease := withPyPIFallback(provider("github", tt.githubErr, &calls), provider("pypi", tt.pypiErr, &calls))
			ctx, cancel := context.WithCancel(pipConfigContext(t, "", ""))
			defer cancel()
			if tt.cancelled {
				cancel()
			}

			release, err := getRelease(ctx, "amd64", tt.version, "")
			assert.Equal(t, tt.wantCalls, calls)
			if tt.wantErrs != nil {
				for _, wantErr := range tt.wantErrs {
					assert.ErrorIs(t, err, wantErr)
				}
				return
			}
			require.NoError(t, err)
			got, err := io.ReadAll(release)
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestWithPyPIFallbackSkipsGitHubOnceItFails(t *testing.T) {
	var calls []string
	github := func(ctx context.Context, architecture, version, releasesDir string) (io.ReadCloser, error) {
		calls = append(calls, "github "+architecture)
		return nil, errors.New("dial tcp: i/o timeout")
	}
	pypi := func(ctx context.Context, architecture, version, releasesDir string) (io.ReadCloser, error) {
		calls = append(calls, "pypi "+architecture)
		return io.NopCloser(strings.NewReader(architecture)), nil
	}
	getRelease := withPyPIFallback(github, pypi)
	for _, arch := range []string{"amd64", "arm64"} {
		release, err := getRelease(t.Context(), arch, testWheelVersion, "")
		require.NoError(t, err)
		require.NoError(t, release.Close())
	}
	assert.Equal(t, []string{"github amd64", "pypi amd64", "pypi arm64"}, calls)
}

func TestWithPyPIFallbackNamesWhatToAllow(t *testing.T) {
	failing := func(ctx context.Context, architecture, version, releasesDir string) (io.ReadCloser, error) {
		return nil, errors.New("blocked")
	}

	tests := []struct {
		name        string
		pipIndexURL string
		wantErr     string
	}{
		{
			name: "no index configured",
			wantErr: "Downloading it from the Python package index failed too: blocked\n\n" +
				"The SSH tunnel uploads the Linux build of the Databricks CLI to your workspace. " +
				"Allow access to GitHub (github.com, release-assets.githubusercontent.com) or PyPI " +
				"(pypi.org, files.pythonhosted.org), or set PIP_INDEX_URL, or index-url in your per-user pip config, " +
				"to an HTTPS PyPI mirror that serves the databricks package",
		},
		{
			name:        "mirror configured",
			pipIndexURL: "https://pypi-mirror.test/simple",
			wantErr: "Allow access to GitHub (github.com, release-assets.githubusercontent.com), or check that the " +
				"HTTPS package index set by PIP_INDEX_URL, or index-url in your per-user pip config, serves the databricks package",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := pipConfigContext(t, tt.pipIndexURL, "")
			_, err := withPyPIFallback(failing, failing)(ctx, "amd64", testWheelVersion, "")
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

// TestUploadReleasesFromPyPI uploads, with GitHub blocked, the archives built from the wheels
// to the fake workspace, which unpacks them as the import-file endpoint does, and reads back
// the binaries the bootstrap runs.
func TestUploadReleasesFromPyPI(t *testing.T) {
	binaries := testBinaries()
	index := newTestIndex(t, testIndexFiles(t, binaries), testIndexOptions{})
	ctx := pipConfigContext(t, index.URL+"/simple", "")

	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:  server.URL,
		Token: "testtoken",
	})
	require.NoError(t, err)

	github := githubReleaseProvider
	t.Cleanup(func() { githubReleaseProvider = github })
	githubReleaseProvider = func(ctx context.Context, architecture, version, releasesDir string) (io.ReadCloser, error) {
		return nil, errors.New("failed to download: HTTP 403")
	}
	require.NoError(t, UploadTunnelReleases(ctx, client, testWheelVersion, ""))

	versionedDir, err := workspace.GetWorkspaceVersionedDir(ctx, client, testWheelVersion)
	require.NoError(t, err)
	workspaceFiler, err := filer.NewWorkspaceFilesClient(client, versionedDir)
	require.NoError(t, err)
	for arch, binary := range binaries {
		remoteBinaryPath := path.Join(strings.TrimSuffix(getReleaseName(arch, testWheelVersion), ".zip"), "databricks")
		r, err := workspaceFiler.Read(ctx, remoteBinaryPath)
		require.NoError(t, err)
		got, err := io.ReadAll(r)
		require.NoError(t, err)
		require.NoError(t, r.Close())
		assert.Equal(t, binary, got, arch)
	}
}

// TestNewHTTP11WorkspaceClientNotAnAgentDropsAgentHeader checks that the upload client leaves the agent token out of
// User-Agent under DATABRICKS_NOT_AN_AGENT, and keeps it without the variable.
func TestNewHTTP11WorkspaceClientNotAnAgentDropsAgentHeader(t *testing.T) {
	testutil.SetAgentEnv(t, "CLAUDECODE", "1")

	cases := []struct {
		name    string
		noAgent bool
	}{
		{name: "variable set", noAgent: true},
		{name: "variable unset", noAgent: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var userAgent string
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				// The SDK first asks for the host's well-known config; only the Me call is checked.
				if r.URL.Path != "/api/2.0/preview/scim/v2/Me" {
					rw.WriteHeader(http.StatusNotFound)
					return
				}
				userAgent = r.Header.Get("User-Agent")
				rw.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(rw, `{"id":"1","userName":"user@example.com"}`)
			}))
			t.Cleanup(server.Close)

			ctx := t.Context()
			if tc.noAgent {
				t.Setenv("DATABRICKS_NOT_AN_AGENT", "1")
			}
			w, err := newHTTP11WorkspaceClient(ctx, &config.Config{Host: server.URL, Token: "dapi-test"})
			require.NoError(t, err)
			_, err = w.CurrentUser.Me(ctx, iam.MeRequest{})
			require.NoError(t, err)

			if tc.noAgent {
				assert.NotContains(t, userAgent, "agent/")
			} else {
				assert.Contains(t, userAgent, "agent/claude-code")
			}
		})
	}
}
