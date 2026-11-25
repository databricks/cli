package client

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testWheelVersion = "1.27.0"

// The platform tags of the CLI wheels, from tools/build_cli_wheels.py.
var testWheelPlatforms = map[string]string{
	"amd64": "manylinux_2_17_x86_64.manylinux2014_x86_64.musllinux_1_1_x86_64",
	"arm64": "manylinux_2_17_aarch64.manylinux2014_aarch64.musllinux_1_1_aarch64",
}

func testWheelName(version, platform string) string {
	return "databricks-" + version + "-py3-none-" + platform + ".whl"
}

// testWheel returns a wheel laid out as tools/build_cli_wheels.py lays out the CLI wheels.
func testWheel(t *testing.T, version string, binary []byte) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files := []struct {
		name string
		data []byte
		mode fs.FileMode
	}{
		{"databricks-" + version + ".data/scripts/databricks", binary, 0o755},
		{"databricks-" + version + ".dist-info/METADATA", []byte("Metadata-Version: 2.4\nName: databricks\nVersion: " + version + "\n"), 0o644},
		{"databricks-" + version + ".dist-info/WHEEL", []byte("Wheel-Version: 1.0\nRoot-Is-Purelib: false\n"), 0o644},
		{"databricks-" + version + ".dist-info/RECORD", []byte{}, 0o644},
	}
	for _, f := range files {
		header := &zip.FileHeader{Name: f.name, Method: zip.Deflate}
		header.SetMode(f.mode)
		w, err := zw.CreateHeader(header)
		require.NoError(t, err)
		_, err = w.Write(f.data)
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

// testBinaries returns a distinct stand-in binary per architecture.
func testBinaries() map[string][]byte {
	return map[string][]byte{
		"amd64": bytes.Repeat([]byte("ELF amd64 databricks "), 1000),
		"arm64": bytes.Repeat([]byte("ELF arm64 databricks "), 1000),
	}
}

// testIndexFiles returns the files of the databricks project: the Linux wheels of the
// version holding the binaries, and wheels and sdists that must not be picked.
func testIndexFiles(t *testing.T, binaries map[string][]byte) map[string][]byte {
	files := map[string][]byte{
		"databricks-0.2.tar.gz":                              []byte("an unrelated old sdist"),
		"databricks-0.2-py2.py3-none-any.whl":                testWheel(t, "0.2", []byte("an unrelated old wheel")),
		testWheelName(testWheelVersion, "macosx_13_0_arm64"): testWheel(t, testWheelVersion, []byte("Mach-O")),
		testWheelName(testWheelVersion, "win_amd64"):         testWheel(t, testWheelVersion, []byte("PE")),
	}
	for arch, binary := range binaries {
		files[testWheelName(testWheelVersion, testWheelPlatforms[arch])] = testWheel(t, testWheelVersion, binary)
		files[testWheelName("1.26.0", testWheelPlatforms[arch])] = testWheel(t, "1.26.0", []byte("the previous release"))
	}
	return files
}

type testIndexOptions struct {
	// prefix is the path the index is served under, such as /api/pypi/pypi-remote on Artifactory.
	prefix string
	// relativeLinks links files relative to the project page, as Artifactory and Nexus do,
	// instead of with absolute URLs, as PyPI and other mirrors do.
	relativeLinks bool
	// filesURL, when set, is the server that absolute links point to instead of this one, as
	// PyPI points to files.pythonhosted.org.
	filesURL string
	// digest renders the URL fragment of a file link; nil renders #sha256=<digest>.
	digest func(sum string) string
	// username and password, when set, are required on every request.
	username, password string
	// rejectCredentials fails every request that carries credentials, or a Referer, which
	// can carry the query string of the URL it came from.
	rejectCredentials bool
	// redirectURL, when set, is the server that /moved redirects to instead of this one.
	redirectURL string
	// dropDownloads closes the connection of every file download instead of answering it.
	dropDownloads bool
	// undigestedCopies lists every file a second time, before the copy with the digest,
	// without a digest.
	undigestedCopies bool
}

// newTestIndex serves over HTTPS the PEP 503 project page of databricks at
// <prefix>/simple/databricks/, the files it links to under <prefix>/packages/, a relative
// redirect to the project page from /moved<prefix>/simple/databricks/, an absolute one from
// /moved-absolute<prefix>/simple/databricks/, and a redirect with a Location header that
// does not parse, and carries a token, from /bad-redirect/simple/databricks/. The PyPI
// client trusts it until the test ends.
func newTestIndex(t *testing.T, files map[string][]byte, opts testIndexOptions) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+opts.prefix+"/simple/databricks/", func(w http.ResponseWriter, r *http.Request) {
		filesURL := "https://" + r.Host + opts.prefix
		if opts.filesURL != "" {
			filesURL = opts.filesURL
		}
		var page strings.Builder
		page.WriteString("<!DOCTYPE html>\n<html><head><meta name=\"pypi:repository-version\" content=\"1.1\"></head><body>\n")
		for _, name := range slices.Sorted(maps.Keys(files)) {
			sum := sha256.Sum256(files[name])
			fragment := "#sha256=" + hex.EncodeToString(sum[:])
			if opts.digest != nil {
				fragment = opts.digest(hex.EncodeToString(sum[:]))
			}
			href := filesURL + "/packages/" + name
			if opts.relativeLinks {
				href = "../../packages/" + name
			}
			if opts.undigestedCopies {
				fmt.Fprintf(&page, "<a href=\"%s\">%s</a><br />\n", href, name)
			}
			fmt.Fprintf(&page, "<a href=\"%s\" data-requires-python=\"&gt;=3.8\">%s</a><br />\n", href+fragment, name)
		}
		page.WriteString("</body></html>\n")
		_, _ = io.WriteString(w, page.String())
	})
	mux.HandleFunc("GET "+opts.prefix+"/packages/{name}", func(w http.ResponseWriter, r *http.Request) {
		if opts.dropDownloads {
			if conn, _, err := http.NewResponseController(w).Hijack(); err == nil {
				conn.Close()
			}
			return
		}
		data, ok := files[r.PathValue("name")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(data)
	})
	mux.HandleFunc("GET /moved"+opts.prefix+"/simple/databricks/", func(w http.ResponseWriter, r *http.Request) {
		target := opts.prefix + "/simple/databricks/"
		if opts.redirectURL != "" {
			target = opts.redirectURL + target
		}
		http.Redirect(w, r, target, http.StatusFound)
	})
	mux.HandleFunc("GET /moved-absolute"+opts.prefix+"/simple/databricks/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://"+r.Host+opts.prefix+"/simple/databricks/", http.StatusFound)
	})
	mux.HandleFunc("GET /bad-redirect/simple/databricks/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/signed/token-secret%zz")
		w.WriteHeader(http.StatusFound)
	})

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The mux redirects a path with a doubled slash to its clean form. An index need not,
		// so such a request is a miss here.
		if strings.Contains(r.URL.Path, "//") {
			http.NotFound(w, r)
			return
		}
		if opts.rejectCredentials && (r.Header.Get("Authorization") != "" || r.Header.Get("Referer") != "") {
			http.Error(w, "this host must not receive credentials", http.StatusBadRequest)
			return
		}
		if opts.username != "" {
			if username, password, ok := r.BasicAuth(); !ok || username != opts.username || password != opts.password {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	trustTestServer(t, server)
	return server
}

// trustTestServer makes the PyPI client trust the certificate of the test server, which
// every httptest server shares, until the test ends.
func trustTestServer(t *testing.T, server *httptest.Server) {
	transport := pypiHTTPClient.Transport
	t.Cleanup(func() { pypiHTTPClient.Transport = transport })
	pypiHTTPClient.Transport = redirectLocationTransport{base: server.Client().Transport}
}

// withCredentials returns serverURL with the credentials as its userinfo.
func withCredentials(serverURL, username, password string) string {
	return strings.Replace(serverURL, "https://", "https://"+username+":"+password+"@", 1)
}

// pipConfigContext returns a context in which pip is configured only by PIP_INDEX_URL, when
// not empty, and by a per-user pip config with index-url set to pipConfIndexURL, when not empty.
func pipConfigContext(t *testing.T, pipIndexURL, pipConfIndexURL string) context.Context {
	home := t.TempDir()
	appData := filepath.Join(home, "AppData", "Roaming")
	ctx := env.WithUserHomeDir(t.Context(), home)
	ctx = env.Set(ctx, "APPDATA", appData)
	ctx = env.Set(ctx, "PIP_INDEX_URL", pipIndexURL)
	if pipConfIndexURL != "" {
		conf := "[global]\nindex-url = " + pipConfIndexURL + "\n"
		// The per-user pip config is ~/.config/pip/pip.conf on Linux and macOS, and
		// %APPDATA%\pip\pip.ini on Windows.
		for _, p := range []string{filepath.Join(home, ".config", "pip", "pip.conf"), filepath.Join(appData, "pip", "pip.ini")} {
			require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
			require.NoError(t, os.WriteFile(p, []byte(conf), 0o644))
		}
	}
	return ctx
}

// archiveEntries returns the files of a release archive with their contents.
func archiveEntries(t *testing.T, release io.Reader) map[string]string {
	data, err := io.ReadAll(release)
	require.NoError(t, err)
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	entries := map[string]string{}
	for _, f := range zr.File {
		r, err := f.Open()
		require.NoError(t, err)
		content, err := io.ReadAll(r)
		require.NoError(t, err)
		require.NoError(t, r.Close())
		entries[f.Name] = string(content)
	}
	return entries
}

// requireReleases checks that the archives getPyPIRelease builds hold the binaries.
func requireReleases(t *testing.T, ctx context.Context, binaries map[string][]byte) {
	for arch, binary := range binaries {
		release, err := getPyPIRelease(ctx, arch, testWheelVersion, "")
		require.NoError(t, err, arch)
		assert.Equal(t, map[string]string{"databricks": string(binary)}, archiveEntries(t, release), arch)
		require.NoError(t, release.Close())
	}
}

func TestGetPyPIRelease(t *testing.T) {
	tests := []struct {
		name string
		opts testIndexOptions
		// indexURL renders the index URL from the server URL.
		indexURL func(serverURL string) string
	}{
		{
			name:     "absolute links",
			indexURL: func(serverURL string) string { return serverURL + "/simple" },
		},
		{
			name:     "relative links and an index URL with a trailing slash",
			opts:     testIndexOptions{prefix: "/api/pypi/pypi-remote", relativeLinks: true},
			indexURL: func(serverURL string) string { return serverURL + "/api/pypi/pypi-remote/simple/" },
		},
		{
			name:     "relative links on a project page that redirects",
			opts:     testIndexOptions{prefix: "/api/pypi/pypi-remote", relativeLinks: true},
			indexURL: func(serverURL string) string { return serverURL + "/moved/api/pypi/pypi-remote/simple" },
		},
		{
			name: "index credentials sent to the index host for absolute links",
			opts: testIndexOptions{username: "user", password: "secret"},
			indexURL: func(serverURL string) string {
				return withCredentials(serverURL, "user", "secret") + "/simple"
			},
		},
		{
			name: "index credentials sent to the index host for relative links",
			opts: testIndexOptions{prefix: "/api/pypi/pypi-remote", relativeLinks: true, username: "user", password: "secret"},
			indexURL: func(serverURL string) string {
				return withCredentials(serverURL, "user", "secret") + "/api/pypi/pypi-remote/simple"
			},
		},
		{
			name: "index credentials kept on an absolute redirect to the index host",
			opts: testIndexOptions{prefix: "/api/pypi/pypi-remote", relativeLinks: true, username: "user", password: "secret"},
			indexURL: func(serverURL string) string {
				return withCredentials(serverURL, "user", "secret") + "/moved-absolute/api/pypi/pypi-remote/simple"
			},
		},
		{
			name:     "wheel listed again with a digest after a copy without one",
			opts:     testIndexOptions{undigestedCopies: true},
			indexURL: func(serverURL string) string { return serverURL + "/simple" },
		},
		{
			name:     "upper-case digest",
			opts:     testIndexOptions{digest: func(sum string) string { return "#sha256=" + strings.ToUpper(sum) }},
			indexURL: func(serverURL string) string { return serverURL + "/simple" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			binaries := testBinaries()
			index := newTestIndex(t, testIndexFiles(t, binaries), tt.opts)
			requireReleases(t, pipConfigContext(t, tt.indexURL(index.URL), ""), binaries)
		})
	}
}

func TestGetPyPIReleaseSendsCredentialsOnlyToTheIndexHost(t *testing.T) {
	// Same machine, other port: a different host, as pip and net/http see it.
	binaries := testBinaries()
	files := testIndexFiles(t, binaries)
	filesHost := newTestIndex(t, files, testIndexOptions{rejectCredentials: true})
	index := newTestIndex(t, files, testIndexOptions{username: "user", password: "secret", filesURL: filesHost.URL})

	requireReleases(t, pipConfigContext(t, withCredentials(index.URL, "user", "secret")+"/simple", ""), binaries)
}

func TestGetPyPIReleaseKeepsCredentialsOffRedirectsToOtherHosts(t *testing.T) {
	// The index redirects its project page to a mirror on another host, which must receive
	// neither the index credentials nor, in a Referer, the token in the index URL.
	binaries := testBinaries()
	files := testIndexFiles(t, binaries)
	mirror := newTestIndex(t, files, testIndexOptions{rejectCredentials: true})
	index := newTestIndex(t, files, testIndexOptions{username: "user", password: "secret", redirectURL: mirror.URL})

	indexURL := withCredentials(index.URL, "user", "secret") + "/moved/simple?token=secret"
	requireReleases(t, pipConfigContext(t, indexURL, ""), binaries)
}

func TestCheckPyPIRedirect(t *testing.T) {
	const indexURL = "https://user:secret@index.test/simple/databricks/?token=secret"
	const authorization = "Basic dXNlcjpzZWNyZXQ="

	tests := []struct {
		name   string
		target string
		// followed is the number of redirects followed before this one.
		followed int
		wantURL  string
		// wantAuthorization is the Authorization header left on the redirect, which starts
		// with the one net/http copied from the previous request.
		wantAuthorization string
		wantErr           string
	}{
		{
			name:              "absolute URL on the index origin",
			target:            "https://index.test/packages/databricks.whl",
			wantURL:           "https://user:secret@index.test/packages/databricks.whl",
			wantAuthorization: authorization,
		},
		{
			name:              "index origin with the default port written out",
			target:            "https://index.test:443/packages/databricks.whl",
			wantURL:           "https://user:secret@index.test:443/packages/databricks.whl",
			wantAuthorization: authorization,
		},
		{
			name:              "index origin with an upper-case host name",
			target:            "https://INDEX.test/packages/databricks.whl",
			wantURL:           "https://user:secret@INDEX.test/packages/databricks.whl",
			wantAuthorization: authorization,
		},
		{
			name:    "other host",
			target:  "https://files.test/packages/databricks.whl",
			wantURL: "https://files.test/packages/databricks.whl",
		},
		{
			name:    "subdomain of the index host",
			target:  "https://files.index.test/packages/databricks.whl",
			wantURL: "https://files.index.test/packages/databricks.whl",
		},
		{
			name:    "other port",
			target:  "https://index.test:8443/packages/databricks.whl",
			wantURL: "https://index.test:8443/packages/databricks.whl",
		},
		{
			name:    "other host with the index credentials in the URL",
			target:  "https://user:secret@files.test/packages/databricks.whl",
			wantURL: "https://files.test/packages/databricks.whl",
		},
		{
			name:    "plain HTTP on the index host",
			target:  "http://index.test/packages/databricks.whl",
			wantErr: "refusing to follow a redirect from index.test to plain HTTP at index.test",
		},
		{
			name:    "plain HTTP on another host",
			target:  "http://files.test/packages/databricks.whl",
			wantErr: "refusing to follow a redirect from index.test to plain HTTP at files.test",
		},
		{
			name:     "too many redirects",
			target:   "https://index.test/packages/databricks.whl",
			followed: 9,
			wantErr:  "stopped after 10 redirects",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first := httptest.NewRequest(http.MethodGet, indexURL, nil)
			via := []*http.Request{first}
			for range tt.followed {
				via = append(via, first)
			}
			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			req.Header.Set("Referer", "https://index.test/simple/databricks/?token=secret")
			req.Header.Set("Authorization", authorization)

			err := checkPyPIRedirect(req, via)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantURL, req.URL.String())
			assert.Equal(t, tt.wantAuthorization, req.Header.Get("Authorization"))
			assert.Empty(t, req.Header.Get("Referer"))
		})
	}
}

func TestCheckPyPIRedirectRefusesPlainHTTPWithoutCredentials(t *testing.T) {
	// The wheel is checked against the digest on the project page, so neither may come over
	// plain HTTP, credentials or not.
	first := httptest.NewRequest(http.MethodGet, "https://index.test/simple/databricks/", nil)
	req := httptest.NewRequest(http.MethodGet, "http://index.test/packages/databricks.whl", nil)

	require.EqualError(t, checkPyPIRedirect(req, []*http.Request{first}), "refusing to follow a redirect from index.test to plain HTTP at index.test")
}

func TestGetPyPIReleaseRefusesPlainHTTPIndex(t *testing.T) {
	ctx := pipConfigContext(t, "http://user:secret@index.test/simple", "")
	_, err := getPyPIRelease(ctx, "amd64", testWheelVersion, "")
	require.EqualError(t, err, "the Python package index index.test must use HTTPS: the wheel is checked against the digest on its project page, which plain HTTP does not protect")
}

func TestGetPyPIReleaseTimesOutOnStalledProjectPage(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(server.Close)
	trustTestServer(t, server)
	timeout := indexPageTimeout
	t.Cleanup(func() { indexPageTimeout = timeout })
	indexPageTimeout = 100 * time.Millisecond

	_, err := getPyPIRelease(pipConfigContext(t, server.URL+"/simple", ""), "amd64", testWheelVersion, "")
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestGetPyPIReleaseErrors(t *testing.T) {
	binaries := testBinaries()
	files := testIndexFiles(t, binaries)

	tests := []struct {
		name    string
		files   map[string][]byte
		opts    testIndexOptions
		version string
		// indexPath is the path of the index on the server.
		indexPath string
		wantErr   string
	}{
		{
			name:      "version not on the index",
			files:     files,
			version:   "1.28.0",
			indexPath: "/simple",
			wantErr:   "has no Linux x86_64 wheel of databricks 1.28.0",
		},
		{
			name:      "no project page",
			files:     files,
			version:   testWheelVersion,
			indexPath: "/other-index",
			wantErr:   "failed to fetch the databricks package from 127.0.0.1:",
		},
		{
			name:      "digest mismatch",
			files:     files,
			opts:      testIndexOptions{digest: func(string) string { return "#sha256=" + strings.Repeat("0", 64) }},
			version:   testWheelVersion,
			indexPath: "/simple",
			wantErr:   "but the index lists " + strings.Repeat("0", 64),
		},
		{
			name:      "no digest",
			files:     files,
			opts:      testIndexOptions{digest: func(string) string { return "" }},
			version:   testWheelVersion,
			indexPath: "/simple",
			wantErr:   "lists no SHA-256 digest for " + testWheelName(testWheelVersion, testWheelPlatforms["amd64"]),
		},
		{
			name:      "only an MD5 digest",
			files:     files,
			opts:      testIndexOptions{digest: func(string) string { return "#md5=0123456789abcdef0123456789abcdef" }},
			version:   testWheelVersion,
			indexPath: "/simple",
			wantErr:   "lists no SHA-256 digest",
		},
		{
			name:      "wheel linked over plain HTTP",
			files:     files,
			opts:      testIndexOptions{filesURL: "http://files.test"},
			version:   testWheelVersion,
			indexPath: "/simple",
			wantErr:   "links " + testWheelName(testWheelVersion, testWheelPlatforms["amd64"]) + " over plain HTTP",
		},
		{
			name: "wheel without the binary",
			files: map[string][]byte{
				testWheelName(testWheelVersion, testWheelPlatforms["amd64"]): testWheel(t, "1.26.0", []byte("misnamed data directory")),
			},
			version:   testWheelVersion,
			indexPath: "/simple",
			wantErr:   "the databricks " + testWheelVersion + " wheel has no databricks-" + testWheelVersion + ".data/scripts/databricks",
		},
		{
			name: "file that is not a wheel",
			files: map[string][]byte{
				testWheelName(testWheelVersion, testWheelPlatforms["amd64"]): []byte("<html>blocked by policy</html>"),
			},
			version:   testWheelVersion,
			indexPath: "/simple",
			wantErr:   "failed to read the databricks " + testWheelVersion + " wheel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			index := newTestIndex(t, tt.files, tt.opts)
			ctx := pipConfigContext(t, index.URL+tt.indexPath, "")

			_, err := getPyPIRelease(ctx, "amd64", tt.version, "")
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestGetPyPIReleaseKeepsCredentialsOutOfErrors(t *testing.T) {
	files := testIndexFiles(t, testBinaries())
	index := newTestIndex(t, files, testIndexOptions{username: "index-user", password: "index-secret"})
	dropping := newTestIndex(t, files, testIndexOptions{relativeLinks: true, username: "index-user", password: "index-secret", dropDownloads: true})
	closed := httptest.NewTLSServer(http.NotFoundHandler())
	closed.Close()

	tests := []struct {
		name     string
		indexURL string
		wantErr  string
	}{
		{
			name:     "rejected credentials",
			indexURL: withCredentials(index.URL, "index-user", "not-the-secret") + "/simple",
			wantErr:  "HTTP 401",
		},
		{
			name:     "unreachable index",
			indexURL: withCredentials(closed.URL, "index-user", "index-secret") + "/simple",
			wantErr:  "failed to fetch the databricks package from " + strings.TrimPrefix(closed.URL, "https://"),
		},
		{
			name:     "redirect with a Location header that does not parse",
			indexURL: withCredentials(index.URL, "index-user", "index-secret") + "/bad-redirect/simple",
			wantErr:  "invalid redirect from " + strings.TrimPrefix(index.URL, "https://"),
		},
		{
			name:     "wheel download cut off",
			indexURL: withCredentials(dropping.URL, "index-user", "index-secret") + "/simple",
			wantErr:  "from " + strings.TrimPrefix(dropping.URL, "https://"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := pipConfigContext(t, tt.indexURL, "")
			_, err := getPyPIRelease(ctx, "amd64", testWheelVersion, "")
			require.ErrorContains(t, err, tt.wantErr)
			assert.NotContains(t, err.Error(), "index-user")
			assert.NotContains(t, err.Error(), "secret")
		})
	}
}

func TestPyPIIndexURL(t *testing.T) {
	tests := []struct {
		name            string
		pipIndexURL     string
		pipConfIndexURL string
		want            string
	}{
		{
			name: "PyPI by default",
			want: "https://pypi.org/simple",
		},
		{
			name:            "index-url in the pip config",
			pipConfIndexURL: "https://pypi-mirror.example/simple",
			want:            "https://pypi-mirror.example/simple",
		},
		{
			name:            "PIP_INDEX_URL overrides the pip config",
			pipIndexURL:     "https://env-mirror.example/simple",
			pipConfIndexURL: "https://pypi-mirror.example/simple",
			want:            "https://env-mirror.example/simple",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := pipConfigContext(t, tt.pipIndexURL, tt.pipConfIndexURL)
			assert.Equal(t, tt.want, pypiIndexURL(ctx))
		})
	}
}

func TestIsLinuxWheel(t *testing.T) {
	tests := []struct {
		fileName string
		machine  string
		want     bool
	}{
		{testWheelName(testWheelVersion, testWheelPlatforms["amd64"]), "x86_64", true},
		{testWheelName(testWheelVersion, testWheelPlatforms["arm64"]), "aarch64", true},
		{testWheelName(testWheelVersion, "manylinux_2_28_x86_64"), "x86_64", true},
		{testWheelName(testWheelVersion, "musllinux_1_2_aarch64"), "aarch64", true},
		{testWheelName(testWheelVersion, testWheelPlatforms["arm64"]), "x86_64", false},
		{testWheelName(testWheelVersion, testWheelPlatforms["amd64"]), "aarch64", false},
		{testWheelName(testWheelVersion, "macosx_13_0_x86_64"), "x86_64", false},
		{testWheelName(testWheelVersion, "win_amd64"), "x86_64", false},
		{testWheelName(testWheelVersion+"1", testWheelPlatforms["amd64"]), "x86_64", false},
		{testWheelName(testWheelVersion+".1", testWheelPlatforms["amd64"]), "x86_64", false},
		{testWheelName("1.26.0", testWheelPlatforms["amd64"]), "x86_64", false},
		{"databricks-" + testWheelVersion + ".tar.gz", "x86_64", false},
		{"databricks_connect-" + testWheelVersion + "-py3-none-" + testWheelPlatforms["amd64"] + ".whl", "x86_64", false},
	}

	for _, tt := range tests {
		t.Run(tt.fileName+" "+tt.machine, func(t *testing.T) {
			assert.Equal(t, tt.want, isLinuxWheel(tt.fileName, testWheelVersion, tt.machine))
		})
	}
}
