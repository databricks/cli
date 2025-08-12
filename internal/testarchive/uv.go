package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// UVDownloader handles downloading and extracting UV releases
type UVDownloader struct {
	downloadDir string
}

// NewUVDownloader creates a new UV downloader
func NewUVDownloader() *UVDownloader {
	return &UVDownloader{
		downloadDir: "./testdata",
	}
}

// mapArchitecture maps our architecture names to UV's naming convention
func (u *UVDownloader) mapArchitecture(arch string) (string, error) {
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
func (u *UVDownloader) Download(arch string) error {
	// Map architecture names to UV's naming convention
	uvArch, err := u.mapArchitecture(arch)
	if err != nil {
		return err
	}

	downloadDir := filepath.Join(u.downloadDir, uvArch)

	// Construct the download URL for the latest release
	url := fmt.Sprintf("https://github.com/astral-sh/uv/releases/latest/download/uv-%s-unknown-linux-gnu.tar.gz", uvArch)

	// Create bin directory using shared utility
	if err := ensureBinDir(downloadDir); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Create temporary filename
	tempFile := filepath.Join(downloadDir, "uv.tar.gz")

	fmt.Printf("Downloading uv for Linux %s...\n", arch)

	// Download the file using shared utility
	if err := downloadFile(url, tempFile); err != nil {
		return err
	}

	// Get file size for confirmation
	fileInfo, err := os.Stat(tempFile)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	fmt.Printf("Downloaded %s (%.2f MB)\n", tempFile, float64(fileInfo.Size())/1024/1024)

	// Extract the archive using shared utility
	if err := extractTarGz(tempFile, downloadDir); err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}

	// Remove the downloaded archive using shared utility
	cleanupTempFile(tempFile)

	fmt.Printf("✅ Successfully downloaded and extracted uv for Linux %s\n", arch)
	fmt.Printf("📁 Extracted to: %s/\n", downloadDir)
	fmt.Printf("🚀 Add to PATH: export PATH=$PWD/%s:$PATH\n", downloadDir)

	return nil
}
