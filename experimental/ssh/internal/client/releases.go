package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/databricks/cli/experimental/ssh/internal/releases"
	"github.com/databricks/cli/experimental/ssh/internal/workspace"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/agent"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
	"golang.org/x/net/http2"
)

type releaseProvider func(ctx context.Context, architecture, version, releasesDir string) (io.ReadCloser, error)

// githubReleaseProvider is the provider UploadTunnelReleases tries first. Tests replace it
// to block GitHub.
var githubReleaseProvider releaseProvider = getGithubRelease

// githubDownloadTimeout bounds a GitHub download, body included: a proxy that answers and
// then stalls must count as a GitHub failure, so the PyPI fallback still runs.
const githubDownloadTimeout = 10 * time.Minute

func UploadTunnelReleases(ctx context.Context, client *databricks.WorkspaceClient, version, releasesDir string) error {
	versionedDir, err := workspace.GetWorkspaceVersionedDir(ctx, client, version)
	if err != nil {
		return fmt.Errorf("failed to get versioned directory: %w", err)
	}

	// Upload the CLI bundle over HTTP/1.1. It is a single ~14 MB POST, so HTTP/2
	// buys us nothing, and some corporate proxies reset large HTTP/2 request bodies
	// with RST_STREAM(NO_ERROR), which aborts the upload. Forcing HTTP/1.1 only for
	// this client keeps the rest of the connect flow on HTTP/2.
	uploadClient, err := newHTTP11WorkspaceClient(ctx, client.Config)
	if err != nil {
		return fmt.Errorf("failed to create upload client: %w", err)
	}
	workspaceFiler, err := filer.NewWorkspaceFilesClient(uploadClient, versionedDir)
	if err != nil {
		return fmt.Errorf("failed to create workspace files client: %w", err)
	}

	getRelease := withPyPIFallback(githubReleaseProvider, getPyPIRelease)
	if releasesDir != "" {
		getRelease = getLocalRelease
	}
	return uploadReleases(ctx, workspaceFiler, getRelease, version, releasesDir)
}

// withPyPIFallback returns a provider that downloads the release from GitHub and, when that
// fails, from the CLI's wheel on PyPI: networks that block GitHub often allow PyPI or point pip
// at a mirror of it. Once GitHub fails, the provider goes straight to PyPI for the remaining
// architectures. Development versions are not published to PyPI, so their GitHub error is
// returned as it is, as is the error of a cancelled command.
func withPyPIFallback(github, pypi releaseProvider) releaseProvider {
	var githubErr error
	return func(ctx context.Context, architecture, version, releasesDir string) (io.ReadCloser, error) {
		if githubErr == nil {
			release, err := github(ctx, architecture, version, releasesDir)
			if err == nil || build.IsDevelopmentVersion(version) || ctx.Err() != nil {
				return release, err
			}
			githubErr = err
			// Expected on networks that block GitHub, so not a warning. If PyPI fails too, the
			// error below carries the GitHub failure.
			log.Infof(ctx, "Failed to download the CLI release from GitHub, trying PyPI: %v", githubErr)
		}
		release, pypiErr := pypi(ctx, architecture, version, releasesDir)
		if pypiErr != nil {
			return nil, fmt.Errorf("%w\n\nDownloading it from the Python package index failed too: %w\n\n"+
				"The SSH tunnel uploads the Linux build of the Databricks CLI to your workspace. %s",
				githubErr, pypiErr, fallbackHint(ctx))
		}
		return release, nil
	}
}

// fallbackHint tells what to allow when neither GitHub nor the package index serves the
// release. A mirror is suggested only when none is configured.
func fallbackHint(ctx context.Context) string {
	const github = "Allow access to GitHub (github.com, release-assets.githubusercontent.com)"
	if pypiIndexURL(ctx) != defaultPyPIIndexURL {
		return github + ", or check that the HTTPS package index set by PIP_INDEX_URL, or index-url in " +
			"your per-user pip config, serves the databricks package"
	}
	return github + " or PyPI (pypi.org, files.pythonhosted.org), or set PIP_INDEX_URL, or index-url in " +
		"your per-user pip config, to an HTTPS PyPI mirror that serves the databricks package"
}

// newHTTP11WorkspaceClient returns a workspace client that is identical to one
// built from src but negotiates HTTP/1.1 only, by setting an HTTP/1.1 transport
// on its config. config.Config embeds a sync.Mutex and cannot be copied by value,
// so we reconstruct a fresh config from src's public attributes rather than clone
// the struct; the client then re-resolves auth from those attributes.
func newHTTP11WorkspaceClient(ctx context.Context, src *config.Config) (*databricks.WorkspaceClient, error) {
	cfg := &config.Config{}
	for _, attr := range config.ConfigAttributes {
		if attr.IsZero(src) {
			continue
		}
		if err := attr.SetS(cfg, attr.GetString(src)); err != nil {
			return nil, fmt.Errorf("failed to copy config attribute %s: %w", attr.Name, err)
		}
	}
	cfg.HTTPTransport = newHTTP11Transport(src)
	agent.ApplyNotAnAgentHeader(ctx, cfg)
	return databricks.NewWorkspaceClient((*databricks.Config)(cfg))
}

// newHTTP11Transport builds a fresh HTTP/1.1-only transport, mirroring the SDK's
// default transport tuning (see httpclient.makeDefaultTransport) but with HTTP/2
// left off. It must construct a fresh transport rather than clone one and disable
// HTTP/2 afterwards: cloning an HTTP/2-capable transport and then clearing
// ForceAttemptHTTP2 / TLSNextProto leaves the clone in an inconsistent state that
// fails every request with a bare EOF. A transport built HTTP/1.1-only from the
// start negotiates http/1.1 cleanly. See https://pkg.go.dev/net/http#Transport
func newHTTP11Transport(src *config.Config) *http.Transport {
	t := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       180 * time.Second,
		TLSHandshakeTimeout:   30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		// ForceAttemptHTTP2 is intentionally left false, and a non-nil, empty
		// TLSNextProto disables the transport's automatic HTTP/2 support.
		TLSNextProto: map[string]func(string, *tls.Conn) http.RoundTripper{},
	}
	if src.InsecureSkipVerify {
		t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return t
}

func uploadReleases(ctx context.Context, workspaceFiler filer.Filer, getRelease releaseProvider, version, releasesDir string) error {
	architectures := []string{"amd64", "arm64"}

	for _, arch := range architectures {
		fileName := getReleaseName(arch, version)
		remoteSubFolder := strings.TrimSuffix(fileName, ".zip")
		remoteBinaryPath := filepath.ToSlash(filepath.Join(remoteSubFolder, "databricks"))
		remoteArchivePath := filepath.ToSlash(filepath.Join(remoteSubFolder, "databricks.zip"))

		if build.IsDevelopmentVersion(version) {
			log.Infof(ctx, "Development version %s, overwriting %s in the workspace", version, remoteBinaryPath)
		} else {
			_, err := workspaceFiler.Stat(ctx, remoteBinaryPath)
			if err == nil {
				log.Infof(ctx, "File %s already exists in the workspace, skipping upload", remoteBinaryPath)
				continue
			} else if !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("failed to check if file %s exists in workspace: %w", remoteBinaryPath, err)
			}
		}

		err := workspaceFiler.Mkdir(ctx, remoteSubFolder)
		if err != nil {
			return fmt.Errorf("failed to create directory %s in workspace: %w", remoteSubFolder, err)
		}

		releaseReader, err := getRelease(ctx, arch, version, releasesDir)
		if err != nil {
			return fmt.Errorf("failed to get archive for architecture %s: %w", arch, err)
		}
		defer releaseReader.Close()

		log.Infof(ctx, "Uploading %s to the workspace", fileName)
		// workspace-files/import-file API will automatically unzip the payload,
		// producing the filerRoot/remoteSubFolder/*archive-contents* structure, with 'databricks' binary inside.
		err = workspaceFiler.Write(ctx, remoteArchivePath, releaseReader, filer.OverwriteIfExists, filer.CreateParentDirectories)
		if err != nil {
			if isProxyUploadError(err) {
				return fmt.Errorf("failed to upload file %s to workspace: %w\n\n"+
					"The upload was rejected before it finished. The CLI already sends this upload over HTTP/1.1, "+
					"so this is most likely a network intermediary (corporate egress proxy, VPN, or firewall/WAF) "+
					"enforcing a request-body size limit on POSTs to *.cloud.databricks.com. "+
					"Ask your network administrator to allow large uploads to that path, "+
					"or run this command from a network without such restrictions",
					remoteArchivePath, err)
			}
			return fmt.Errorf("failed to upload file %s to workspace: %w", remoteArchivePath, err)
		}
		log.Infof(ctx, "Successfully uploaded %s to workspace", remoteBinaryPath)
	}

	return nil
}

// isProxyUploadError reports whether err looks like the binary upload was rejected
// or severed by a network intermediary (corporate proxy / VPN / firewall / WAF)
// rather than by Databricks — typically an enforced request-body size limit. Because
// the upload runs over HTTP/1.1 (see newHTTP11Transport), the usual signatures are a
// 413 response or a connection reset mid-body. The HTTP/2 stream-reset checks are
// kept as a guard in case the upload ever runs over HTTP/2 again; that error reaches
// us either as a typed http2.StreamError or, when a transport layer re-formats it,
// as a string that still preserves the "stream error ... stream ID" shape.
func isProxyUploadError(err error) bool {
	if aerr, ok := errors.AsType[*apierr.APIError](err); ok {
		return aerr.StatusCode == http.StatusRequestEntityTooLarge
	}
	if errors.Is(err, syscall.ECONNRESET) {
		return true
	}
	if _, ok := errors.AsType[http2.StreamError](err); ok {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "stream error") && strings.Contains(msg, "stream ID")
}

func getReleaseName(architecture, version string) string { return releases.Name(architecture, version) }

func getLocalRelease(ctx context.Context, architecture, version, releasesDir string) (io.ReadCloser, error) {
	return releases.OpenLocal(ctx, architecture, version, releasesDir)
}

// getGithubRelease downloads the release archive from GitHub into memory, so that a body
// that is cut off or stalls fails here rather than during the upload.
func getGithubRelease(ctx context.Context, architecture, version, releasesDir string) (io.ReadCloser, error) {
	ctx, cancel := context.WithTimeout(ctx, githubDownloadTimeout)
	defer cancel()
	body, err := releases.OpenPublished(ctx, architecture, version, releasesDir)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	archive, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to download %s from GitHub: %w", getReleaseName(architecture, version), err)
	}
	return io.NopCloser(bytes.NewReader(archive)), nil
}
