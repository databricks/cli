package validation

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const StateFileName = ".databricks_app_state"

// AppState represents the validation state of a Databricks App.
type AppState string

const (
	StateValidated AppState = "validated"
	StateDeployed  AppState = "deployed"
)

// State represents the validation state file contents.
type State struct {
	State       AppState  `json:"state"`
	ValidatedAt time.Time `json:"validated_at"`
	Checksum    string    `json:"checksum"`
}

// excludedPatterns contains patterns to exclude from checksum computation.
var excludedPatterns = []string{
	// Git
	".git/",
	// Node.js
	"node_modules/",
	".next/",
	"dist/",
	"build/",
	"coverage/",
	".cache/",
	// Python
	"__pycache__/",
	".venv/",
	"venv/",
	".env/",
	"env/",
	".pytest_cache/",
	".mypy_cache/",
	".ruff_cache/",
	"htmlcov/",
	// Editor/OS
	".DS_Store",
	".idea/",
	".vscode/",
	// Temp files
	"*.log",
	"*.tmp",
	"*.temp",
	"*.swp",
	"*.swo",
	// Databricks
	StateFileName,
	".databricks/",
}

// excludedExtensions contains file extensions to exclude.
var excludedExtensions = []string{
	".pyc",
	".pyo",
}

// excludedSuffixes contains directory suffixes to exclude.
var excludedSuffixes = []string{
	".egg-info/",
}

// shouldExclude checks if a path should be excluded from checksum.
func shouldExclude(relPath string) bool {
	// Normalize path separators
	normalizedPath := filepath.ToSlash(relPath)

	for _, pattern := range excludedPatterns {
		// Directory patterns (ending with /)
		if strings.HasSuffix(pattern, "/") {
			dir := strings.TrimSuffix(pattern, "/")
			if normalizedPath == dir || strings.HasPrefix(normalizedPath, dir+"/") {
				return true
			}
		} else if strings.HasPrefix(pattern, "*.") {
			// Extension patterns
			ext := strings.TrimPrefix(pattern, "*")
			if strings.HasSuffix(normalizedPath, ext) {
				return true
			}
		} else {
			// Exact match
			if normalizedPath == pattern || strings.HasSuffix(normalizedPath, "/"+pattern) {
				return true
			}
		}
	}

	for _, ext := range excludedExtensions {
		if strings.HasSuffix(normalizedPath, ext) {
			return true
		}
	}

	for _, suffix := range excludedSuffixes {
		trimmed := strings.TrimSuffix(suffix, "/")
		if strings.HasSuffix(normalizedPath, trimmed) || strings.Contains(normalizedPath, trimmed+"/") {
			return true
		}
	}

	return false
}

// ComputeChecksum computes a SHA256 checksum of all relevant files in workDir.
func ComputeChecksum(workDir string) (string, error) {
	var files []string

	err := filepath.WalkDir(workDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(workDir, path)
		if err != nil {
			return err
		}

		// Skip root
		if relPath == "." {
			return nil
		}

		if shouldExclude(relPath) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !d.IsDir() {
			files = append(files, relPath)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to walk directory: %w", err)
	}

	// Sort for deterministic ordering
	sort.Strings(files)

	h := sha256.New()
	for _, relPath := range files {
		fullPath := filepath.Join(workDir, relPath)

		// Include filename in hash
		h.Write([]byte(relPath))

		f, err := os.Open(fullPath)
		if err != nil {
			return "", fmt.Errorf("failed to open %s: %w", relPath, err)
		}

		_, err = io.Copy(h, f)
		f.Close()
		if err != nil {
			return "", fmt.Errorf("failed to read %s: %w", relPath, err)
		}
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// LoadState reads the state file from workDir.
func LoadState(workDir string) (*State, error) {
	statePath := filepath.Join(workDir, StateFileName)

	data, err := os.ReadFile(statePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file (file may be corrupted, delete %s to reset): %w", statePath, err)
	}

	return &state, nil
}

// SaveState writes the state file to workDir atomically.
func SaveState(workDir string, state *State) error {
	statePath := filepath.Join(workDir, StateFileName)

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to temp file first for atomicity
	tmpPath := statePath + ".tmp"
	// Clean up any leftover temp file from previous failed attempt
	os.Remove(tmpPath)
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write temp state file: %w", err)
	}

	if err := os.Rename(tmpPath, statePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename state file: %w", err)
	}

	return nil
}
