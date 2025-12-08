package main

import (
	"maps"
	"slices"
	"strings"
)

type targetMapping struct {
	prefixes []string
	target   string
}

var fileTargetMappings = []targetMapping{
	{
		prefixes: []string{
			"experimental/aitools/",
		},
		target: "test-exp-aitools",
	},
	{
		prefixes: []string{
			"experimental/apps-mcp/",
		},
		target: "test-exp-apps-mcp",
	},
	{
		prefixes: []string{
			"experimental/ssh/",
			"acceptance/ssh/",
		},
		target: "test-exp-ssh",
	},
	{
		prefixes: []string{
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
			for _, prefix := range mapping.prefixes {
				if strings.HasPrefix(file, prefix) {
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

	return slices.Sorted(maps.Keys(targetSet))
}
