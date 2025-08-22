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

// gitFiles returns a list of all git-tracked files in the repository.
func gitFiles(repoRoot string) ([]string, error) {
	cmd := exec.Command("git", "ls-files")
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get git tracked files: %w", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	var gitFiles []string
	for scanner.Scan() {
		file := strings.TrimSpace(scanner.Text())
		if file != "" {
			gitFiles = append(gitFiles, file)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading git ls-files output: %w", err)
	}

	return gitFiles, nil
}

func binFiles(binDir string) ([]string, error) {
	var binFiles []string

	err := filepath.WalkDir(binDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(binDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		binFiles = append(binFiles, relPath)
		return nil
	})

	return binFiles, err
}

// addFileToArchive adds a single file to the tar archive
func addFileToArchive(tarWriter *tar.Writer, src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat: %w", err)
	}

	// Skip directories and non-regular files
	if !info.Mode().IsRegular() {
		return nil
	}

	header := &tar.Header{
		Name:    dst,
		Size:    info.Size(),
		Mode:    int64(info.Mode()),
		ModTime: info.ModTime(),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	fileReader, err := os.Open(src)
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

// createArchive creates a tar.gz archive of all git-tracked files plus downloaded tools
func createArchive(archiveDir, binDir, repoRoot string) error {
	archivePath := filepath.Join(archiveDir, "archive.tar.gz")

	// Download tools for both arm and amd64 architectures.
	// The right architecture to use is decided at runtime on the serverless driver.
	// The Databricks platform explicitly does not provide any guarantees around
	// the CPU architecture to keep the door open for future optimizations.
	downloaders := []downloader{
		goDownloader{arch: "amd64", binDir: binDir},
		goDownloader{arch: "arm64", binDir: binDir},
		uvDownloader{arch: "amd64", binDir: binDir},
		uvDownloader{arch: "arm64", binDir: binDir},
		jqDownloader{arch: "amd64", binDir: binDir},
		jqDownloader{arch: "arm64", binDir: binDir},
	}

	for _, downloader := range downloaders {
		err := downloader.Download()
		if err != nil {
			return fmt.Errorf("failed to download %s: %w", downloader, err)
		}
	}

	gitFiles, err := gitFiles(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to get git tracked files: %w", err)
	}

	binFiles, err := binFiles(binDir)
	if err != nil {
		return fmt.Errorf("failed to get bin files: %w", err)
	}

	totalFiles := len(gitFiles) + len(binFiles)
	fmt.Printf("Found %d git-tracked files and %d downloaded files (%d total)\n",
		len(gitFiles), len(binFiles), totalFiles)

	// Create archive directory if it doesn't exist
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create the tar.gz file
	outFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Create gzip writer
	gzWriter := gzip.NewWriter(outFile)
	defer gzWriter.Close()

	// Create tar.gz writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	fmt.Printf("Creating archive %s...\n", archivePath)

	// Add git-tracked files to the archive
	for _, file := range gitFiles {
		err := addFileToArchive(tarWriter, filepath.Join(repoRoot, file), filepath.Join("cli", file))
		if err != nil {
			fmt.Printf("Warning: failed to add git file %s: %v\n", file, err)
		}
	}

	// Add downloaded files / binaries to the archive
	for _, file := range binFiles {
		err := addFileToArchive(tarWriter, filepath.Join(binDir, file), filepath.Join("bin", file))
		if err != nil {
			fmt.Printf("Warning: failed to add downloaded file %s: %v\n", file, err)
		}
	}

	stat, err := outFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat archive: %w", err)
	}

	fmt.Printf("✅ Successfully created comprehensive archive. Archive size: %.1f MB\n", float64(stat.Size())/(1024*1024))
	return nil
}
