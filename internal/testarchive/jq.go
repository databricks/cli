package testarchive

import (
	"fmt"
	"os"
	"path/filepath"
)

// JqDownloader handles downloading and extracting jq releases
type JqDownloader struct {
	BinDir string
	Arch   string
}

// Download downloads and extracts jq for Linux
func (j JqDownloader) Download() error {
	// create the directory for the download if it doesn't exist
	dir := filepath.Join(j.BinDir, j.Arch)
	err := os.MkdirAll(dir, 0o755)
	if err != nil {
		return err
	}

	// Construct the download URL for the latest release
	url := "https://github.com/jqlang/jq/releases/latest/download/jq-linux-" + j.Arch

	binaryPath := filepath.Join(dir, "jq")
	if err := downloadFile(url, binaryPath); err != nil {
		return err
	}

	// Make the binary executable
	if err := os.Chmod(binaryPath, 0o755); err != nil {
		return fmt.Errorf("failed to make jq executable: %w", err)
	}

	return nil
}
