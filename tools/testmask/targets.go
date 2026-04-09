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

// commonTriggerPatterns lists patterns that trigger all test targets.
var commonTriggerPatterns = []string{
	"go.mod",
	"go.sum",
	".github/actions/setup-build-environment/",
}

var fileTargetMappings = []targetMapping{
	{
		prefixes: slices.Concat(commonTriggerPatterns, []string{
			// Specify files that match targets below and should still trigger the "test" target.
		}),
		target: "test",
	},
	{
		prefixes: slices.Concat(commonTriggerPatterns, []string{
			"experimental/aitools/",
		}),
		target: "test-exp-aitools",
	},
	{
		prefixes: slices.Concat(commonTriggerPatterns, []string{
			"experimental/ssh/",
			"acceptance/ssh/",
		}),
		target: "test-exp-ssh",
	},
	{
		prefixes: slices.Concat(commonTriggerPatterns, []string{
			"cmd/pipelines/",
			"acceptance/pipelines/",
		}),
		target: "test-pipelines",
	},
}

// GetTargets matches files to targets based on patterns and returns the matched targets.
func GetTargets(files []string) []string {
	targetSet := make(map[string]bool)
	unmatchedFiles := []string{}

	for _, file := range files {
		// Check all mappings for this file (a file can match multiple targets).
		matched := false
		for _, mapping := range fileTargetMappings {
			for _, prefix := range mapping.prefixes {
				if strings.HasPrefix(file, prefix) {
					targetSet[mapping.target] = true
					matched = true
					break
				}
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
