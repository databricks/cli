package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/databricks/cli/experimental/ssh/internal/workspace"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	sdkclient "github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/httpclient"
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
	// with RST_STREAM(NO_ERROR), which aborts the upload (see DECO-27497). Forcing
	// HTTP/1.1 only for this client keeps the rest of the connect flow on HTTP/2.
	uploadClient, err := newHTTP11Client(client.Config)
	if err != nil {
		return fmt.Errorf("failed to create upload client: %w", err)
	}
	workspaceFiler := filer.NewWorkspaceFilesClientWithClient(client, versionedDir, uploadClient)

	getRelease := getGithubRelease
	if releasesDir != "" {
		getRelease = getLocalRelease
	}
	return uploadReleases(ctx, workspaceFiler, getRelease, version, releasesDir)
}

// newHTTP11Client returns an SDK client derived from cfg that negotiates HTTP/1.1
// only. cfg is reused, not copied (it embeds a sync.Mutex); only the transport is
// overridden, mirroring how client.New builds its client from the same config.
func newHTTP11Client(cfg *config.Config) (*sdkclient.DatabricksClient, error) {
	clientCfg, err := config.HTTPClientConfigFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	clientCfg.Transport = newHTTP11Transport(cfg)
	return sdkclient.NewWithClient(cfg, httpclient.NewApiClient(clientCfg))
}

// newHTTP11Transport clones cfg's transport (or the default) and disables HTTP/2.
// A non-nil, empty TLSNextProto map is the documented way to turn off the transport's
// automatic HTTP/2 support. See https://pkg.go.dev/net/http#Transport
func newHTTP11Transport(cfg *config.Config) *http.Transport {
	t, ok := cfg.HTTPTransport.(*http.Transport)
	if ok && t != nil {
		t = t.Clone()
	} else {
		t = http.DefaultTransport.(*http.Transport).Clone()
	}
	t.ForceAttemptHTTP2 = false
	t.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
	// Cloning http.DefaultTransport drops the InsecureSkipVerify the SDK would
	// otherwise apply, so re-apply it here to honor the resolved config.
	if cfg.InsecureSkipVerify {
		if t.TLSClientConfig == nil {
			t.TLSClientConfig = &tls.Config{}
		} else {
			t.TLSClientConfig = t.TLSClientConfig.Clone()
		}
		t.TLSClientConfig.InsecureSkipVerify = true
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
