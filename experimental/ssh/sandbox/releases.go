package sandbox

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"

	"github.com/databricks/cli/experimental/ssh/internal/releases"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/databricks-sdk-go"
	sdk "github.com/databricks/databricks-sdk-go/service/sandbox"
)

const maxReleaseSize = 256 << 20

//go:embed cache.py
var executableCacheScript string

// executableUploadReader preserves Seek so SDK retries rewind the upload body.
type executableUploadReader struct {
	*bytes.Reader
}

func (executableUploadReader) Close() error { return nil }

func uploadExecutable(ctx context.Context, w *databricks.WorkspaceClient, name string, setup *remoteSetup, releasesDir string) (string, error) {
	architecture := map[string]string{"x86_64": "amd64", "aarch64": "arm64"}[setup.Architecture]
	if architecture == "" {
		return "", fmt.Errorf("unsupported sandbox architecture %q", setup.Architecture)
	}
	version := build.GetInfo().Version
	cacheID := fmt.Sprintf("%x", sha256.Sum256([]byte(version)))
	var data []byte
	var err error
	if build.GetInfo().IsDevelopment() || releasesDir != "" {
		data, err = readLinuxExecutable(ctx, architecture, releasesDir)
		if err != nil {
			return "", err
		}
		cacheID = fmt.Sprintf("%x", sha256.Sum256(data))
	}
	releaseID := architecture + "-" + cacheID
	destination := path.Join(setup.Home, ".databricks", "sandbox-ssh", "releases", releaseID, "databricks")
	executable := path.Join(setup.SessionDir, releaseID)
	cached, err := executableCacheCommand(ctx, w, name, "check", destination, setup.SessionDir)
	if err != nil {
		return "", err
	}
	if cached {
		return executable, nil
	}
	if data == nil {
		data, err = readLinuxExecutable(ctx, architecture, releasesDir)
		if err != nil {
			return "", err
		}
	}
	temporary := path.Join(setup.SessionDir, "databricks.upload")
	if err := w.Sandbox.UploadFile(ctx, sdk.UploadFileRequest{Parent: name, Path: temporary, Contents: executableUploadReader{bytes.NewReader(data)}}); err != nil {
		return "", fmt.Errorf("upload sandbox SSH executable: %w", err)
	}
	cached, err = executableCacheCommand(ctx, w, name, "install", destination, setup.SessionDir, temporary, fmt.Sprintf("%x", sha256.Sum256(data)))
	if err != nil {
		return "", err
	}
	if !cached {
		return "", errors.New("sandbox SSH executable installation did not complete")
	}
	return executable, nil
}

func readLinuxExecutable(ctx context.Context, architecture, releasesDir string) ([]byte, error) {
	executable, err := linuxExecutable(ctx, architecture, releasesDir)
	if err != nil {
		return nil, err
	}
	defer executable.Close()
	data, err := io.ReadAll(io.LimitReader(executable, maxReleaseSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxReleaseSize {
		return nil, errors.New("linux CLI executable exceeds size limit")
	}
	return data, nil
}

func executableCacheCommand(ctx context.Context, w *databricks.WorkspaceClient, name string, args ...string) (bool, error) {
	result, err := w.Sandbox.ExecuteCommandSync(ctx, sdk.ExecuteCommandSyncRequest{Name: name, Cmd: "python3", Args: append([]string{"-c", executableCacheScript}, args...)})
	if err != nil {
		return false, fmt.Errorf("access sandbox SSH executable cache: %w", err)
	}
	if result.Status != sdk.ExecuteCommandStatusExecuteCommandStatusCompleted || result.ExitCode != 0 || result.Truncated {
		return false, fmt.Errorf("access sandbox SSH executable cache: %s", result.Stderr)
	}
	var response struct {
		Cached bool `json:"cached"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &response); err != nil {
		return false, fmt.Errorf("read sandbox SSH executable cache result: %w", err)
	}
	return response.Cached, nil
}

func linuxExecutable(ctx context.Context, architecture, releasesDir string) (io.ReadCloser, error) {
	if releasesDir == "" && runtime.GOOS == "linux" && runtime.GOARCH == architecture {
		executable, err := os.Executable()
		if err != nil {
			return nil, err
		}
		return os.Open(executable)
	}
	version := build.GetInfo().Version
	if releasesDir == "" && build.GetInfo().IsDevelopment() {
		return nil, errors.New("a development build requires --releases-dir with compatible Linux CLI archives")
	}
	openRelease := releases.OpenPublished
	if releasesDir != "" {
		openRelease = releases.OpenLocal
	}
	archive, err := openRelease(ctx, architecture, version, releasesDir)
	if err != nil {
		return nil, err
	}
	defer archive.Close()
	return executableFromArchive(archive)
}

func executableFromArchive(archive io.Reader) (io.ReadCloser, error) {
	data, err := io.ReadAll(io.LimitReader(archive, maxReleaseSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxReleaseSize {
		return nil, errors.New("linux CLI archive exceeds size limit")
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open Linux CLI archive: %w", err)
	}
	for _, file := range reader.File {
		if file.Name == "databricks" {
			if file.UncompressedSize64 > maxReleaseSize {
				return nil, errors.New("linux CLI executable exceeds size limit")
			}
			return file.Open()
		}
	}
	return nil, errors.New("linux CLI archive does not contain databricks")
}
