package main

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// ciTargets lists the Taskfile task names whose `sources:` define the
// trigger set for their corresponding CI job of the same name.
var ciTargets = []string{
	"test-exp-aitools",
	"test-exp-ssh",
	"test-pipelines",
}

// commonTriggerPatterns lists patterns that trigger all test targets.
var commonTriggerPatterns = []string{
	"go.mod",
	"go.sum",
	".github/actions/setup-build-environment/",
	"Taskfile.yml",
	"task",
	"tools/task/",
}

type targetMapping struct {
	prefixes []string
	target   string
}

type taskfile struct {
	Tasks map[string]taskfileTask `yaml:"tasks"`
}

// Sources uses []any to tolerate non-string entries (e.g. `- exclude: tools/**`)
// that appear on other tasks in Taskfile.yml. We only care about string globs;
// map entries are skipped in LoadTargetMappings.
type taskfileTask struct {
	Sources []any `yaml:"sources"`
}

// LoadTargetMappings reads Taskfile.yml and builds target mappings for CI tasks
// by extracting `sources:` from each task listed in ciTargets.
func LoadTargetMappings(taskfilePath string) ([]targetMapping, error) {
	data, err := os.ReadFile(taskfilePath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", taskfilePath, err)
	}
	var tf taskfile
	if err := yaml.Unmarshal(data, &tf); err != nil {
		return nil, fmt.Errorf("parse %s: %w", taskfilePath, err)
	}

	mappings := []targetMapping{
		{prefixes: slices.Clone(commonTriggerPatterns), target: "test"},
	}
	for _, name := range ciTargets {
		t, ok := tf.Tasks[name]
		if !ok {
			return nil, fmt.Errorf("task %q not found in %s", name, taskfilePath)
		}
		if len(t.Sources) == 0 {
			return nil, fmt.Errorf("task %q in %s has no sources", name, taskfilePath)
		}
		prefixes := slices.Clone(commonTriggerPatterns)
		for _, src := range t.Sources {
			s, ok := src.(string)
			if !ok {
				continue
			}
			prefixes = append(prefixes, sourceToPrefix(s))
		}
		mappings = append(mappings, targetMapping{prefixes: prefixes, target: name})
	}
	return mappings, nil
}

// sourceToPrefix converts a Taskfile source glob like "dir/**" into a prefix
// suitable for strings.HasPrefix matching ("dir/").
func sourceToPrefix(src string) string {
	src = strings.TrimSuffix(src, "/**")
	if !strings.HasSuffix(src, "/") {
		src += "/"
	}
	return src
}

// GetTargets matches files to targets based on patterns and returns the matched targets.
func GetTargets(files []string, mappings []targetMapping) []string {
	targetSet := make(map[string]bool)
	unmatchedFiles := []string{}

	for _, file := range files {
		matched := false
		for _, mapping := range mappings {
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

	if len(unmatchedFiles) > 0 {
		targetSet["test"] = true
	}

	if len(targetSet) == 0 {
		return []string{"test"}
	}

	return slices.Sorted(maps.Keys(targetSet))
}
