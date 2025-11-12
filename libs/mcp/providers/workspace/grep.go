package workspace

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GrepArgs contains arguments for grep operation
type GrepArgs struct {
	Pattern    string `json:"pattern"`
	Path       string `json:"path,omitempty"` // Limit to specific path
	IgnoreCase bool   `json:"ignore_case,omitempty"`
	MaxResults int    `json:"max_results,omitempty"` // Default 100
}

// GrepResult contains the result of a grep operation
type GrepResult struct {
	Matches []GrepMatch `json:"matches"`
	Total   int         `json:"total"`
}

// GrepMatch represents a single grep match
type GrepMatch struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Content string `json:"content"`
}

// Grep searches for a pattern in files within the workspace
func (p *Provider) Grep(ctx context.Context, args *GrepArgs) (*GrepResult, error) {
	workDir, err := p.getWorkDir(ctx)
	if err != nil {
		return nil, err
	}

	// Compile regex
	flags := ""
	if args.IgnoreCase {
		flags = "(?i)"
	}
	re, err := regexp.Compile(flags + args.Pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}

	// Determine search path
	searchPath := workDir
	if args.Path != "" {
		searchPath, err = validatePath(workDir, args.Path)
		if err != nil {
			return nil, err
		}
	}

	// Walk directory
	maxResults := args.MaxResults
	if maxResults == 0 {
		maxResults = 100
	}

	matches := []GrepMatch{}
	err = filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip directories and non-text files
		if info.IsDir() || !isTextFile(path) {
			return nil
		}

		// Read file
		content, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip unreadable files
		}

		// Search lines
		lines := bytes.Split(content, []byte("\n"))
		for i, line := range lines {
			if re.Match(line) {
				relPath, _ := filepath.Rel(workDir, path)
				matches = append(matches, GrepMatch{
					File:    relPath,
					Line:    i + 1,
					Content: string(line),
				})

				if len(matches) >= maxResults {
					return filepath.SkipAll
				}
			}
		}

		return nil
	})

	if err != nil && err != filepath.SkipAll {
		return nil, fmt.Errorf("grep failed: %w", err)
	}

	return &GrepResult{
		Matches: matches,
		Total:   len(matches),
	}, nil
}

// isTextFile checks if a file is likely a text file based on extension
func isTextFile(path string) bool {
	// Simple heuristic: check extension
	ext := strings.ToLower(filepath.Ext(path))
	textExts := []string{
		".txt", ".md", ".go", ".ts", ".js", ".tsx", ".jsx",
		".py", ".rb", ".java", ".c", ".cpp", ".h", ".hpp",
		".json", ".yaml", ".yml", ".toml", ".xml", ".html",
		".css", ".scss", ".sql", ".sh", ".bash",
	}

	for _, textExt := range textExts {
		if ext == textExt {
			return true
		}
	}

	return false
}
