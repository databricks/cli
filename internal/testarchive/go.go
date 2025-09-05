package testarchive

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type downloader interface {
	Download() error
}

type GoDownloader struct {
	BinDir   string
	Arch     string
	RepoRoot string
}

func (g GoDownloader) readGoVersionFromMod() (string, error) {
	goModPath := filepath.Join(g.RepoRoot, "go.mod")

	file, err := os.Open(goModPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Get regex version from toolchain go version specified. Eg: toolchain go1.24.6
	goVersionRegex := regexp.MustCompile(`^toolchain go(\d+\.\d+\.\d+)$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		matches := goVersionRegex.FindStringSubmatch(line)
		if matches != nil {
			return matches[1], nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", errors.New("go version not found in go.mod")
}

// Download downloads and extracts Go for Linux
func (g GoDownloader) Download() error {
	goVersion, err := g.readGoVersionFromMod()
	if err != nil {
		return fmt.Errorf("failed to read Go version from go.mod: %w", err)
	}

	// Create the directory for the download if it doesn't exist
	dir := filepath.Join(g.BinDir, g.Arch)
	err = os.MkdirAll(dir, 0o755)
	if err != nil {
		return err
	}

	// Download the tar archive.
	fileName := fmt.Sprintf("go%s.linux-%s.tar.gz", goVersion, g.Arch)
	url := "https://go.dev/dl/" + fileName

	tempFile := filepath.Join(dir, fileName)
	err = downloadFile(url, tempFile)
	if err != nil {
		return err
	}

	err = ExtractTarGz(tempFile, dir)
	if err != nil {
		return err
	}

	return os.Remove(tempFile)
}
