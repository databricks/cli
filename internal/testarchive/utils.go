package testarchive

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Verbose controls whether detailed progress is printed.
// Set to true for detailed output, false for quiet operation.
var Verbose = true

// logf prints a message only if Verbose is true.
func logf(format string, args ...any) {
	if Verbose {
		fmt.Printf(format, args...)
	}
}

// getCacheDir returns the cache directory for downloads.
// It uses ~/.cache/testarchive by default.
func getCacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	cacheDir := filepath.Join(homeDir, ".cache", "testarchive")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}
	return cacheDir, nil
}

// getCacheKey returns a stable cache key for a URL.
func getCacheKey(url string) string {
	hash := sha256.Sum256([]byte(url))
	return fmt.Sprintf("%x", hash[:8])
}

func downloadFile(url, outputPath string) error {
	// Check if we have this file cached
	cacheDir, err := getCacheDir()
	if err == nil {
		cacheKey := getCacheKey(url)
		ext := filepath.Ext(outputPath)
		if ext == "" {
			ext = ".bin"
		}
		cachedFile := filepath.Join(cacheDir, cacheKey+ext)
		if _, err := os.Stat(cachedFile); err == nil {
			// Cache hit - copy from cache
			logf("Using cached file for %s\n", url)
			return copyFile(cachedFile, outputPath)
		}
	}

	// Download the file
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	logf("Downloading %s to %s\n", url, outputPath)
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	// Cache the downloaded file for future use
	if cacheDir != "" {
		cacheKey := getCacheKey(url)
		ext := filepath.Ext(outputPath)
		if ext == "" {
			ext = ".bin"
		}
		cachedFile := filepath.Join(cacheDir, cacheKey+ext)
		if err := copyFile(outputPath, cachedFile); err != nil {
			logf("Warning: failed to cache file: %v\n", err)
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// ExtractTarGz extracts a tar.gz file to the specified directory
func ExtractTarGz(archivePath, destDir string) error {
	logf("Extracting %s to %s\n", archivePath, destDir)

	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		targetPath := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			outFile, err := os.Create(targetPath)
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", targetPath, err)
			}

			_, err = io.Copy(outFile, tarReader)
			outFile.Close()
			if err != nil {
				return fmt.Errorf("failed to extract file %s: %w", targetPath, err)
			}

			// Set file permissions
			if err := os.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
				logf("Warning: failed to set permissions for %s: %v\n", targetPath, err)
			}
		}
	}

	return nil
}
