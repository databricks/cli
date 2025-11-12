// Package local provides a filesystem-based sandbox implementation.
package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/appdotbuild/go-mcp/pkg/fileutil"
	"github.com/appdotbuild/go-mcp/pkg/sandbox"
)

func init() {
	sandbox.Register(sandbox.TypeLocal, func(cfg *sandbox.Config) (sandbox.Sandbox, error) {
		if cfg.BaseDir == "" {
			return nil, fmt.Errorf("base directory required for local sandbox")
		}
		return NewLocalSandbox(cfg.BaseDir)
	})
}

// LocalSandbox implements the Sandbox interface using the local filesystem.
// All operations are restricted to a base directory for security.
type LocalSandbox struct {
	baseDir string       // Base directory for all operations (absolute path)
	workDir string       // Current working directory (relative to baseDir)
	mu      sync.RWMutex // Protects workDir
}

// NewLocalSandbox creates a new LocalSandbox with the specified base directory.
// The base directory is created if it doesn't exist.
func NewLocalSandbox(baseDir string) (*LocalSandbox, error) {
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	if err := os.MkdirAll(absBase, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	resolvedBase, err := filepath.EvalSymlinks(absBase)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve base directory symlinks: %w", err)
	}

	return &LocalSandbox{
		baseDir: resolvedBase,
		workDir: ".",
	}, nil
}

// Exec executes a command in the sandbox.
func (s *LocalSandbox) Exec(ctx context.Context, command string) (*sandbox.ExecResult, error) {
	s.mu.RLock()
	workDir := s.workDir
	s.mu.RUnlock()

	absWorkDir, err := ValidatePath(s.baseDir, workDir)
	if err != nil {
		return nil, fmt.Errorf("invalid working directory: %w", err)
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = absWorkDir

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("failed to execute command: %w", err)
		}
	}

	return &sandbox.ExecResult{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}, nil
}

// WriteFile writes content to a file.
func (s *LocalSandbox) WriteFile(ctx context.Context, path, content string) error {
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute paths not allowed: %s", path)
	}

	s.mu.RLock()
	workDir := s.workDir
	s.mu.RUnlock()

	fullPath := filepath.Join(workDir, path)

	absPath, err := ValidatePath(s.baseDir, fullPath)
	if err != nil {
		return err
	}

	if err := fileutil.AtomicWriteFile(absPath, []byte(content), 0644); err != nil {
		return err
	}

	return nil
}

// WriteFiles writes multiple files atomically (or as close as possible).
func (s *LocalSandbox) WriteFiles(ctx context.Context, files map[string]string) error {
	for path := range files {
		if filepath.IsAbs(path) {
			return fmt.Errorf("absolute paths not allowed: %s", path)
		}
	}

	s.mu.RLock()
	workDir := s.workDir
	s.mu.RUnlock()

	validatedPaths := make(map[string]string, len(files))
	for path := range files {
		fullPath := filepath.Join(workDir, path)
		absPath, err := ValidatePath(s.baseDir, fullPath)
		if err != nil {
			return fmt.Errorf("invalid path %s: %w", path, err)
		}
		validatedPaths[path] = absPath
	}

	for path, content := range files {
		absPath := validatedPaths[path]

		parentDir := filepath.Dir(absPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("failed to create parent directory for %s: %w", path, err)
		}

		if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", path, err)
		}
	}

	return nil
}

// ReadFile reads a file's content.
func (s *LocalSandbox) ReadFile(ctx context.Context, path string) (string, error) {
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("absolute paths not allowed: %s", path)
	}

	s.mu.RLock()
	workDir := s.workDir
	s.mu.RUnlock()

	fullPath := filepath.Join(workDir, path)

	absPath, err := ValidatePath(s.baseDir, fullPath)
	if err != nil {
		return "", err
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(content), nil
}

// DeleteFile deletes a file.
func (s *LocalSandbox) DeleteFile(ctx context.Context, path string) error {
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute paths not allowed: %s", path)
	}

	s.mu.RLock()
	workDir := s.workDir
	s.mu.RUnlock()

	fullPath := filepath.Join(workDir, path)

	absPath, err := ValidatePath(s.baseDir, fullPath)
	if err != nil {
		return err
	}

	if err := os.Remove(absPath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// ListDirectory lists files and directories.
func (s *LocalSandbox) ListDirectory(ctx context.Context, path string) ([]string, error) {
	if filepath.IsAbs(path) {
		return nil, fmt.Errorf("absolute paths not allowed: %s", path)
	}

	s.mu.RLock()
	workDir := s.workDir
	s.mu.RUnlock()

	fullPath := filepath.Join(workDir, path)

	absPath, err := ValidatePath(s.baseDir, fullPath)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	return names, nil
}

// SetWorkdir changes the current working directory.
func (s *LocalSandbox) SetWorkdir(ctx context.Context, path string) error {
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute paths not allowed: %s", path)
	}

	s.mu.RLock()
	workDir := s.workDir
	s.mu.RUnlock()

	fullPath := filepath.Join(workDir, path)

	absPath, err := ValidatePath(s.baseDir, fullPath)
	if err != nil {
		return err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("failed to stat path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}

	relPath, err := filepath.Rel(s.baseDir, absPath)
	if err != nil {
		return fmt.Errorf("failed to compute relative path: %w", err)
	}

	s.mu.Lock()
	s.workDir = relPath
	s.mu.Unlock()

	return nil
}

// ExportDirectory exports a directory from the sandbox to the host.
// For LocalSandbox, this is essentially a copy operation.
func (s *LocalSandbox) ExportDirectory(ctx context.Context, containerPath, hostPath string) (string, error) {
	if filepath.IsAbs(containerPath) {
		return "", fmt.Errorf("absolute paths not allowed: %s", containerPath)
	}

	s.mu.RLock()
	workDir := s.workDir
	s.mu.RUnlock()

	fullPath := filepath.Join(workDir, containerPath)

	absSrcPath, err := ValidatePath(s.baseDir, fullPath)
	if err != nil {
		return "", fmt.Errorf("invalid container path: %w", err)
	}

	absHostPath, err := filepath.Abs(hostPath)
	if err != nil {
		return "", fmt.Errorf("invalid host path: %w", err)
	}

	if err := os.MkdirAll(absHostPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create host directory: %w", err)
	}

	err = copyDir(absSrcPath, absHostPath)
	if err != nil {
		return "", fmt.Errorf("failed to copy directory: %w", err)
	}

	return absHostPath, nil
}

// RefreshFromHost imports files from the host into the sandbox.
func (s *LocalSandbox) RefreshFromHost(ctx context.Context, hostPath, containerPath string) error {
	if filepath.IsAbs(containerPath) {
		return fmt.Errorf("absolute paths not allowed: %s", containerPath)
	}

	absHostPath, err := filepath.Abs(hostPath)
	if err != nil {
		return fmt.Errorf("invalid host path: %w", err)
	}

	s.mu.RLock()
	workDir := s.workDir
	s.mu.RUnlock()

	fullPath := filepath.Join(workDir, containerPath)

	absContainerPath, err := ValidatePath(s.baseDir, fullPath)
	if err != nil {
		return fmt.Errorf("invalid container path: %w", err)
	}

	err = copyDir(absHostPath, absContainerPath)
	if err != nil {
		return fmt.Errorf("failed to copy from host: %w", err)
	}

	return nil
}

// Close releases resources. For LocalSandbox, this is a no-op.
func (s *LocalSandbox) Close() error {
	return nil
}

// copyDir recursively copies a directory.
func copyDir(src, dst string) error {
	// Get source directory info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	// Read source directory
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursively copy subdirectory
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// Copy file
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// copyFile copies a single file.
func copyFile(src, dst string) (err error) {
	// Open source file
	srcFile, openErr := os.Open(src)
	if openErr != nil {
		return openErr
	}
	defer func() {
		if closeErr := srcFile.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	// Get source file info for permissions
	srcInfo, statErr := srcFile.Stat()
	if statErr != nil {
		return statErr
	}

	// Create destination file
	dstFile, createErr := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if createErr != nil {
		return createErr
	}
	defer func() {
		if closeErr := dstFile.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	// Copy contents
	if _, copyErr := io.Copy(dstFile, srcFile); copyErr != nil {
		return copyErr
	}

	return nil
}
