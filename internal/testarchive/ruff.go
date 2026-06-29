package testarchive

import (
	"fmt"
	"os"
	"path/filepath"
)

// ruffVersion pins the ruff release bundled into the archive. It must match the
// version pinned across the repo (python/pyproject.toml, Taskfile.yml) and the
// minimum required by acceptance/internal/ruff.go, because the check-formatting
// test's golden output assumes that formatter's behavior. Unlike uv and jq
// (which the archive tracks at latest), ruff is pinned for that reason.
const ruffVersion = "0.9.1"

// RuffDownloader handles downloading and extracting ruff releases.
type RuffDownloader struct {
	BinDir string
	Arch   string
}

// mapArchitecture maps our architecture names to ruff's naming convention.
func (r RuffDownloader) mapArchitecture(arch string) (string, error) {
	switch arch {
	case "arm64":
		return "aarch64", nil
	case "amd64":
		return "x86_64", nil
	default:
		return "", fmt.Errorf("unsupported architecture: %s (supported: arm64, amd64)", arch)
	}
}

// Download downloads and extracts ruff for Linux.
func (r RuffDownloader) Download() error {
	ruffArch, err := r.mapArchitecture(r.Arch)
	if err != nil {
		return err
	}

	dir := filepath.Join(r.BinDir, r.Arch)
	err = os.MkdirAll(dir, 0o755)
	if err != nil {
		return err
	}

	// Ruff releases are tagged with the bare version (no "v" prefix).
	// https://github.com/astral-sh/ruff/releases
	ruffTarName := fmt.Sprintf("ruff-%s-unknown-linux-gnu", ruffArch)
	url := fmt.Sprintf("https://github.com/astral-sh/ruff/releases/download/%s/%s.tar.gz", ruffVersion, ruffTarName)

	tempFile := filepath.Join(dir, "ruff.tar.gz")
	if err := downloadFile(url, tempFile); err != nil {
		return err
	}

	if err := ExtractTarGz(tempFile, dir); err != nil {
		return err
	}

	err = os.Remove(tempFile)
	if err != nil {
		return err
	}

	// The ruff binary is extracted into a directory named like
	// ruff-x86_64-unknown-linux-gnu. Move the binary one level up and drop the
	// extra directory to keep the bin layout flat, matching uv.
	err = os.Rename(filepath.Join(dir, ruffTarName, "ruff"), filepath.Join(dir, "ruff"))
	if err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(dir, ruffTarName))
}
