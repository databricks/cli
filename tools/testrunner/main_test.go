package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigRuleMatches(t *testing.T) {
	tests := []struct {
		input       string
		packageName string
		testcase    string
		match       bool
	}{
		// Exact matches
		{"bundle TestDeploy", "bundle", "TestDeploy", true},
		{"bundle TestDeploy", "libs", "TestDeploy", false},
		{"bundle TestDeploy", "bundle", "TestSomethingElse", false},

		// Package prefix matches
		{"libs/ TestSomething", "libs/auth", "TestSomething", true},
		{"libs/ TestSomething", "libs", "TestSomething", true},
		{"libs/ TestSomething", "libsother", "TestSomething", false},

		// Test prefix matches
		{"bundle TestAccept/", "bundle", "TestAcceptDeploy", false},
		{"bundle TestAccept/", "bundle", "TestAccept", true},
		{"bundle TestAccept/", "bundle", "TestAccept/Deploy", true},

		// Wildcard matches
		{"* *", "any/package", "AnyTest", true},
		{"* TestAccept/", "any/package", "TestAcceptDeploy", false},
		{"* TestAccept/", "any/package", "TestAccept/Deploy", true},
		{"libs/ *", "libs/auth", "AnyTest", true},

		// Path prefix edge cases
		{"TestAccept/ TestAccept/", "TestAccept", "TestAccept", true},                    //nolint:dupword
		{"TestAccept/ TestAccept/", "TestAccept/bundle", "TestAccept/deploy", true},      //nolint:dupword
		{"TestAccept/ TestAccept/", "TestAcceptSomething", "TestAcceptSomething", false}, //nolint:dupword

		// Empty values cases
		{"* TestDeploy", "", "TestDeploy", true},
		{"bundle *", "bundle", "", true},

		// A rule does not match a parent of the listed test; the parent's
		// cascading failure is handled separately (see TestCheckFailures).
		{"acceptance TestAccept/bundle/templates/default-python/combinations/classic", "acceptance", "TestAccept", false},
		{"acceptance TestAccept/bundle/templates/default-python/combinations/classic", "acceptance", "TestAnother", false},
		{"acceptance TestAccept/bundle/templates/default-python/combinations/classic", "acceptance", "TestAccept/bundle/templates/default-python/combinations/classic/x", false},

		// pattern version
		{"acceptance TestAccept/bundle/templates/default-python/combinations/classic/", "acceptance", "TestAccept", false},
		{"acceptance TestAccept/bundle/templates/default-python/combinations/classic/", "acceptance", "TestAnother", false},
		{"acceptance TestAccept/bundle/templates/default-python/combinations/classic/", "acceptance", "TestAccept/bundle/templates/default-python/combinations/classic/x", true},
		{"acceptance TestAccept/bundle/templates/default-python/combinations/classic/", "acceptance", "TestAccept/bundle/templates/default-python/combinations/classic", true},
		{"acceptance TestAccept/bundle/templates/default-python/combinations/classic/", "acceptance", "TestAccept/bundle/templates/default-python/combinations", false},

		// Interior "*" spans any number of path segments. This is the DMS=true
		// use case: match every permutation that carries a DMS=true segment,
		// regardless of the test name or engine segments preceding it.
		{"acceptance TestAccept/*/DMS=true/", "acceptance", "TestAccept/bundle/invariant/no_drift/DATABRICKS_BUNDLE_ENGINE=direct/DMS=true/INPUT_CONFIG=job.yml.tmpl", true},
		{"acceptance TestAccept/*/DMS=true/", "acceptance", "TestAccept/bundle/invariant/migrate/DATABRICKS_BUNDLE_ENGINE=terraform/DMS=true", true},
		{"acceptance TestAccept/*/DMS=true/", "acceptance", "TestAccept/bundle/invariant/no_drift/DATABRICKS_BUNDLE_ENGINE=direct/DMS=/INPUT_CONFIG=job.yml.tmpl", false},
		{"acceptance TestAccept/*/DMS=true/", "acceptance", "TestAccept", false},

		// Interior "*" matching zero segments.
		{"* TestAccept/*/leaf", "any", "TestAccept/leaf", true},
		{"* a/*/b", "any", "a/b", true},
		{"* a/*/b", "any", "a/x/y/b", true},
		{"* a/*/b", "any", "a/x/b/c", false},

		// Multiple interior wildcards.
		{"* a/*/b/*/c", "any", "a/1/b/2/c", true},
		{"* a/*/b/*/c", "any", "a/1/2/b/3/4/c", true},
		{"* a/*/b/*/c", "any", "a/b/c", true},
	}

	for _, tt := range tests {
		t.Run(tt.input+"_"+tt.packageName+"_"+tt.testcase, func(t *testing.T) {
			rule, err := parseConfigRule(tt.input, tt.input)
			require.NoError(t, err)
			result := rule.matches(strings.Split(tt.packageName, "/"), strings.Split(tt.testcase, "/"))
			assert.Equal(t, tt.match, result)
		})
	}
}

func TestParseConfigRuleErrors(t *testing.T) {
	for _, input := range []string{
		"bundle",
		"a b c",
		"bundle// TestDeploy",
		"bundle /",
		"* TestAccept//Deploy",
	} {
		t.Run(input, func(t *testing.T) {
			_, err := parseConfigRule(input, input)
			assert.Error(t, err)
		})
	}
}

func TestCheckFailures(t *testing.T) {
	const config = "* TestAccept/ssh/connection\n"

	tests := []struct {
		name     string
		results  []TestResult
		wantExit int
	}{
		{
			// A parent failing because a listed subtest failed is allowed.
			name: "parent allowed when subtest failed",
			results: []TestResult{
				{Action: "fail", Package: "acceptance", Test: "TestAccept/ssh/connection"},
				{Action: "fail", Package: "acceptance", Test: "TestAccept"},
			},
			wantExit: 0,
		},
		{
			// A parent failing on its own (no subtest failed) is not allowed,
			// even though a subtest is listed as a known failure. This is the
			// ruff-missing-in-setup case.
			name: "parent not allowed without failing subtest",
			results: []TestResult{
				{Action: "fail", Package: "acceptance", Test: "TestAccept"},
			},
			wantExit: 7,
		},
		{
			name: "unlisted failure is not allowed",
			results: []TestResult{
				{Action: "fail", Package: "acceptance", Test: "TestSomethingElse"},
			},
			wantExit: 7,
		},
		{
			// gotestsum reruns failed tests; a failure followed by a pass is
			// flaky, not a failure.
			name: "failure passing on retry is allowed",
			results: []TestResult{
				{Action: "fail", Package: "acceptance", Test: "TestAccept"},
				{Action: "pass", Package: "acceptance", Test: "TestAccept"},
			},
			wantExit: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := parseConfig(config)
			require.NoError(t, err)

			jsonFile := filepath.Join(t.TempDir(), "results.json")
			var sb strings.Builder
			for _, r := range tt.results {
				line, err := json.Marshal(r)
				require.NoError(t, err)
				sb.Write(line)
				sb.WriteByte('\n')
			}
			require.NoError(t, os.WriteFile(jsonFile, []byte(sb.String()), 0o600))

			assert.Equal(t, tt.wantExit, checkFailures(cfg, jsonFile, 7))
		})
	}
}
