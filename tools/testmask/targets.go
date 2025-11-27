package main

import (
	"sort"
	"strings"
)

type targetMapping struct {
	patterns []string
	target   string
}

var fileTargetMappings = []targetMapping{
	{
		patterns: []string{
			"experimental/aitools/",
		},
		target: "test-exp-aitools",
	},
	{
		patterns: []string{
			"experimental/apps-mcp/",
		},
		target: "test-exp-apps-mcp",
	},
	{
		patterns: []string{
			"experimental/ssh/",
			"acceptance/ssh/",
		},
		target: "test-exp-ssh",
	},
	{
		patterns: []string{
			"cmd/pipelines/",
			"acceptance/pipelines/",
		},
		target: "test-pipelines",
	},
}

// GetTargets matches files to targets based on patterns and returns the matched targets.
func GetTargets(files []string) []string {
	targetSet := make(map[string]bool)
	unmatchedFiles := []string{}

	for _, file := range files {
		matched := false
		for _, mapping := range fileTargetMappings {
			for _, pattern := range mapping.patterns {
				if strings.HasPrefix(file, pattern) {
					targetSet[mapping.target] = true
					matched = true
					break
				}
			}
			if matched {
				break
			}
		}
		if !matched {
			unmatchedFiles = append(unmatchedFiles, file)
		}
	}

	// If there are unmatched files, add the "test" target to run all tests.
	if len(unmatchedFiles) > 0 {
		targetSet["test"] = true
	}

	// If there are no targets, add the "test" target to run all tests.
	if len(targetSet) == 0 {
		return []string{"test"}
	}

	// Convert map to sorted slice
	targets := make([]string, 0, len(targetSet))
	for target := range targetSet {
		targets = append(targets, target)
	}

	// Sort for consistent output
	sort.Strings(targets)
	return targets
}
