package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/databricks/cli/experimental/ssh/internal/workspace"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"golang.org/x/net/http2"
)

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
	} else if strings.Contains(version, "dev") {
		// Dev/snapshot builds have no GitHub Release to download the server
		// binary from, so fetch it from the branch's GitHub Actions build
		// instead. This lets `ssh connect` work out of the box on a bugbash
		// build without the caller passing --releases-dir.
		cmdio.LogString(ctx, "Snapshot build detected, downloading Linux binaries from GitHub Actions...")
		snapshotDir, cleanup, err := prepareSnapshotReleases(ctx)
		if err != nil {
			return fmt.Errorf("failed to prepare snapshot releases: %w", err)
		}
		defer cleanup()
		releasesDir = snapshotDir
		getRelease = getLocalRelease
	}
	return uploadReleases(ctx, workspaceFiler, getRelease, version, releasesDir)
}

// prepareSnapshotReleases downloads the release-build "cli" artifact for the
// current branch from GitHub Actions and returns the directory containing the
// per-arch zips (databricks_cli_linux_<arch>.zip), plus a cleanup function.
// The artifact already ships those zips, so they can be used as a releases dir
// directly. Requires the GitHub CLI (`gh`).
func prepareSnapshotReleases(ctx context.Context) (string, func(), error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return "", nil, fmt.Errorf("the GitHub CLI (gh) is required to download snapshot builds: %w", err)
	}

	branch := build.GetInfo().Branch
	if branch == "" || branch == "undefined" {
		// Binary not built with GoReleaser (e.g. `go build`); fall back to git.
		out, err := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD").Output()
		if err != nil {
			return "", nil, fmt.Errorf("cannot determine branch: not set at build time and git failed: %w", err)
		}
		branch = strings.TrimSpace(string(out))
	}

	runID, err := findLatestSnapshotRunID(ctx, branch)
	if err != nil {
		return "", nil, err
	}

	tmpDir, err := os.MkdirTemp("", "cli-snapshot-*")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	cleanup := func() { os.RemoveAll(tmpDir) }

	if err := downloadSnapshotArtifact(ctx, runID, tmpDir); err != nil {
		cleanup()
		return "", nil, err
	}
	return tmpDir, cleanup, nil
}

type ghRunEntry struct {
	DatabaseID int    `json:"databaseId"`
	Conclusion string `json:"conclusion"`
}

// findLatestSnapshotRunID finds the most recent successful release-build
// workflow run for the given branch.
func findLatestSnapshotRunID(ctx context.Context, branch string) (string, error) {
	cmd := exec.CommandContext(ctx,
		"gh", "run", "list",
		"-b", branch,
		"-w", "release-build",
		"-R", "databricks/cli",
		"--json", "databaseId,conclusion",
		"--limit", "20",
	)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to list GitHub Actions runs: %w", err)
	}

	var runs []ghRunEntry
	if err := json.Unmarshal(out, &runs); err != nil {
		return "", fmt.Errorf("failed to parse GitHub Actions run list: %w", err)
	}

	for _, r := range runs {
		if r.Conclusion == "success" {
			return strconv.Itoa(r.DatabaseID), nil
		}
	}

	return "", fmt.Errorf("no successful release-build run found for branch %s", branch)
}

// downloadSnapshotArtifact downloads the "cli" artifact (which contains the
// databricks_cli_<os>_<arch>.zip files) from the given GitHub Actions run.
func downloadSnapshotArtifact(ctx context.Context, runID, destDir string) error {
	cmd := exec.CommandContext(ctx,
		"gh", "run", "download", runID,
		"-n", "cli",
		"-D", destDir,
		"-R", "databricks/cli",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to download snapshot artifact: %w\n%s", err, out)
	}
	return nil
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
			if isStreamResetError(err) {
				return fmt.Errorf("failed to upload file %s to workspace: %w\n\n"+
					"The connection was closed before the upload finished. "+
					"This is usually caused by a network intermediary (corporate egress proxy, VPN, or firewall/WAF) "+
					"enforcing a request-body size limit on POSTs to *.cloud.databricks.com. "+
					"Try running this command from a network without such restrictions",
					remoteArchivePath, err)
			}
			return fmt.Errorf("failed to upload file %s to workspace: %w", remoteArchivePath, err)
		}
		log.Infof(ctx, "Successfully uploaded %s to workspace", remoteBinaryPath)
	}

	return nil
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
