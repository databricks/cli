package state

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var (
	// directories to scan for source files
	sourceDirs = []string{"client", "server"}

	// directories to exclude
	excludeDirs = map[string]bool{
		"node_modules": true,
		"dist":         true,
		".git":         true,
		"build":        true,
		"coverage":     true,
	}

	// file extensions to include
	sourceExtensions = map[string]bool{
		".ts":   true,
		".tsx":  true,
		".js":   true,
		".jsx":  true,
		".json": true,
		".css":  true,
		".html": true,
		".yaml": true,
		".yml":  true,
	}
)

// ComputeChecksum computes SHA256 checksum of all source files
func ComputeChecksum(workDir string) (string, error) {
	var files []string

	// collect files from source directories
	for _, dir := range sourceDirs {
		dirPath := filepath.Join(workDir, dir)
		if _, err := os.Stat(dirPath); err == nil {
			if err := collectSourceFiles(dirPath, &files); err != nil {
				return "", err
			}
		}
	}

	// include root package.json
	packageJSON := filepath.Join(workDir, "package.json")
	if _, err := os.Stat(packageJSON); err == nil {
		files = append(files, packageJSON)
	}

	// sort for deterministic order
	slices.Sort(files)

	if len(files) == 0 {
		return "", errors.New("no source files found - project structure appears invalid")
	}

	// compute combined hash
	hasher := sha256.New()
	for _, file := range files {
		if err := hashFile(hasher, file); err != nil {
			return "", fmt.Errorf("failed to hash %s: %w", file, err)
		}
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// VerifyChecksum verifies that current checksum matches expected
func VerifyChecksum(workDir, expected string) (bool, error) {
	current, err := ComputeChecksum(workDir)
	if err != nil {
		return false, err
	}
	return current == expected, nil
}

func collectSourceFiles(dir string, files *[]string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())

		if entry.IsDir() {
			if excludeDirs[entry.Name()] {
				continue
			}
			if err := collectSourceFiles(path, files); err != nil {
				return err
			}
		} else {
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if sourceExtensions[ext] {
				*files = append(*files, path)
			}
		}
	}

	return nil
}

func hashFile(hasher io.Writer, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(hasher, f)
	return err
}
