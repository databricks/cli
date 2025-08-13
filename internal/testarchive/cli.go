package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// createGitArchive creates a tar.gz archive of all git-tracked files plus downloaded tools
func createGitArchive(outputPath string) error {
	repoRoot := filepath.Join("..", "..")
	testdataDir := "./testdata"

	// Download Go for both architectures
	goDownloader := NewGoDownloader()
	if err := goDownloader.Download("amd64"); err != nil {
		return fmt.Errorf("failed to download Go amd64: %w", err)
	}
	if err := goDownloader.Download("arm64"); err != nil {
		return fmt.Errorf("failed to download Go arm64: %w", err)
	}

	// Download UV for both architectures
	uvDownloader := NewUVDownloader()
	if err := uvDownloader.Download("amd64"); err != nil {
		return fmt.Errorf("failed to download UV amd64: %w", err)
	}
	if err := uvDownloader.Download("arm64"); err != nil {
		return fmt.Errorf("failed to download UV arm64: %w", err)
	}

	// Download jq for both architectures
	jqDownloader := NewJqDownloader()
	if err := jqDownloader.Download("amd64"); err != nil {
		return fmt.Errorf("failed to download jq amd64: %w", err)
	}
	if err := jqDownloader.Download("arm64"); err != nil {
		return fmt.Errorf("failed to download jq arm64: %w", err)
	}

	// Get list of git-tracked files
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get git tracked files: %w", err)
	}

	// Parse git-tracked files
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	var gitFiles []string
	for scanner.Scan() {
		file := strings.TrimSpace(scanner.Text())
		if file != "" {
			gitFiles = append(gitFiles, file)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading git ls-files output: %w", err)
	}

	// Get downloaded tools if they exist
	var downloadedFiles []string
	if _, err := os.Stat(testdataDir); err == nil {
		err := filepath.WalkDir(testdataDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// Skip the testdata directory itself
			if path == testdataDir {
				return nil
			}

			// Get relative path from testdata directory
			relPath, err := filepath.Rel(testdataDir, path)
			if err != nil {
				return err
			}

			// Add with testdata/ prefix to maintain structure in archive
			downloadedFiles = append(downloadedFiles, filepath.Join("testdata", relPath))
			return nil
		})
		if err != nil {
			fmt.Printf("Warning: failed to scan testdata directory: %v\n", err)
		}
	}

	totalFiles := len(gitFiles) + len(downloadedFiles)
	fmt.Printf("Found %d git-tracked files and %d downloaded files (%d total)\n",
		len(gitFiles), len(downloadedFiles), totalFiles)

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create the tar.gz file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	fmt.Printf("Creating archive %s...\n", outputPath)

	processedCount := 0

	// Add git-tracked files to the archive
	for _, file := range gitFiles {
		if processedCount%1000 == 0 {
			fmt.Printf("Progress: %d/%d Git files processed\n", processedCount, totalFiles)
		}

		fullPath := filepath.Join(repoRoot, file)

		if err := addFileToArchive(tarWriter, fullPath, file); err != nil {
			fmt.Printf("Warning: failed to add git file %s: %v\n", file, err)
		}
		processedCount++
	}

	// Add downloaded files to the archive
	for _, file := range downloadedFiles {
		if processedCount%10000 == 0 {
			fmt.Printf("Progress: %d/%d downloaded files processed\n", processedCount, totalFiles)
		}

		// Remove "testdata/" prefix to get actual file path
		actualPath := strings.TrimPrefix(file, "testdata/")
		fullPath := filepath.Join(testdataDir, actualPath)

		if err := addFileToArchive(tarWriter, fullPath, file); err != nil {
			fmt.Printf("Warning: failed to add downloaded file %s: %v\n", file, err)
		}
		processedCount++
	}

	fmt.Printf("✅ Successfully created comprehensive archive: %s\n", outputPath)
	fmt.Printf("📁 Archive contains %d files (%d git-tracked + %d downloaded)\n",
		totalFiles, len(gitFiles), len(downloadedFiles))
	fmt.Printf("🔧 Includes: Go (amd64 + arm64), UV (amd64 + arm64), jq (amd64 + arm64), and all source code\n")

	// Show archive size
	if stat, err := outFile.Stat(); err == nil {
		size := float64(stat.Size()) / (1024 * 1024)
		fmt.Printf("📦 Archive size: %.1f MB\n", size)
	}

	return nil
}

// addFileToArchive adds a single file to the tar archive
func addFileToArchive(tarWriter *tar.Writer, fullPath, archivePath string) error {
	// Get file info
	info, err := os.Stat(fullPath)
	if err != nil {
		return fmt.Errorf("failed to stat: %w", err)
	}

	// Skip directories and non-regular files
	if !info.Mode().IsRegular() {
		return nil
	}

	// Create tar header
	header := &tar.Header{
		Name:    archivePath, // Use the archive path (preserves structure)
		Size:    info.Size(),
		Mode:    int64(info.Mode()),
		ModTime: info.ModTime(),
	}

	// Write header
	if err := tarWriter.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Copy file content
	fileReader, err := os.Open(fullPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer fileReader.Close()

	_, err = io.Copy(tarWriter, fileReader)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	return nil
}
