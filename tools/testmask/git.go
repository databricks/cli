package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetChangedFiles returns the list of files changed between two git refs.
func GetChangedFiles(headRef, baseRef string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", baseRef, headRef)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get diff between %s and %s: %w", baseRef, headRef, err)
	}

	lines := strings.Split(string(output), "\n")

	// Drop the last line (always empty)
	if len(lines) > 0 {
		lines = lines[:len(lines)-1]
	}

	return lines, nil
}
