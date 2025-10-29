package testarchive

import (
	"fmt"
	"os"
	"path/filepath"
)

// UvDownloader handles downloading and extracting UV releases
type UvDownloader struct {
	BinDir string
	Arch   string
}

// uvDownloader creates a new UV downloader

// mapArchitecture maps our architecture names to UV's naming convention
func (u UvDownloader) mapArchitecture(arch string) (string, error) {
	switch arch {
	case "arm64":
		return "aarch64", nil
	case "amd64":
		return "x86_64", nil
	default:
		return "", fmt.Errorf("unsupported architecture: %s (supported: arm64, amd64)", arch)
	}
}

// Download downloads and extracts UV for Linux
func (u UvDownloader) Download() error {
	// Map architecture names to UV's naming convention
	uvArch, err := u.mapArchitecture(u.Arch)
	if err != nil {
		return err
	}

	// create the directory for the download if it doesn't exist
	dir := filepath.Join(u.BinDir, u.Arch)
	err = os.MkdirAll(dir, 0o755)
	if err != nil {
		return err
	}

	// Construct the download URL for the latest release
	uvTarName := fmt.Sprintf("uv-%s-unknown-linux-gnu", uvArch)
	url := fmt.Sprintf("https://github.com/astral-sh/uv/releases/latest/download/%s.tar.gz", uvTarName)

	// Download the file using shared utility
	tempFile := filepath.Join(dir, "uv.tar.gz")
	if err := downloadFile(url, tempFile); err != nil {
		return err
	}

	// Extract the archive to the directory
	if err := ExtractTarGz(tempFile, dir); err != nil {
		return err
	}

	err = os.Remove(tempFile)
	if err != nil {
		return err
	}

	// The uv binary is extracted into a directory called something like
	// uv-aarch64-unknown-linux-gnu. We remove that additional directory here
	// and move the uv binary one level above to keep the directory structure clean.
	err = os.Rename(filepath.Join(dir, uvTarName, "uv"), filepath.Join(dir, "uv"))
	if err != nil {
		return err
	}
	err = os.RemoveAll(filepath.Join(dir, uvTarName))
	if err != nil {
		return err
	}
	return nil
}
