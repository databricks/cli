package main

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTargets(t *testing.T) {
	tests := []struct {
		name    string
		files   []string
		targets []string
	}{
		{
			name: "experimental_ssh",
			files: []string{
				"experimental/ssh/main.go",
				"experimental/ssh/lib/server.go",
			},
			targets: []string{"test-exp-ssh"},
		},
		{
			name: "experimental_aitools",
			files: []string{
				"experimental/aitools/server.go",
			},
			targets: []string{"test-exp-aitools"},
		},
		{
			name: "pipelines",
			files: []string{
				"cmd/pipelines/main.go",
			},
			targets: []string{"test-pipelines"},
		},
		{
			name: "non_matching",
			files: []string{
				"bundle/config.go",
				"cmd/bundle/deploy.go",
			},
			targets: []string{"test"},
		},
		{
			name: "mixed_matching_and_unmatched",
			files: []string{
				"experimental/ssh/main.go",
				"go.mod",
			},
			targets: []string{"test", "test-exp-ssh"},
		},
		{
			name:    "empty_files",
			files:   []string{},
			targets: []string{"test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets := GetTargets(tt.files)
			assert.Equal(t, tt.targets, targets)
		})
	}
}

func TestTargetsExistInMakefile(t *testing.T) {
	// Collect all targets from fileTargetMappings
	expectedTargets := make(map[string]bool)
	for _, mapping := range fileTargetMappings {
		expectedTargets[mapping.target] = true
	}

	// Also include "test" since it's used in GetTargets
	expectedTargets["test"] = true

	// Read and parse Makefile to extract target names
	makefileTargets := parseMakefileTargets(t, "../../Makefile")

	// Verify all expected targets exist in Makefile
	var missingTargets []string
	for target := range expectedTargets {
		if !makefileTargets[target] {
			missingTargets = append(missingTargets, target)
		}
	}

	if len(missingTargets) > 0 {
		t.Errorf("The following targets are defined in targets.go but do not exist in Makefile: %v", missingTargets)
	}
}

// parseMakefileTargets parses a Makefile and returns a set of target names
func parseMakefileTargets(t *testing.T, makefilePath string) map[string]bool {
	targets := make(map[string]bool)
	targetRegex := regexp.MustCompile(`^([a-zA-Z0-9_-]+):`)

	content, err := os.ReadFile(makefilePath)
	require.NoError(t, err)

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		// Match Makefile target pattern: target:
		matches := targetRegex.FindStringSubmatch(line)
		if len(matches) > 1 {
			targets[matches[1]] = true
		}
	}

	return targets
}
