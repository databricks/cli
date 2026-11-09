package acceptance_test

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/databricks/cli/acceptance/internal/selection"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

// Regression mode runs the acceptance tests this branch changed against other CLI
// builds, to tell tests that pin new behavior (fail on main) from tests that cover
// existing behavior (pass everywhere). The tests and the test harness always come from
// this branch; only the CLI binary under test is swapped (see -clipath, -useversion).
var (
	RegrMain    bool
	RegrRelease bool
	RegrOutput  string
)

func init() {
	flag.BoolVar(&RegrMain, "regr-main", false, "Run the tests changed on this branch against a CLI built from the merge base with origin/main and report the difference. Use with -v to see the summary")
	flag.BoolVar(&RegrRelease, "regr-release", false, "Run the tests changed on this branch against the latest released CLI and report the difference. Use with -v to see the summary")
	flag.StringVar(&RegrOutput, "regr-output", "../regression_report.md", "Where -regr-main and -regr-release write the full markdown report")
}

const (
	// regrDefaultLimit caps how many changed tests are compared unless selection.EnvVar says otherwise.
	regrDefaultLimit = 20

	// regrDetailLines caps the output of one failed run in the report.
	regrDetailLines = 200

	prerequisitesTest = "TestAccept/prerequisites"
)

const (
	statusPass = "pass"
	statusFail = "fail"
	statusSkip = "skip"
)

// Categories a test falls into, see classifyRegression.
const (
	// catRegression passes on this branch and fails on main: it pins behavior this branch introduced.
	catRegression = "regression"

	// catUnreleased passes on this branch and main and fails on the latest release: it covers
	// behavior that is merged but not shipped yet.
	catUnreleased = "unreleased"

	// catCoverage passes everywhere.
	catCoverage = "coverage"

	// catFailing fails on this branch.
	catFailing = "failing"
)

// regrCategories lists the categories in the order reports show them.
var regrCategories = []string{catRegression, catUnreleased, catCoverage, catFailing}

// testResult is the outcome of one test in a child run, with the log lines it produced.
type testResult struct {
	Status string
	Output []string
}

// regrColumn holds the results of running the selected tests with one CLI.
type regrColumn struct {
	// Title is the column header in the report.
	Title string

	// Category is what a failure in this column means for the test, empty for the branch column.
	Category string

	Results map[string]*testResult
}

func (c regrColumn) status(test string) string {
	if r, ok := c.Results[test]; ok {
		return r.Status
	}
	return ""
}

var (
	// "=== RUN   TestAccept/foo", also CONT, NAME and PAUSE: following log lines belong to that test.
	markerLineRE = regexp.MustCompile(`^=== (?:RUN|CONT|NAME|PAUSE)\s+(\S+)`)

	// "    --- FAIL: TestAccept/foo (0.10s)"
	resultLineRE = regexp.MustCompile(`^\s*--- (PASS|FAIL|SKIP): (\S+) `)

	// The temp dir of a test, e.g. "/tmp/TestAcceptfoo123456"; the random suffix differs on every run.
	tmpDirRE = regexp.MustCompile(`/TestAccept[^/\s]*`)
)

// parseTestOutput parses the output of a test binary run with -test.v into results by test name.
func parseTestOutput(output string) map[string]*testResult {
	results := map[string]*testResult{}
	current := ""
	for line := range strings.Lines(output) {
		line = strings.TrimRight(line, "\r\n")
		if m := markerLineRE.FindStringSubmatch(line); m != nil {
			current = m[1]
			if _, ok := results[current]; !ok {
				results[current] = &testResult{}
			}
			continue
		}
		if m := resultLineRE.FindStringSubmatch(line); m != nil {
			if _, ok := results[m[2]]; !ok {
				results[m[2]] = &testResult{}
			}
			results[m[2]].Status = strings.ToLower(m[1])
			continue
		}
		if current != "" {
			results[current].Output = append(results[current].Output, line)
		}
	}
	return results
}

// leafTests returns the tests that ran a test script, sorted. A test with EnvMatrix variants
// is represented by its variants (named KEY=value) rather than by itself.
func leafTests(results map[string]*testResult) []string {
	parents := map[string]bool{}
	for name := range results {
		for i, c := range name {
			if c != '/' {
				continue
			}
			next, _, _ := strings.Cut(name[i+1:], "/")
			if strings.Contains(next, "=") {
				parents[name[:i]] = true
			}
		}
	}

	var leaves []string
	for name, r := range results {
		if parents[name] || name == "TestAccept" || name == prerequisitesTest || r.Status == "" {
			continue
		}
		leaves = append(leaves, name)
	}
	slices.Sort(leaves)
	return leaves
}

func isOK(status string) bool {
	return status == statusPass || status == statusSkip
}

// classifyRegression says what the status of one test on every CLI means. refs are the
// non-branch columns in the order they are checked.
func classifyRegression(branch string, refs, refCategories []string) string {
	if !isOK(branch) {
		return catFailing
	}
	for i, status := range refs {
		if !isOK(status) {
			return refCategories[i]
		}
	}
	return catCoverage
}

func categorize(test string, branch regrColumn, refs []regrColumn) string {
	statuses := make([]string, len(refs))
	categories := make([]string, len(refs))
	for i, ref := range refs {
		statuses[i] = ref.status(test)
		categories[i] = ref.Category
	}
	return classifyRegression(branch.status(test), statuses, categories)
}

func statusMark(status string) string {
	switch status {
	case statusPass:
		return "✅"
	case statusFail:
		return "❌"
	case statusSkip:
		return "⏭️"
	default:
		return "➖"
	}
}

// countByCategory counts the tests in each category.
func countByCategory(tests []string, branch regrColumn, refs []regrColumn) map[string]int {
	counts := map[string]int{}
	for _, test := range tests {
		counts[categorize(test, branch, refs)]++
	}
	return counts
}

func formatCounts(counts map[string]int) string {
	var parts []string
	for _, cat := range regrCategories {
		parts = append(parts, fmt.Sprintf("%s=%d", cat, counts[cat]))
	}
	return strings.Join(parts, " ")
}

// renderRegressionReport returns the markdown report. columns[0] is the branch, the rest are
// the CLIs it is compared against.
func renderRegressionReport(selected string, columns []regrColumn) string {
	branch, refs := columns[0], columns[1:]
	tests := leafTests(branch.Results)

	var b strings.Builder
	b.WriteString("## Regression test report\n\n")
	fmt.Fprintf(&b, "%s\n\n", selected)

	// The table has a row per test variant, often hundreds, so it is collapsed. The workflow
	// that posts the report always keeps this first <details> block when it trims the comment.
	fmt.Fprintf(&b, "<details>\n<summary>%d tests: %s</summary>\n\n", len(tests), formatCounts(countByCategory(tests, branch, refs)))
	b.WriteString("| test | " + strings.Join(columnTitles(columns), " | ") + " |\n")
	b.WriteString("|---|" + strings.Repeat("---|", len(columns)) + "\n")
	for _, test := range tests {
		marks := make([]string, len(columns))
		for i, c := range columns {
			marks[i] = statusMark(c.status(test))
		}
		fmt.Fprintf(&b, "| %s | %s |\n", test, strings.Join(marks, " | "))
	}
	b.WriteString("\n</details>\n\n")

	for _, test := range tests {
		renderDetails(&b, test, columns)
	}
	return b.String()
}

func columnTitles(columns []regrColumn) []string {
	titles := make([]string, len(columns))
	for i, c := range columns {
		titles[i] = c.Title
	}
	return titles
}

// renderDetails adds a collapsible section with the output of every failed run of the test.
func renderDetails(b *strings.Builder, test string, columns []regrColumn) {
	var failed []regrColumn
	for _, c := range columns {
		if c.status(test) == statusFail {
			failed = append(failed, c)
		}
	}
	if len(failed) == 0 {
		return
	}

	marks := make([]string, len(columns))
	for i, c := range columns {
		marks[i] = c.Title + " " + statusMark(c.status(test))
	}
	fmt.Fprintf(b, "<details>\n<summary>%s: %s</summary>\n\n", test, strings.Join(marks, " | "))
	// A new test commonly fails the same way on main and on the release, which would print the
	// same long diff twice.
	firstWith := map[string]string{}
	for _, c := range failed {
		output := strings.TrimSpace(strings.Join(c.Results[test].Output, "\n"))
		key := tmpDirRE.ReplaceAllString(output, "/TestAccept")
		if first, ok := firstWith[key]; ok {
			fmt.Fprintf(b, "**%s:** same output as %s\n\n", c.Title, first)
			continue
		}
		firstWith[key] = c.Title
		// The longer fence keeps backticks in the output from ending the block.
		fmt.Fprintf(b, "**%s:**\n````\n%s\n````\n\n", c.Title, truncateLines(output, regrDetailLines))
	}
	b.WriteString("</details>\n\n")
}

// printRegressionSummary prints the short form of the report: counts, and the names of the
// tests that need attention.
func printRegressionSummary(columns []regrColumn, reportPath string) {
	branch, refs := columns[0], columns[1:]
	tests := leafTests(branch.Results)

	fmt.Printf("\nRegression summary (%d tests): %s\n", len(tests), formatCounts(countByCategory(tests, branch, refs)))
	for _, cat := range []string{catRegression, catFailing} {
		for _, test := range tests {
			if categorize(test, branch, refs) == cat {
				fmt.Printf("  %s: %s\n", cat, test)
			}
		}
	}
	fmt.Printf("Full report: %s\n", reportPath)
}

// runRegression runs the tests this branch changed with the branch CLI and with each CLI
// requested by -regr-main and -regr-release, then reports the difference.
//
// Each run is a child process of this test binary, because the harness builds the CLI and
// starts the servers once per process. The children select the changed tests themselves
// (selection.EnvVar), so all of them run the same tests.
func runRegression(t *testing.T) {
	testDirs, err := selection.FindTestDirs(".")
	require.NoError(t, err)
	testDirsSet := make(map[string]bool, len(testDirs))
	for _, dir := range testDirs {
		testDirsSet[dir] = true
	}

	limit, err := selection.ParseLimit(os.Getenv(selection.EnvVar))
	require.NoError(t, err)
	if limit == 0 {
		limit = regrDefaultLimit
	}

	result, err := selection.FromGit(".", testDirsSet, limit)
	require.NoError(t, err)
	fmt.Println(result.Counts())
	if len(result.Selected) == 0 {
		fmt.Println("No acceptance tests were changed on this branch, nothing to compare")
		return
	}

	cwd, err := os.Getwd()
	require.NoError(t, err)

	// The CLI of main does not depend on the branch run, so it is built while that runs.
	var mainCLI, mainShort string
	var mainBuild errgroup.Group
	if RegrMain {
		mergeBase := gitOutput(t, "merge-base", "HEAD", "origin/main")
		mainShort = gitOutput(t, "rev-parse", "--short", mergeBase)
		src, fips := t.TempDir(), readGOFIPS140(t, cwd)
		fmt.Printf("Building the CLI of main (%s)\n", mainShort)
		mainBuild.Go(func() (err error) {
			mainCLI, err = buildCLIAtRef(src, mergeBase, fips)
			return err
		})
	}

	fmt.Println("Running the selected tests with the CLI of this branch")
	columns := []regrColumn{{Title: "branch", Results: runAcceptanceChild(t, limit)}}

	if RegrMain {
		require.NoError(t, mainBuild.Wait())
		fmt.Printf("Running the selected tests with the CLI of main (%s)\n", mainShort)
		columns = append(columns, regrColumn{
			Title:    "main (" + mainShort + ")",
			Category: catRegression,
			Results:  runAcceptanceChild(t, limit, "-clipath="+mainCLI),
		})
	}

	if RegrRelease {
		version := resolveLatestVersion(t, getBuildDir(t, cwd, runtime.GOOS, runtime.GOARCH))
		fmt.Printf("Running the selected tests with the latest release (v%s)\n", version)
		columns = append(columns, regrColumn{
			Title:    "latest (v" + version + ")",
			Category: catUnreleased,
			Results:  runAcceptanceChild(t, limit, "-useversion="+version),
		})
	}

	report := renderRegressionReport(result.Counts(), columns)
	require.NoError(t, os.WriteFile(RegrOutput, []byte(report), 0o644))
	printRegressionSummary(columns, filepath.ToSlash(RegrOutput))
}

// runAcceptanceChild runs TestAccept in a child process of this test binary and returns what it
// reports. Failing tests are expected and are not an error, a run without results is.
func runAcceptanceChild(t *testing.T, limit int, args ...string) map[string]*testResult {
	args = append([]string{
		"-test.run=^TestAccept$",
		"-test.v",
		"-test.count=1",
		"-test.timeout=" + flag.Lookup("test.timeout").Value.String(),
	}, args...)

	cmd := exec.Command(os.Args[0], args...)
	cmd.Env = append(os.Environ(), selection.EnvVar+"="+strconv.Itoa(limit))
	out, err := cmd.CombinedOutput()
	if _, ok := errors.AsType[*exec.ExitError](err); err != nil && !ok {
		require.NoError(t, err)
	}

	results := parseTestOutput(string(out))
	if len(leafTests(results)) == 0 {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		require.Fail(t, "no test results", "%s failed:\n%s", cmd.Args, strings.Join(lines[max(0, len(lines)-50):], "\n"))
	}
	return results
}

// buildCLIAtRef builds the CLI from the source tree at ref in the empty directory src, and
// returns the path of the binary. The tree is exported with git archive so the working tree is
// not touched. It runs next to a test, so it reports errors instead of failing t.
func buildCLIAtRef(src, ref, gofips140 string) (string, error) {
	archive := exec.Command("git", "archive", "--format=tar", ref)
	// git archive exports the subtree of its working directory, and the CLI is the whole repo.
	archive.Dir = ".."
	extract := exec.Command("tar", "-x", "-C", src)
	pipe, err := archive.StdoutPipe()
	if err != nil {
		return "", err
	}
	extract.Stdin = pipe
	if err := archive.Start(); err != nil {
		return "", fmt.Errorf("git archive %s: %w", ref, err)
	}
	// Waiting for the extraction first lets tar drain the pipe before archive.Wait closes it.
	if err := extract.Run(); err != nil {
		return "", fmt.Errorf("extracting %s: %w", ref, err)
	}
	if err := archive.Wait(); err != nil {
		return "", fmt.Errorf("git archive %s: %w", ref, err)
	}

	cli := filepath.Join(src, "databricks"+exeSuffix)
	// The exported tree has no .git, so VCS stamping would fail.
	build := exec.Command("go", "build", "-buildvcs=false", "-o", cli, ".")
	build.Dir = src
	build.Env = append(os.Environ(), "GOFIPS140="+gofips140)
	if out, err := build.CombinedOutput(); err != nil {
		return "", fmt.Errorf("building the CLI at %s: %w\n%s", ref, err, strings.TrimSpace(string(out)))
	}
	return cli, nil
}

func gitOutput(t *testing.T, args ...string) string {
	out, err := exec.Command("git", args...).Output()
	require.NoError(t, err, "git %s", strings.Join(args, " "))
	return strings.TrimSpace(string(out))
}
