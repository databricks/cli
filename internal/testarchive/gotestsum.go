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

// GotestsumDownloader handles downloading and extracting gotestsum releases
type GotestsumDownloader struct {
	BinDir   string
	Arch     string
	RepoRoot string
}

func (g GotestsumDownloader) readVersionFromGoMod() (string, error) {
	goModPath := filepath.Join(g.RepoRoot, "tools", "go.mod")

	file, err := os.Open(goModPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Match: gotest.tools/gotestsum v1.12.1 // indirect
	versionRegex := regexp.MustCompile(`^\s*gotest\.tools/gotestsum\s+v(\d+\.\d+\.\d+)`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		matches := versionRegex.FindStringSubmatch(line)
		if matches != nil {
			return matches[1], nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", errors.New("gotestsum version not found in tools/go.mod")
}

// Download downloads and extracts gotestsum for Linux
func (g GotestsumDownloader) Download() error {
	version, err := g.readVersionFromGoMod()
	if err != nil {
		return fmt.Errorf("failed to read gotestsum version from tools/go.mod: %w", err)
	}

	// Create the directory for the download if it doesn't exist
	dir := filepath.Join(g.BinDir, g.Arch)
	err = os.MkdirAll(dir, 0o755)
	if err != nil {
		return err
	}

	// Construct the download URL
	// Example: https://github.com/gotestyourself/gotestsum/releases/download/v1.12.0/gotestsum_1.12.0_linux_amd64.tar.gz
	fileName := fmt.Sprintf("gotestsum_%s_linux_%s.tar.gz", version, g.Arch)
	url := fmt.Sprintf("https://github.com/gotestyourself/gotestsum/releases/download/v%s/%s", version, fileName)

	tempFile := filepath.Join(dir, fileName)
	err = downloadFile(url, tempFile)
	if err != nil {
		return err
	}

	err = ExtractTarGz(tempFile, dir)
	if err != nil {
		return err
	}

	// Make the binary executable
	binaryPath := filepath.Join(dir, "gotestsum")
	if err := os.Chmod(binaryPath, 0o755); err != nil {
		return fmt.Errorf("failed to make gotestsum executable: %w", err)
	}

	return os.Remove(tempFile)
}
