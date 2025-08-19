package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// jqDownloader handles downloading and extracting jq releases
type jqDownloader struct {
	binDir string
	arch   string
}

// Download downloads and extracts jq for Linux
func (j jqDownloader) Download() error {
	// create the directory for the download if it doesn't exist
	dir := filepath.Join(j.binDir, j.arch)
	err := os.MkdirAll(dir, 0o755)
	if err != nil {
		return err
	}

	// Construct the download URL for the latest release
	url := fmt.Sprintf("https://github.com/jqlang/jq/releases/latest/download/jq-linux-%s", j.arch)

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
