package libraries

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

// libraryLocalPaths returns the patterns from the configuration root of the
// local path fields that hold libraries, which may be outside the sync root.
func libraryLocalPaths() map[string]bool {
	out := make(map[string]bool)
	for key, declarer := range config.ResourceTypesImplementing[resources.LocalPathDeclarer]() {
		base := dyn.NewPattern(dyn.Key("resources"), dyn.Key(key), dyn.AnyKey())
		for _, field := range declarer.LocalPaths() {
			if field.AllowOutsideSyncRoot {
				out[base.Append(field.Pattern...).String()] = true
			}
		}
	}
	return out
}

func patternStrings(patterns []dyn.Pattern) []string {
	var out []string
	for _, p := range patterns {
		out = append(out, p.String())
	}
	return out
}

// TestLibraryLocations pins every declared library location, its attributes,
// and whether its values are also declared as local paths, which makes them
// relative to the sync root before the library mutators see them. A change to a
// declaration shows up here in review.
func TestLibraryLocations(t *testing.T) {
	type declaration struct {
		pattern           string
		kind              resources.LibraryKind
		skipPatchedWheels bool
		localPath         bool
	}

	const (
		list = resources.LibraryList
		deps = resources.DependencyList
		file = resources.LibraryFile
	)

	expected := []declaration{
		{"resources.jobs.*.tasks[*].libraries", list, false, true},
		{"resources.jobs.*.tasks[*].for_each_task.task.libraries", list, false, true},
		{"resources.jobs.*.environments[*].spec.dependencies", deps, false, true},
		{"resources.jobs.*.tasks[*].ai_runtime_task.code_source_path", file, false, false},
		{"resources.jobs.*.tasks[*].for_each_task.task.ai_runtime_task.code_source_path", file, false, false},
		{"resources.pipelines.*.environment.dependencies", deps, true, true},
		{"resources.pipelines.*.testing.environment.dependencies", deps, false, true},
		{"resources.clusters.*.libraries", list, false, false},
	}

	localPaths := libraryLocalPaths()
	var actual []declaration
	for _, l := range locations() {
		values := patternStrings(l.valuePatterns())
		declared := 0
		for _, v := range values {
			if localPaths[v] {
				declared++
			}
		}
		if declared != 0 && declared != len(values) {
			t.Errorf("only some of %v are declared as local paths", values)
		}
		actual = append(actual, declaration{
			pattern:           l.pattern.String(),
			kind:              l.Kind,
			skipPatchedWheels: l.SkipPatchedWheels,
			localPath:         declared > 0,
		})
	}

	assert.Equal(t, expected, actual)
}

// TestLibraryLocalPathsAreLibraryLocations checks that every local path field
// that holds libraries is in a declared library location, so that the libraries
// it holds are uploaded.
func TestLibraryLocalPathsAreLibraryLocations(t *testing.T) {
	values := make(map[string]bool)
	for _, l := range locations() {
		for _, p := range l.valuePatterns() {
			values[p.String()] = true
		}
	}

	for p := range libraryLocalPaths() {
		assert.True(t, values[p], "local path %s holds libraries but is not in a library location", p)
	}
}

// TestExpandGlobReferencesOrder pins the order in which glob references are
// expanded, which is the order of the diagnostics.
func TestExpandGlobReferencesOrder(t *testing.T) {
	var actual []dyn.Pattern
	for _, e := range expanders() {
		actual = append(actual, e.pattern)
	}

	assert.Equal(t, []string{
		"resources.jobs.*.tasks[*].libraries",
		"resources.jobs.*.tasks[*].for_each_task.task.libraries",
		"resources.jobs.*.environments[*].spec.dependencies",
		"resources.pipelines.*.environment.dependencies",
		"resources.pipelines.*.testing.environment.dependencies",
		"resources.clusters.*.libraries",
	}, patternStrings(actual))
}
