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

	"github.com/databricks/cli/experimental/ssh/internal/workspace"
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
