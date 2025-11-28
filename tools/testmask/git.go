package main

import (
	"bufio"
	"bytes"
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

	return parseLines(output), nil
}

// parseLines parses command output into a slice of non-empty lines.
func parseLines(output []byte) []string {
	var lines []string
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
