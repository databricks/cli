package workspace

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/databricks/cli/experimental/apps-mcp/lib/fileutil"
)

// ReadFileArgs contains arguments for reading a file
type ReadFileArgs struct {
	FilePath string `json:"file_path"`
	Offset   int    `json:"offset,omitempty"` // Line number to start (1-indexed)
	Limit    int    `json:"limit,omitempty"`  // Number of lines to read
}

// ReadFile reads a file from the workspace
func (p *Provider) ReadFile(ctx context.Context, args *ReadFileArgs) (string, error) {
	workDir, err := p.getWorkDir(ctx)
	if err != nil {
		return "", err
	}

	// Validate path
	fullPath, err := validatePath(workDir, args.FilePath)
	if err != nil {
		return "", err
	}

	// Read file
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Apply line offset and limit if specified
	if args.Offset > 0 || args.Limit > 0 {
		content = applyLineRange(content, args.Offset, args.Limit)
	}

	return string(content), nil
}

// WriteFileArgs contains arguments for writing a file
type WriteFileArgs struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

// WriteFile writes a file to the workspace
func (p *Provider) WriteFile(ctx context.Context, args *WriteFileArgs) error {
	workDir, err := p.getWorkDir(ctx)
	if err != nil {
		return err
	}

	// Validate path
	fullPath, err := validatePath(workDir, args.FilePath)
	if err != nil {
		return err
	}

	if err := fileutil.AtomicWriteFile(fullPath, []byte(args.Content), 0o644); err != nil {
		return err
	}

	return nil
}

// EditFileArgs contains arguments for editing a file
type EditFileArgs struct {
	FilePath  string `json:"file_path"`
	OldString string `json:"old_string"`
	NewString string `json:"new_string"`
}

// EditFile edits a file in the workspace by replacing old_string with new_string
func (p *Provider) EditFile(ctx context.Context, args *EditFileArgs) error {
	workDir, err := p.getWorkDir(ctx)
	if err != nil {
		return err
	}

	// Validate path
	fullPath, err := validatePath(workDir, args.FilePath)
	if err != nil {
		return err
	}

	// Read current content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Check if old_string exists
	contentStr := string(content)
	if !strings.Contains(contentStr, args.OldString) {
		return errors.New("old_string not found in file")
	}

	// Count occurrences
	count := strings.Count(contentStr, args.OldString)
	if count > 1 {
		return fmt.Errorf("old_string appears %d times, must be unique", count)
	}

	// Replace
	newContent := strings.Replace(contentStr, args.OldString, args.NewString, 1)

	// Write back
	if err := os.WriteFile(fullPath, []byte(newContent), 0o644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// applyLineRange applies line offset and limit to file content
func applyLineRange(content []byte, offset, limit int) []byte {
	lines := bytes.Split(content, []byte("\n"))

	// Adjust for 1-indexed
	if offset > 0 {
		offset--
	}

	// Apply offset
	if offset > 0 {
		if offset >= len(lines) {
			return []byte{}
		}
		lines = lines[offset:]
	}

	// Apply limit
	if limit > 0 && limit < len(lines) {
		lines = lines[:limit]
	}

	return bytes.Join(lines, []byte("\n"))
}
