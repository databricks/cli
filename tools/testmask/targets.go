package main

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// ciTargetTasks maps the dash-separated output name used by CI to the
// colon-separated Taskfile task name whose `sources:` define its trigger set.
var ciTargetTasks = map[string]string{
	"test-exp-aitools": "test:exp-aitools",
	"test-exp-ssh":     "test:exp-ssh",
	"test-pipelines":   "test:pipelines",
}

// commonTriggerPatterns lists patterns that trigger all test targets.
var commonTriggerPatterns = []string{
	"go.mod",
	"go.sum",
	".github/actions/setup-build-environment/",
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
// by extracting `sources:` from each task listed in ciTargetTasks.
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
	for _, outputName := range slices.Sorted(maps.Keys(ciTargetTasks)) {
		taskName := ciTargetTasks[outputName]
		t, ok := tf.Tasks[taskName]
		if !ok {
			return nil, fmt.Errorf("task %q not found in %s", taskName, taskfilePath)
		}
		if len(t.Sources) == 0 {
			return nil, fmt.Errorf("task %q in %s has no sources", taskName, taskfilePath)
		}
		prefixes := slices.Clone(commonTriggerPatterns)
		for _, src := range t.Sources {
			s, ok := src.(string)
			if !ok {
				continue
			}
			prefixes = append(prefixes, sourceToPrefix(s))
		}
		mappings = append(mappings, targetMapping{prefixes: prefixes, target: outputName})
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
