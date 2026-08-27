package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/databricks/cli/cmd/ssh/internal/workspace"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
	"golang.org/x/net/http2"
)

type releaseProvider func(ctx context.Context, architecture, version, releasesDir string) (io.ReadCloser, error)

func UploadTunnelReleases(ctx context.Context, client *databricks.WorkspaceClient, version, releasesDir string) error {
	versionedDir, err := workspace.GetWorkspaceVersionedDir(ctx, client, version)
	if err != nil {
		return fmt.Errorf("failed to get versioned directory: %w", err)
	}

	// Upload the CLI bundle over HTTP/1.1. It is a single ~14 MB POST, so HTTP/2
	// buys us nothing, and some corporate proxies reset large HTTP/2 request bodies
	// with RST_STREAM(NO_ERROR), which aborts the upload. Forcing HTTP/1.1 only for
	// this client keeps the rest of the connect flow on HTTP/2.
	uploadClient, err := newHTTP11WorkspaceClient(client.Config)
	if err != nil {
		return fmt.Errorf("failed to create upload client: %w", err)
	}
	workspaceFiler, err := filer.NewWorkspaceFilesClient(uploadClient, versionedDir)
	if err != nil {
		return fmt.Errorf("failed to create workspace files client: %w", err)
	}

	getRelease := getGithubRelease
	if releasesDir != "" {
		getRelease = getLocalRelease
	}
	return uploadReleases(ctx, workspaceFiler, getRelease, version, releasesDir)
}

// newHTTP11WorkspaceClient returns a workspace client that is identical to one
// built from src but negotiates HTTP/1.1 only, by setting an HTTP/1.1 transport
// on its config. config.Config embeds a sync.Mutex and cannot be copied by value,
// so we reconstruct a fresh config from src's public attributes rather than clone
// the struct; the client then re-resolves auth from those attributes.
func newHTTP11WorkspaceClient(src *config.Config) (*databricks.WorkspaceClient, error) {
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

		_, err := workspaceFiler.Stat(ctx, remoteBinaryPath)
		if err == nil {
			log.Infof(ctx, "File %s already exists in the workspace, skipping upload", remoteBinaryPath)
			continue
		} else if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("failed to check if file %s exists in workspace: %w", remoteBinaryPath, err)
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

func getReleaseName(architecture, version string) string {
	if strings.Contains(version, "dev") {
		return fmt.Sprintf("databricks_cli_linux_%s.zip", architecture)
	}
	return fmt.Sprintf("databricks_cli_%s_linux_%s.zip", version, architecture)
}

func getLocalRelease(ctx context.Context, architecture, version, releasesDir string) (io.ReadCloser, error) {
	log.Infof(ctx, "Looking for CLI releases in directory: %s", releasesDir)
	releaseName := getReleaseName(architecture, version)
	releasePath := filepath.Join(releasesDir, releaseName)
	file, err := os.Open(releasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", releasePath, err)
	}
	return file, nil
}

func getGithubRelease(ctx context.Context, architecture, version, releasesDir string) (io.ReadCloser, error) {
	// TODO: download and check databricks_cli_<version>_SHA256SUMS
	fileName := getReleaseName(architecture, version)
	downloadURL := fmt.Sprintf("https://github.com/databricks/cli/releases/download/v%s/%s", version, fileName)
	log.Infof(ctx, "Downloading %s from %s", fileName, downloadURL)

	resp, err := http.Get(downloadURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download %s: %w", downloadURL, err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("failed to download %s: HTTP %d", downloadURL, resp.StatusCode)
	}

	return resp.Body, nil
}
