package acceptance_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const sampleOutput = `=== RUN   TestAccept
=== RUN   TestAccept/prerequisites
--- PASS: TestAccept/prerequisites (0.10s)
=== RUN   TestAccept/bundle/a
=== PAUSE TestAccept/bundle/a
=== RUN   TestAccept/bundle/inv
=== RUN   TestAccept/bundle/inv/DMS=true/INPUT_CONFIG=x.yml.tmpl
=== CONT  TestAccept/bundle/a
    acceptance_test.go:1: boom
=== CONT  TestAccept/bundle/inv/DMS=true/INPUT_CONFIG=x.yml.tmpl
--- FAIL: TestAccept (1.00s)
    --- PASS: TestAccept/prerequisites (0.10s)
    --- FAIL: TestAccept/bundle/a (0.50s)
    --- SKIP: TestAccept/bundle/inv (0.00s)
        --- PASS: TestAccept/bundle/inv/DMS=true/INPUT_CONFIG=x.yml.tmpl (0.00s)
`

func TestParseTestOutput(t *testing.T) {
	results := parseTestOutput(sampleOutput)
	assert.Equal(t, statusFail, results["TestAccept/bundle/a"].Status)
	assert.Equal(t, []string{"    acceptance_test.go:1: boom"}, results["TestAccept/bundle/a"].Output)
	assert.Equal(t, statusPass, results["TestAccept/bundle/inv/DMS=true/INPUT_CONFIG=x.yml.tmpl"].Status)
}

func TestLeafTests(t *testing.T) {
	assert.Equal(t,
		[]string{"TestAccept/bundle/a", "TestAccept/bundle/inv/DMS=true/INPUT_CONFIG=x.yml.tmpl"},
		leafTests(parseTestOutput(sampleOutput)))
}

func TestClassifyRegression(t *testing.T) {
	refCats := []string{catRegression, catUnreleased}
	tests := []struct {
		branch string
		refs   []string
		want   string
	}{
		{statusFail, []string{statusPass, statusPass}, catFailing},
		{statusPass, []string{statusFail, statusFail}, catRegression},
		{statusPass, []string{statusPass, statusFail}, catUnreleased},
		{statusPass, []string{statusSkip, statusSkip}, catCoverage},
		{statusPass, []string{"", statusPass}, catRegression},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, classifyRegression(tt.branch, tt.refs, refCats), "%v", tt)
	}
}

func TestRenderRegressionReport(t *testing.T) {
	branch := regrColumn{Title: "branch", Results: parseTestOutput(sampleOutput)}
	main := regrColumn{Title: "main (abc)", Category: catRegression, Results: map[string]*testResult{
		"TestAccept/bundle/a": {Status: statusFail, Output: []string{"main failure"}},
	}}

	report := renderRegressionReport("Selected 2", []regrColumn{branch, main})
	assert.Contains(t, report, "<details>\n<summary>2 tests: regression=1 unreleased=0 coverage=0 failing=1</summary>\n\n| test | branch | main (abc) |")
	assert.Contains(t, report, "| TestAccept/bundle/a | ❌ | ❌ |")
	assert.Contains(t, report, "| TestAccept/bundle/inv/DMS=true/INPUT_CONFIG=x.yml.tmpl | ✅ | ➖ |\n\n</details>\n")
	assert.Contains(t, report, "**main (abc):**\n````\nmain failure\n````")
	assert.Contains(t, report, "failing=1")
}

func TestRenderRegressionReportDedupesIdenticalFailures(t *testing.T) {
	failure := func(tmp string) *testResult {
		return &testResult{Status: statusFail, Output: []string{"Messages: output.txt vs /tmp/TestAcceptfoo" + tmp + "/002/output.txt", "diff"}}
	}
	test := "TestAccept/bundle/a"
	columns := []regrColumn{
		{Title: "branch", Results: map[string]*testResult{test: {Status: statusPass}}},
		{Title: "main (abc)", Category: catRegression, Results: map[string]*testResult{test: failure("111")}},
		{Title: "latest (v1)", Category: catUnreleased, Results: map[string]*testResult{test: failure("222")}},
	}

	report := renderRegressionReport("Selected 1", columns)
	assert.Equal(t, 1, strings.Count(report, "\ndiff\n"))
	assert.Contains(t, report, "**latest (v1):** same output as main (abc)")
}
