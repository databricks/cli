package main

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTargets(t *testing.T) {
	mappings, err := LoadTargetMappings("../../Taskfile.yml")
	require.NoError(t, err)

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
			name: "pipelines",
			files: []string{
				"cmd/pipelines/main.go",
			},
			targets: []string{"test-pipelines"},
		},
		{
			name: "acceptance_apps_triggers_aitools",
			files: []string{
				"acceptance/apps/basic/script",
			},
			targets: []string{"test-exp-aitools"},
		},
		{
			name: "sandbox",
			files: []string{
				"cmd/sandbox/create.go",
				"acceptance/cmd/sandbox/create/script",
			},
			targets: []string{"test-sandbox"},
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
			name: "readme_only",
			files: []string{
				"README.md",
			},
			targets: []string{},
		},
		{
			name: "docs_dir_only",
			files: []string{
				"docs/output.md",
				"docs/sync.md",
			},
			targets: []string{},
		},
		{
			name: "nextchanges_fragment_only",
			files: []string{
				".nextchanges/cli/my-feature.md",
			},
			targets: []string{},
		},
		{
			name: "multiple_inert_files",
			files: []string{
				"README.md",
				".nextchanges/bundles/fix.md",
				"LICENSE",
				".github/PULL_REQUEST_TEMPLATE.md",
			},
			targets: []string{},
		},
		{
			name: "inert_plus_code_still_runs",
			files: []string{
				"README.md",
				"bundle/config.go",
			},
			targets: []string{"test"},
		},
		{
			name: "inert_plus_scoped_target_runs_only_that_target",
			files: []string{
				"README.md",
				"experimental/ssh/main.go",
			},
			targets: []string{"test-exp-ssh"},
		},
		{
			// NOTICE is verified by internal/build/notice_test.go, so it is not
			// inert; a change to it must run the test suite.
			name: "notice_is_not_inert",
			files: []string{
				"NOTICE",
			},
			targets: []string{"test"},
		},
		{
			name: "go_mod_triggers_all",
			files: []string{
				"go.mod",
			},
			targets: []string{"test", "test-exp-aitools", "test-exp-ssh", "test-pipelines", "test-sandbox"},
		},
		{
			name: "go_sum_triggers_all",
			files: []string{
				"go.sum",
			},
			targets: []string{"test", "test-exp-aitools", "test-exp-ssh", "test-pipelines", "test-sandbox"},
		},
		{
			name: "go_mod_with_other_files_triggers_all",
			files: []string{
				"experimental/ssh/main.go",
				"go.mod",
			},
			targets: []string{"test", "test-exp-aitools", "test-exp-ssh", "test-pipelines", "test-sandbox"},
		},
		{
			name: "setup_build_environment_triggers_all",
			files: []string{
				".github/actions/setup-build-environment/action.yml",
			},
			targets: []string{"test", "test-exp-aitools", "test-exp-ssh", "test-pipelines", "test-sandbox"},
		},
		{
			name:    "empty_files",
			files:   []string{},
			targets: []string{"test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets := GetTargets(tt.files, mappings)
			assert.Equal(t, tt.targets, targets)
		})
	}
}

func TestLoadTargetMappingsMissingFile(t *testing.T) {
	_, err := LoadTargetMappings("nonexistent.yml")
	assert.Error(t, err)
}

// TestAllInertEncodesAsEmptyArray locks in the wire format the CI skip relies
// on: main.go JSON-encodes the result, and push.yml gates jobs on
// `contains(fromJSON(targets), 'test')`. An all-inert diff must encode as `[]`
// (a valid empty array), not `null`, which fromJSON rejects. This is why
// GetTargets returns a non-nil empty slice rather than nil.
func TestAllInertEncodesAsEmptyArray(t *testing.T) {
	mappings, err := LoadTargetMappings("../../Taskfile.yml")
	require.NoError(t, err)

	targets := GetTargets([]string{"README.md"}, mappings)

	var buf bytes.Buffer
	require.NoError(t, json.NewEncoder(&buf).Encode(targets))
	assert.JSONEq(t, "[]", buf.String())
}

// TestInertPathsExcludeTestInputs guards the correctness invariant of the inert
// list: a file a test actually reads must never be classified as inert, or a
// change to it would wrongly skip the suite. NOTICE is verified by
// internal/build/notice_test.go, and README.md files under acceptance/ and
// libs/template/ are checked-in test fixtures.
func TestInertPathsExcludeTestInputs(t *testing.T) {
	mustRunTests := []string{
		"NOTICE",
		"acceptance/README.md",
		"acceptance/bundle/templates/default-python/classic/output/my_default_python/README.md",
		"libs/template/templates/default-sql/README.md",
		"bundle/config.go",
	}
	for _, file := range mustRunTests {
		assert.False(t, isInertPath(file), "%s must not be inert: a change to it must run tests", file)
	}
}

// TestInertPathsExist guards against typos in the inert list: an entry that does
// not exist in the repository silently stops matching, disabling the skip
// optimization for that file without any test failing. Paths are repo-relative;
// the package runs from tools/testmask, so the repo root is ../../.
func TestInertPathsExist(t *testing.T) {
	const repoRoot = "../../"
	for file := range inertFiles {
		_, err := os.Stat(repoRoot + file)
		assert.NoError(t, err, "inert file %q does not exist in the repo", file)
	}
	for _, prefix := range inertPrefixes {
		info, err := os.Stat(repoRoot + prefix)
		if assert.NoError(t, err, "inert prefix %q does not exist in the repo", prefix) {
			assert.True(t, info.IsDir(), "inert prefix %q is not a directory", prefix)
		}
	}
}
