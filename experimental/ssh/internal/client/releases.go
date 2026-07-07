package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/databricks/cli/experimental/ssh/internal/workspace"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/retries"
	"golang.org/x/net/http2"
)

// uploadRetryTimeout bounds how long the transient-error retries for a single
// workspace-files operation (stat or upload) may run before giving up.
const uploadRetryTimeout = 5 * time.Minute

type releaseProvider func(ctx context.Context, architecture, version, releasesDir string) (io.ReadCloser, error)

func UploadTunnelReleases(ctx context.Context, client *databricks.WorkspaceClient, version, releasesDir string) error {
	versionedDir, err := workspace.GetWorkspaceVersionedDir(ctx, client, version)
	if err != nil {
		return fmt.Errorf("failed to get versioned directory: %w", err)
	}

	workspaceFiler, err := filer.NewWorkspaceFilesClient(client, versionedDir)
	if err != nil {
		return fmt.Errorf("failed to create workspace files client: %w", err)
	}

	getRelease := getGithubRelease
	if releasesDir != "" {
		getRelease = getLocalRelease
	}
	return uploadReleases(ctx, workspaceFiler, getRelease, version, releasesDir)
}

func uploadReleases(ctx context.Context, workspaceFiler filer.Filer, getRelease releaseProvider, version, releasesDir string) error {
	architectures := []string{"amd64", "arm64"}

	for _, arch := range architectures {
		fileName := getReleaseName(arch, version)
		remoteSubFolder := strings.TrimSuffix(fileName, ".zip")
		remoteBinaryPath := filepath.ToSlash(filepath.Join(remoteSubFolder, "databricks"))
		remoteArchivePath := filepath.ToSlash(filepath.Join(remoteSubFolder, "databricks.zip"))

		exists, err := binaryExists(ctx, workspaceFiler, remoteBinaryPath)
		if err != nil {
			return fmt.Errorf("failed to check if file %s exists in workspace: %w", remoteBinaryPath, err)
		}
		if exists {
			log.Infof(ctx, "File %s already exists in the workspace, skipping upload", remoteBinaryPath)
			continue
		}

		log.Infof(ctx, "Uploading %s to the workspace", fileName)
		if err := uploadRelease(ctx, workspaceFiler, getRelease, arch, version, releasesDir, remoteArchivePath); err != nil {
			return err
		}
		log.Infof(ctx, "Successfully uploaded %s to workspace", remoteBinaryPath)
	}

	return nil
}

// binaryExists reports whether the tunnel binary is already present in the workspace,
// retrying the stat on transient errors. The workspace-files get-status endpoint can
// stall (a cold request routinely takes ~60s, right at the SDK's per-request timeout)
// or return a transient 5xx, and a single such failure otherwise aborts the whole connect.
func binaryExists(ctx context.Context, workspaceFiler filer.Filer, remoteBinaryPath string) (bool, error) {
	var exists bool
	err := retries.Wait(ctx, uploadRetryTimeout, func() *retries.Err {
		_, statErr := workspaceFiler.Stat(ctx, remoteBinaryPath)
		switch {
		case statErr == nil:
			exists = true
			return nil
		case errors.Is(statErr, fs.ErrNotExist):
			exists = false
			return nil
		case isRetriableUploadError(ctx, statErr):
			return retries.Continue(statErr)
		default:
			return retries.Halt(statErr)
		}
	})
	return exists, err
}

// uploadRelease uploads a single architecture's release archive, retrying transient
// failures. The reader is re-fetched on each attempt because Write drains it.
func uploadRelease(ctx context.Context, workspaceFiler filer.Filer, getRelease releaseProvider, arch, version, releasesDir, remoteArchivePath string) error {
	return retries.Wait(ctx, uploadRetryTimeout, func() *retries.Err {
		releaseReader, err := getRelease(ctx, arch, version, releasesDir)
		if err != nil {
			return retries.Halt(fmt.Errorf("failed to get archive for architecture %s: %w", arch, err))
		}
		defer releaseReader.Close()

		// workspace-files/import-file API will automatically unzip the payload,
		// producing the filerRoot/remoteSubFolder/*archive-contents* structure, with 'databricks' binary inside.
		err = workspaceFiler.Write(ctx, remoteArchivePath, releaseReader, filer.OverwriteIfExists, filer.CreateParentDirectories)
		switch {
		case err == nil:
			return nil
		case isStreamResetError(err):
			// A stream reset is a proxy body-size rejection, not a transient blip: retrying
			// won't help, so fail with the actionable hint instead of exhausting the budget.
			return retries.Halt(fmt.Errorf("failed to upload file %s to workspace: %w\n\n"+
				"The connection was closed before the upload finished. "+
				"This is usually caused by a network intermediary (corporate egress proxy, VPN, or firewall/WAF) "+
				"enforcing a request-body size limit on POSTs to *.cloud.databricks.com. "+
				"Try running this command from a network without such restrictions",
				remoteArchivePath, err))
		case isRetriableUploadError(ctx, err):
			return retries.Continue(fmt.Errorf("failed to upload file %s to workspace: %w", remoteArchivePath, err))
		default:
			return retries.Halt(fmt.Errorf("failed to upload file %s to workspace: %w", remoteArchivePath, err))
		}
	})
}

// isRetriableUploadError reports whether a workspace-files stat/upload error is worth
// retrying: a transient API error (per the SDK's own classification, e.g. 429/503) or a
// timeout/deadline, which the workspace-files get-status endpoint produces on a cold request.
func isRetriableUploadError(ctx context.Context, err error) bool {
	if apiErr, ok := errors.AsType[*apierr.APIError](err); ok && apiErr.IsRetriable(ctx) {
		return true
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded) {
		return true
	}
	// The SDK surfaces a per-request inactivity timeout as a plain error whose message
	// ends with "request timed out after ...", losing any typed deadline value.
	return strings.Contains(err.Error(), "request timed out after")
}

// isStreamResetError reports whether err looks like an HTTP/2 stream reset from
// the server, which typically means an edge proxy or the workspace-files import
// endpoint rejected the request body (e.g. body-size limit). The string fallback
// catches cases where a transport layer re-formats the http2 error before it
// reaches us, losing the typed value but preserving the message shape.
func isStreamResetError(err error) bool {
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
