package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// JqDownloader handles downloading and extracting jq releases
type JqDownloader struct {
	downloadDir string
}

// NewJqDownloader creates a new jq downloader
// TODO: Use _cache instead of testdata. directories with _ are ignored by go.
// TODO: Mount the binaries to an appropriate directory in the archive. Instead of top level.
// Perhaps _bin is a much better name for this? Or no, keep cache and hte same paths.
func NewJqDownloader() *JqDownloader {
	return &JqDownloader{
		downloadDir: "./testdata",
	}
}

// mapArchitecture maps our architecture names to jq's naming convention
func (j *JqDownloader) mapArchitecture(arch string) (string, error) {
	switch arch {
	case "arm64":
		return "arm64", nil
	case "amd64":
		return "amd64", nil
	default:
		return "", fmt.Errorf("unsupported architecture: %s (supported: arm64, amd64)", arch)
	}
}

// Download downloads and extracts jq for Linux
func (j *JqDownloader) Download(arch string) error {
	// Map architecture names to jq's naming convention
	jqArch, err := j.mapArchitecture(arch)
	if err != nil {
		return err
	}

	downloadDir := filepath.Join(j.downloadDir, jqArch)

	// Construct the download URL for the latest release
	url := fmt.Sprintf("https://github.com/jqlang/jq/releases/latest/download/jq-linux-%s", jqArch)

	// Create bin directory using shared utility
	if err := ensureBinDir(downloadDir); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Create target filename
	targetFile := filepath.Join(downloadDir, "jq")

	fmt.Printf("Downloading jq for Linux %s...\n", arch)

	// Download the binary using shared utility
	if err := downloadFile(url, targetFile); err != nil {
		return err
	}

	// Get file size for confirmation
	fileInfo, err := os.Stat(targetFile)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	fmt.Printf("Downloaded %s (%.2f MB)\n", targetFile, float64(fileInfo.Size())/1024/1024)

	// Make the binary executable
	if err := os.Chmod(targetFile, 0o755); err != nil {
		return fmt.Errorf("failed to make jq executable: %w", err)
	}

	fmt.Printf("✅ Successfully downloaded jq for Linux %s\n", arch)
	fmt.Printf("📁 Downloaded to: %s\n", targetFile)
	fmt.Printf("🚀 Add to PATH: export PATH=$PWD/%s:$PATH\n", downloadDir)

	return nil
}
