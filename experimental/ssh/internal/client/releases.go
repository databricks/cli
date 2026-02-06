package client

import (
	"archive/zip"
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
	"github.com/databricks/databricks-sdk-go"
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
		// For snapshot/dev builds, download binaries from GitHub Actions artifacts
		// because there is no GitHub Release for dev versions.
		cmdio.LogString(ctx, "Snapshot build detected, downloading Linux binaries from GitHub Actions...")
		tmpDir, cleanup, err := prepareSnapshotReleases(ctx)
		if err != nil {
			return fmt.Errorf("failed to prepare snapshot releases: %w", err)
		}
		defer cleanup()
		releasesDir = tmpDir
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

		_, err := workspaceFiler.Stat(ctx, remoteBinaryPath)
		if err == nil {
			cmdio.LogString(ctx, fmt.Sprintf("File %s already exists in the workspace, skipping upload", remoteBinaryPath))
			continue
		} else if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("failed to check if file %s exists in workspace: %w", remoteBinaryPath, err)
		}

		releaseReader, err := getRelease(ctx, arch, version, releasesDir)
		if err != nil {
			return fmt.Errorf("failed to get archive for architecture %s: %w", arch, err)
		}
		defer releaseReader.Close()

		cmdio.LogString(ctx, fmt.Sprintf("Uploading %s to the workspace", fileName))
		// workspace-files/import-file API will automatically unzip the payload,
		// producing the filerRoot/remoteSubFolder/*archive-contents* structure, with 'databricks' binary inside.
		err = workspaceFiler.Write(ctx, remoteArchivePath, releaseReader, filer.OverwriteIfExists, filer.CreateParentDirectories)
		if err != nil {
			return fmt.Errorf("failed to upload file %s to workspace: %w", remoteArchivePath, err)
		}
		cmdio.LogString(ctx, fmt.Sprintf("Successfully uploaded %s to workspace", remoteBinaryPath))
	}

	return nil
}

func getReleaseName(architecture, version string) string {
	if strings.Contains(version, "dev") {
		return fmt.Sprintf("databricks_cli_linux_%s.zip", architecture)
	}
	return fmt.Sprintf("databricks_cli_%s_linux_%s.zip", version, architecture)
}

func getLocalRelease(ctx context.Context, architecture, version, releasesDir string) (io.ReadCloser, error) {
	cmdio.LogString(ctx, "Looking for CLI releases in directory: "+releasesDir)
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
	cmdio.LogString(ctx, fmt.Sprintf("Downloading %s from %s", fileName, downloadURL))

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

// prepareSnapshotReleases downloads Linux CLI binaries from GitHub Actions
// artifacts and packages them as zip files for upload. It returns the directory
// containing the zip files and a cleanup function.
func prepareSnapshotReleases(ctx context.Context) (string, func(), error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return "", nil, fmt.Errorf("the GitHub CLI (gh) is required to download snapshot builds: %w", err)
	}

	branch := build.GetInfo().Branch
	if branch == "undefined" {
		// Binary not built with GoReleaser; fall back to git.
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

	downloadDir := filepath.Join(tmpDir, "download")
	if err := downloadSnapshotArtifact(ctx, runID, downloadDir); err != nil {
		cleanup()
		return "", nil, err
	}

	for _, arch := range []string{"amd64", "arm64"} {
		pattern := filepath.Join(downloadDir, fmt.Sprintf("cli_linux_%s*", arch), "databricks")
		matches, err := filepath.Glob(pattern)
		if err != nil {
			cleanup()
			return "", nil, fmt.Errorf("failed to glob for %s binary: %w", arch, err)
		}
		if len(matches) == 0 {
			cleanup()
			return "", nil, fmt.Errorf("no binary found for linux/%s in downloaded artifacts (pattern: %s)", arch, pattern)
		}

		zipPath := filepath.Join(tmpDir, fmt.Sprintf("databricks_cli_linux_%s.zip", arch))
		if err := createZipFromBinary(matches[0], zipPath); err != nil {
			cleanup()
			return "", nil, fmt.Errorf("failed to create zip for %s: %w", arch, err)
		}
		cmdio.LogString(ctx, "Prepared snapshot zip for linux/"+arch)
	}

	return tmpDir, cleanup, nil
}

type ghRunEntry struct {
	DatabaseID int    `json:"databaseId"`
	Conclusion string `json:"conclusion"`
}

// findLatestSnapshotRunID finds the most recent successful release-snapshot
// workflow run for the given branch.
func findLatestSnapshotRunID(ctx context.Context, branch string) (string, error) {
	cmd := exec.CommandContext(ctx,
		"gh", "run", "list",
		"-b", branch,
		"-w", "release-snapshot",
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

	return "", fmt.Errorf("no successful release-snapshot run found for branch %s", branch)
}

// downloadSnapshotArtifact downloads the cli_linux_snapshot artifact from the
// given GitHub Actions run.
func downloadSnapshotArtifact(ctx context.Context, runID, destDir string) error {
	cmd := exec.CommandContext(ctx,
		"gh", "run", "download", runID,
		"-n", "cli_linux_snapshot",
		"-D", destDir,
		"-R", "databricks/cli",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to download snapshot artifact: %w\n%s", err, out)
	}
	return nil
}

// createZipFromBinary creates a zip file at zipPath containing a single
// "databricks" entry with the contents of binaryPath and 0755 permissions.
func createZipFromBinary(binaryPath, zipPath string) error {
	binaryData, err := os.ReadFile(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to read binary %s: %w", binaryPath, err)
	}

	zipFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("failed to create zip file %s: %w", zipPath, err)
	}
	defer zipFile.Close()

	zw := zip.NewWriter(zipFile)
	defer zw.Close()

	header := &zip.FileHeader{
		Name:   "databricks",
		Method: zip.Deflate,
	}
	header.SetMode(0o755)

	w, err := zw.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("failed to create zip entry: %w", err)
	}

	if _, err := w.Write(binaryData); err != nil {
		return fmt.Errorf("failed to write binary to zip: %w", err)
	}

	return nil
}
