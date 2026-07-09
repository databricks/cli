package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// testDirsFixture is the set of known acceptance test dirs used across the
// parser tests. Values mirror real acceptance/ layout: some plain bundle tests
// and several bundle/invariant subdirs that share the configs/ directory.
func testDirsFixture() map[string]bool {
	return map[string]bool{
		"bundle/resources":              true,
		"bundle/resources/jobs":         true,
		"bundle/invariant/no_drift":     true,
		"bundle/invariant/migrate":      true,
		"bundle/invariant/continue_293": true,
		"cmd/version":                   true,
	}
}

func TestParseChangedTests(t *testing.T) {
	tests := []struct {
		name string
		diff string
		want map[string]ChangedTest
	}{
		{
			name: "empty diff",
			diff: "",
			want: map[string]ChangedTest{},
		},
		{
			name: "modified file maps to owning test dir",
			diff: "M\tacceptance/bundle/resources/script",
			want: map[string]ChangedTest{
				"bundle/resources": {Added: false, VariantFilters: nil},
			},
		},
		{
			name: "added script marks the dir as added",
			diff: "A\tacceptance/bundle/resources/jobs/script",
			want: map[string]ChangedTest{
				"bundle/resources/jobs": {Added: true, VariantFilters: nil},
			},
		},
		{
			name: "added dir stays added when another file in it is also changed",
			diff: "A\tacceptance/bundle/resources/jobs/script\nM\tacceptance/bundle/resources/jobs/test.toml",
			want: map[string]ChangedTest{
				"bundle/resources/jobs": {Added: true, VariantFilters: nil},
			},
		},
		{
			name: "added dir stays added regardless of diff line order",
			diff: "M\tacceptance/bundle/resources/jobs/test.toml\nA\tacceptance/bundle/resources/jobs/script",
			want: map[string]ChangedTest{
				"bundle/resources/jobs": {Added: true, VariantFilters: nil},
			},
		},
		{
			name: "nested file maps to innermost test dir",
			diff: "M\tacceptance/bundle/resources/jobs/output.txt",
			want: map[string]ChangedTest{
				"bundle/resources/jobs": {Added: false, VariantFilters: nil},
			},
		},
		{
			name: "file outside acceptance is ignored",
			diff: "M\tlibs/dyn/value.go",
			want: map[string]ChangedTest{},
		},
		{
			name: "file under acceptance but not in any test dir is ignored",
			diff: "M\tacceptance/README.md",
			want: map[string]ChangedTest{},
		},
		{
			name: "rename destination is used and is not treated as added",
			diff: "R092\tacceptance/bundle/resources/old\tacceptance/bundle/resources/script",
			want: map[string]ChangedTest{
				"bundle/resources": {Added: false, VariantFilters: nil},
			},
		},
		{
			name: "deleted file in a still-existing test dir re-enables that dir",
			diff: "D\tacceptance/bundle/resources/output.txt",
			want: map[string]ChangedTest{
				"bundle/resources": {Added: false, VariantFilters: nil},
			},
		},
		{
			name: "parent script.prepare re-enables all descendant test dirs",
			diff: "M\tacceptance/bundle/resources/script.prepare",
			want: map[string]ChangedTest{
				"bundle/resources":      {Added: false, VariantFilters: nil},
				"bundle/resources/jobs": {Added: false, VariantFilters: nil},
			},
		},
		{
			name: "parent script.cleanup re-enables all descendant test dirs",
			diff: "M\tacceptance/bundle/resources/script.cleanup",
			want: map[string]ChangedTest{
				"bundle/resources":      {Added: false, VariantFilters: nil},
				"bundle/resources/jobs": {Added: false, VariantFilters: nil},
			},
		},
		{
			name: "parent test.toml re-enables all descendant test dirs",
			diff: "M\tacceptance/bundle/invariant/test.toml",
			want: map[string]ChangedTest{
				"bundle/invariant/no_drift":     {Added: false, VariantFilters: nil},
				"bundle/invariant/migrate":      {Added: false, VariantFilters: nil},
				"bundle/invariant/continue_293": {Added: false, VariantFilters: nil},
			},
		},
		{
			name: "root script.prepare re-enables every test dir",
			diff: "M\tacceptance/script.prepare",
			want: map[string]ChangedTest{
				"bundle/resources":              {Added: false, VariantFilters: nil},
				"bundle/resources/jobs":         {Added: false, VariantFilters: nil},
				"bundle/invariant/no_drift":     {Added: false, VariantFilters: nil},
				"bundle/invariant/migrate":      {Added: false, VariantFilters: nil},
				"bundle/invariant/continue_293": {Added: false, VariantFilters: nil},
				"cmd/version":                   {Added: false, VariantFilters: nil},
			},
		},
		{
			name: "test.toml directly in a test dir re-enables only that dir",
			diff: "M\tacceptance/bundle/resources/jobs/test.toml",
			want: map[string]ChangedTest{
				"bundle/resources/jobs": {Added: false, VariantFilters: nil},
			},
		},
		{
			name: "parent shared file preserves an added descendant",
			diff: "A\tacceptance/bundle/resources/jobs/script\nM\tacceptance/bundle/resources/test.toml",
			want: map[string]ChangedTest{
				"bundle/resources":      {Added: false, VariantFilters: nil},
				"bundle/resources/jobs": {Added: true, VariantFilters: nil},
			},
		},
		{
			name: "a bin helper re-enables every test dir",
			diff: "M\tacceptance/bin/print_requests.py",
			want: map[string]ChangedTest{
				"bundle/resources":              {Added: false, VariantFilters: nil},
				"bundle/resources/jobs":         {Added: false, VariantFilters: nil},
				"bundle/invariant/no_drift":     {Added: false, VariantFilters: nil},
				"bundle/invariant/migrate":      {Added: false, VariantFilters: nil},
				"bundle/invariant/continue_293": {Added: false, VariantFilters: nil},
				"cmd/version":                   {Added: false, VariantFilters: nil},
			},
		},
		{
			name: "a shared fixture in a non-test dir re-enables its subtree",
			diff: "M\tacceptance/bundle/invariant/_script",
			want: map[string]ChangedTest{
				"bundle/invariant/no_drift":     {Added: false, VariantFilters: nil},
				"bundle/invariant/migrate":      {Added: false, VariantFilters: nil},
				"bundle/invariant/continue_293": {Added: false, VariantFilters: nil},
			},
		},
		{
			name: "a stray file at the acceptance root is ignored",
			diff: "M\tacceptance/README.md",
			want: map[string]ChangedTest{},
		},
		{
			name: "deleted script of a removed test dir is ignored (dir no longer in testDirs)",
			diff: "D\tacceptance/bundle/removed_test/script",
			want: map[string]ChangedTest{},
		},
		{
			name: "deleted invariant config is ignored",
			diff: "D\tacceptance/bundle/invariant/configs/job.yml.tmpl",
			want: map[string]ChangedTest{},
		},
		{
			name: "deleted config alongside a modified config keeps only the modified filter",
			diff: "M\tacceptance/bundle/invariant/configs/job.yml.tmpl\nD\tacceptance/bundle/invariant/configs/job_with_permissions.yml.tmpl",
			want: map[string]ChangedTest{
				"bundle/invariant/no_drift":     {Added: false, VariantFilters: []string{"INPUT_CONFIG=job.yml.tmpl"}},
				"bundle/invariant/migrate":      {Added: false, VariantFilters: []string{"INPUT_CONFIG=job.yml.tmpl"}},
				"bundle/invariant/continue_293": {Added: false, VariantFilters: []string{"INPUT_CONFIG=job.yml.tmpl"}},
			},
		},
		{
			name: "changed invariant config re-enables all invariant subdirs with INPUT_CONFIG filter",
			diff: "M\tacceptance/bundle/invariant/configs/job.yml.tmpl",
			want: map[string]ChangedTest{
				"bundle/invariant/no_drift":     {Added: false, VariantFilters: []string{"INPUT_CONFIG=job.yml.tmpl"}},
				"bundle/invariant/migrate":      {Added: false, VariantFilters: []string{"INPUT_CONFIG=job.yml.tmpl"}},
				"bundle/invariant/continue_293": {Added: false, VariantFilters: []string{"INPUT_CONFIG=job.yml.tmpl"}},
			},
		},
		{
			name: "invariant config init.sh strips suffix to base config name",
			diff: "M\tacceptance/bundle/invariant/configs/job.yml.tmpl-init.sh",
			want: map[string]ChangedTest{
				"bundle/invariant/no_drift":     {Added: false, VariantFilters: []string{"INPUT_CONFIG=job.yml.tmpl"}},
				"bundle/invariant/migrate":      {Added: false, VariantFilters: []string{"INPUT_CONFIG=job.yml.tmpl"}},
				"bundle/invariant/continue_293": {Added: false, VariantFilters: []string{"INPUT_CONFIG=job.yml.tmpl"}},
			},
		},
		{
			name: "direct change to an invariant subdir overrides config-scoped filter with all-variants",
			diff: "M\tacceptance/bundle/invariant/configs/job.yml.tmpl\nM\tacceptance/bundle/invariant/no_drift/script",
			want: map[string]ChangedTest{
				"bundle/invariant/no_drift":     {Added: false, VariantFilters: nil},
				"bundle/invariant/migrate":      {Added: false, VariantFilters: []string{"INPUT_CONFIG=job.yml.tmpl"}},
				"bundle/invariant/continue_293": {Added: false, VariantFilters: []string{"INPUT_CONFIG=job.yml.tmpl"}},
			},
		},
		{
			name: "added invariant subdir overrides config-scoped filter and stays added",
			diff: "M\tacceptance/bundle/invariant/configs/job.yml.tmpl\nA\tacceptance/bundle/invariant/no_drift/script",
			want: map[string]ChangedTest{
				"bundle/invariant/no_drift":     {Added: true, VariantFilters: nil},
				"bundle/invariant/migrate":      {Added: false, VariantFilters: []string{"INPUT_CONFIG=job.yml.tmpl"}},
				"bundle/invariant/continue_293": {Added: false, VariantFilters: []string{"INPUT_CONFIG=job.yml.tmpl"}},
			},
		},
		{
			name: "two changed invariant configs accumulate filters",
			diff: "M\tacceptance/bundle/invariant/configs/job.yml.tmpl\nM\tacceptance/bundle/invariant/configs/pipeline.yml.tmpl",
			want: map[string]ChangedTest{
				"bundle/invariant/no_drift":     {Added: false, VariantFilters: []string{"INPUT_CONFIG=job.yml.tmpl", "INPUT_CONFIG=pipeline.yml.tmpl"}},
				"bundle/invariant/migrate":      {Added: false, VariantFilters: []string{"INPUT_CONFIG=job.yml.tmpl", "INPUT_CONFIG=pipeline.yml.tmpl"}},
				"bundle/invariant/continue_293": {Added: false, VariantFilters: []string{"INPUT_CONFIG=job.yml.tmpl", "INPUT_CONFIG=pipeline.yml.tmpl"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseChangedTests(tt.diff, testDirsFixture())
			assert.Equal(t, tt.want, got)
		})
	}
}
