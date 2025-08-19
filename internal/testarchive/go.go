package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GoVersion represents a Go release version from the API
// TODO: Add caching for go / uv or libraries so that they are not redownloaded constantly.
type GoVersion struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
	Files   []struct {
		Filename string `json:"filename"`
		OS       string `json:"os"`
		Arch     string `json:"arch"`
		Version  string `json:"version"`
		SHA256   string `json:"sha256"`
		Size     int    `json:"size"`
		Kind     string `json:"kind"`
	} `json:"files"`
}

// GoDownloader handles downloading and extracting Go releases
type GoDownloader struct {
	downloadDir string
}

// NewGoDownloader creates a new Go downloader
func NewGoDownloader() *GoDownloader {
	return &GoDownloader{
		downloadDir: "./_downloads",
	}
}

// readGoVersionFromMod reads the Go version from go.mod file
// TODO: Should I also include the go dependencies in the archive?
// It'll make the runs much faster.
func (g *GoDownloader) readGoVersionFromMod() (string, error) {
	goModPath := filepath.Join("..", "..", "go.mod")

	file, err := os.Open(goModPath)
	if err != nil {
		return "", fmt.Errorf("failed to open go.mod: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	goVersionRegex := regexp.MustCompile(`^go\s+(\d+\.\d+(?:\.\d+)?)`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if matches := goVersionRegex.FindStringSubmatch(line); matches != nil {
			version := matches[1]
			if !strings.Contains(version, ".") {
				return "", fmt.Errorf("invalid go version format: %s", version)
			}
			return version, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading go.mod: %w", err)
	}

	return "", fmt.Errorf("go version not found in go.mod")
}

// fetchGoVersions gets available Go versions from the official API
func (g *GoDownloader) fetchGoVersions() ([]GoVersion, error) {
	resp, err := http.Get("https://go.dev/dl/?mode=json")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Go versions: %w", err)
	}
	defer resp.Body.Close()

	var versions []GoVersion
	if err := json.NewDecoder(resp.Body).Decode(&versions); err != nil {
		return nil, fmt.Errorf("failed to parse Go versions: %w", err)
	}

	return versions, nil
}

// findCompatibleVersion finds a compatible Go version
func (g *GoDownloader) findCompatibleVersion(versions []GoVersion, targetVersion string) (*GoVersion, error) {
	possibleVersions := []string{
		fmt.Sprintf("go%s", targetVersion),                           // go1.24.0
		fmt.Sprintf("go%s", strings.TrimSuffix(targetVersion, ".0")), // go1.24
	}

	// First try exact match
	for i := range versions {
		for _, possibleVersion := range possibleVersions {
			if versions[i].Version == possibleVersion {
				return &versions[i], nil
			}
		}
	}

	// If no exact match, try finding the latest version in the same major.minor series
	majorMinor := strings.TrimSuffix(targetVersion, ".0")
	if !strings.Contains(majorMinor, ".") {
		parts := strings.Split(targetVersion, ".")
		if len(parts) >= 2 {
			majorMinor = parts[0] + "." + parts[1]
		}
	}

	for i := range versions {
		if strings.HasPrefix(versions[i].Version, "go"+majorMinor+".") && versions[i].Stable {
			return &versions[i], nil
		}
	}

	return nil, fmt.Errorf("Go version matching %s not found in available releases", targetVersion)
}

// Download downloads and extracts Go for Linux
func (g *GoDownloader) Download(arch string) error {
	// Validate architecture
	if arch != "amd64" && arch != "arm64" {
		return fmt.Errorf("unsupported architecture: %s (use 'amd64' or 'arm64')", arch)
	}

	// Read Go version from go.mod
	goVersion, err := g.readGoVersionFromMod()
	if err != nil {
		return fmt.Errorf("failed to read Go version from go.mod: %w", err)
	}

	// Fetch available versions
	versions, err := g.fetchGoVersions()
	if err != nil {
		return err
	}

	// Find compatible version
	targetVersion, err := g.findCompatibleVersion(versions, goVersion)
	if err != nil {
		return err
	}

	// Find the Linux tarball for the specified architecture
	var targetFile *struct {
		Filename string `json:"filename"`
		OS       string `json:"os"`
		Arch     string `json:"arch"`
		Version  string `json:"version"`
		SHA256   string `json:"sha256"`
		Size     int    `json:"size"`
		Kind     string `json:"kind"`
	}

	for i := range targetVersion.Files {
		file := &targetVersion.Files[i]
		if file.OS == "linux" && file.Arch == arch && file.Kind == "archive" {
			targetFile = file
			break
		}
	}

	if targetFile == nil {
		return fmt.Errorf("no Linux %s archive found for Go %s", arch, targetVersion.Version)
	}

	downloadDir := filepath.Join(g.downloadDir, arch)

	// Create bin directory using shared utility
	if err := ensureBinDir(downloadDir); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Download the file using shared utility
	downloadURL := fmt.Sprintf("https://go.dev/dl/%s", targetFile.Filename)
	tempFile := filepath.Join(downloadDir, targetFile.Filename)

	if err := downloadFile(downloadURL, tempFile); err != nil {
		return err
	}

	// Extract the archive using shared utility
	if err := extractTarGz(tempFile, downloadDir); err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}

	// Remove the downloaded archive using shared utility
	cleanupTempFile(tempFile)
	return nil
}
