package sandbox

import (
	"archive/zip"
	"bytes"
	"context"
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

func uploadExecutable(ctx context.Context, w *databricks.WorkspaceClient, name string, setup *remoteSetup, releasesDir string) (string, error) {
	architecture := map[string]string{"x86_64": "amd64", "aarch64": "arm64"}[setup.Architecture]
	if architecture == "" {
		return "", fmt.Errorf("unsupported sandbox architecture %q", setup.Architecture)
	}
	executable, err := linuxExecutable(ctx, architecture, releasesDir)
	if err != nil {
		return "", err
	}
	defer executable.Close()
	destination := path.Join(setup.SessionDir, "databricks")
	if err := w.Sandbox.UploadFile(ctx, sdk.UploadFileRequest{Parent: name, Path: destination, Contents: executable}); err != nil {
		return "", fmt.Errorf("upload sandbox SSH executable: %w", err)
	}
	result, err := w.Sandbox.ExecuteCommandSync(ctx, sdk.ExecuteCommandSyncRequest{Name: name, Cmd: "chmod", Args: []string{"0700", destination}})
	if err != nil {
		return "", err
	}
	if result.Status != sdk.ExecuteCommandStatusExecuteCommandStatusCompleted || result.ExitCode != 0 {
		return "", fmt.Errorf("make sandbox SSH executable runnable: %s", result.Stderr)
	}
	return destination, nil
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
