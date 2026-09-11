/*
Start the test runner (gotestsum) as usual.

Download https://github.com/databricks/cli/blob/ciconfig/known_failures.txt in parallel.

If download was successful by the time test runner finishes and test runner finishes with non-zero code,
analyze failures in the test output identified by --jsonfile option. If all failures are expected (listed in known_failures.txt)
then the failure is masked, the process exits with 0.

A parent test fails whenever one of its subtests fails, so a rule that lists a
subtest also allows the parent to fail. That allowance only applies when a
subtest actually failed; if the parent failed on its own (e.g. in setup, before
any subtest ran), its failure is reported as unexpected.
*/
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"syscall"
)

const (
	maxConfigSize = 10 * 1024 // 10KB
	repoConfigURL = "https://raw.githubusercontent.com/databricks/cli/ciconfig/known_failures.txt"
	cutPrefix     = "github.com/databricks/cli/"
)

type TestResult struct {
	Action  string `json:"Action,omitempty"`
	Package string `json:"Package,omitempty"`
	Test    string `json:"Test,omitempty"`
}

func getExitCode(err error) (int, error) {
	if err == nil {
		return 0, nil
	}
	if exitError, ok := err.(*exec.ExitError); ok {
		if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus(), nil
		}
	}
	return 1, err
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <command> [args...]\n", os.Args[0])
		os.Exit(1)
	}

	// Find --jsonfile argument
	jsonFile := ""
	for i, arg := range os.Args {
		if arg == "--jsonfile" && i+1 < len(os.Args) {
			jsonFile = os.Args[i+1]
			break
		}
	}

	if jsonFile == "" {
		fmt.Println("No --jsonfile argument found")
	}

	// Start the main subprocess
	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	var configContent atomic.Value

	// Start background config download
	go func() {
		content, err := downloadConfig(context.Background())
		if err != nil {
			fmt.Printf("testrunner: Failed to download %s: %v\n", repoConfigURL, err)
		} else {
			configContent.Store(content)
		}
	}()

	// Run the main command
	err := cmd.Run()

	exitCode, err := getExitCode(err)
	if err != nil {
		fmt.Printf("testrunner: Failed to run command: %v\n", err)
		os.Exit(1)
	}

	// Success case, exit early
	if exitCode == 0 || jsonFile == "" {
		os.Exit(exitCode)
	}

	// Check if config is ready
	content := configContent.Load()

	if content == "" {
		fmt.Printf("CI config download not completed, propagating exit code %d", exitCode)
		os.Exit(exitCode)
	}

	config, err := parseConfig(content.(string))
	if err != nil {
		fmt.Printf("Error parsing CI config: %v\n", err)
		os.Exit(exitCode)
	}

	finalExitCode := checkFailures(config, jsonFile, exitCode)
	os.Exit(finalExitCode)
}

func downloadConfig(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, repoConfigURL, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Read limited body
	limitedReader := io.LimitReader(resp.Body, maxConfigSize+1)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", err
	}

	if len(body) > maxConfigSize {
		fmt.Printf("Warning: CI config body was truncated at %d bytes", maxConfigSize)
		body = body[:maxConfigSize]
	}

	return string(body), nil
}

func checkFailures(config *Config, jsonFile string, originalExitCode int) int {
	// Parse JSON test results
	file, err := os.Open(jsonFile)
	if err != nil {
		fmt.Printf("testrunner: failed to open JSON file %s: %v\n", jsonFile, err)
		return originalExitCode
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	failed := map[string]bool{}
	unexpectedFailures := map[string]bool{}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var result TestResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			fmt.Printf("failed to parse json: %q: %s\n", line, err)
			return originalExitCode
		}

		if result.Test == "" {
			continue
		}

		result.Package, _ = strings.CutPrefix(result.Package, cutPrefix)
		key := result.Package + " " + result.Test

		switch result.Action {
		case "fail":
			// Go reports a parent's failure only after its subtests, so any
			// failing subtest is already in `failed` by the time we get here.
			failed[key] = true
			matchedRule := config.matches(result.Package, result.Test)
			switch {
			case matchedRule != "":
				fmt.Printf("%s %s failure is allowed, matches rule %q\n", result.Package, result.Test, matchedRule)
			case hasFailedSubtest(failed, key):
				// A parent test fails when a subtest fails; the subtest is
				// reported on its own, so the parent's failure is not counted.
				fmt.Printf("%s %s failure is allowed, a subtest failed\n", result.Package, result.Test)
			default:
				fmt.Printf("%s %s failure is not allowed\n", result.Package, result.Test)
				unexpectedFailures[key] = true
			}
		case "pass":
			// We run gotestsum with --rerun-fails, so a test that fails and
			// later passes is flaky, not a failure.
			delete(failed, key)
			if unexpectedFailures[key] {
				fmt.Printf("%s %s passed on retry\n", result.Package, result.Test)
				delete(unexpectedFailures, key)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("testrunner: error reading JSON file: %v\n", err)
		return originalExitCode
	}

	if len(unexpectedFailures) == 0 {
		return 0
	}
	fmt.Printf("testrunner: %d test failures were not expected\n", len(unexpectedFailures))
	return originalExitCode
}

// hasFailedSubtest reports whether any failing test is a strict subtest of key.
// Keys are "<package> <test>", so a subtest's key is prefixed by key + "/".
func hasFailedSubtest(failed map[string]bool, key string) bool {
	prefix := key + "/"
	for k := range failed {
		if strings.HasPrefix(k, prefix) {
			return true
		}
	}
	return false
}

// CI Config Format
//
// The CI config is downloaded from the "ciconfig" branch of the repository.
// It's a text file with the following format:
//
//   package testcase
//
// Where:
// - Lines with whitespace only are ignored
// - Everything after '#' is a comment and is ignored
// - Both package and testcase are matched segment by segment, split on '/'
// - A '*' segment matches any number of segments (including zero), so a whole
//   field of '*' matches anything and an interior '*' spans the segments
//   between two fixed ones
// - A trailing '/' makes the pattern a prefix: it matches the listed segments
//   plus any number of segments below them
//
// Examples:
//   "libs/ *"                  - all packages under "libs/" and all testcases are allowed to fail
//   "* TestAccept/"            - all testcases under "TestAccept/" are allowed to fail
//   "bundle TestDeploy"        - exact match for package "bundle" and testcase "TestDeploy"
//   "* TestAccept/*/DMS=true/" - every testcase with a "DMS=true" segment, at any depth
//
// Parse errors for individual lines are logged but do not abort processing.

type Config struct {
	rules []ConfigRule
}

func (c *Config) matches(packageName, testName string) string {
	pkg := strings.Split(packageName, "/")
	test := strings.Split(testName, "/")
	for _, rule := range c.rules {
		if rule.matches(pkg, test) {
			return rule.OriginalLine
		}
	}
	return ""
}

type ConfigRule struct {
	PackagePattern []string
	TestPattern    []string
	OriginalLine   string
}

func parseConfig(content string) (*Config, error) {
	config := &Config{}
	scanner := bufio.NewScanner(strings.NewReader(content))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines
		if line == "" {
			continue
		}

		// Remove comments
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
			if line == "" {
				continue
			}
		}

		// Parse rule
		rule, err := parseConfigRule(line, scanner.Text())
		if err != nil {
			fmt.Printf("Error parsing config line %d: %q - %v\n", lineNum, line, err)
			continue
		}

		config.rules = append(config.rules, rule)
	}

	return config, scanner.Err()
}

// parsePattern splits a pattern field into segments. A trailing "/" marks a
// prefix match, represented as a trailing "*" segment so it matches any number
// of segments below the listed ones.
func parsePattern(pattern string) []string {
	prefix := strings.HasSuffix(pattern, "/")
	segments := strings.Split(strings.TrimSuffix(pattern, "/"), "/")
	if prefix {
		segments = append(segments, "*")
	}
	return segments
}

func parseConfigRule(line, originalLine string) (ConfigRule, error) {
	parts := strings.Fields(line)
	if len(parts) != 2 {
		return ConfigRule{}, fmt.Errorf("expected 2 fields, got %d", len(parts))
	}

	return ConfigRule{
		PackagePattern: parsePattern(parts[0]),
		TestPattern:    parsePattern(parts[1]),
		OriginalLine:   strings.TrimSpace(originalLine),
	}, nil
}

func (r ConfigRule) matches(packageName, testName []string) bool {
	return matchSegments(r.PackagePattern, packageName) &&
		matchSegments(r.TestPattern, testName)
}

// matchSegments reports whether name matches pattern segment by segment, where
// a "*" pattern segment matches any number of name segments (including zero).
// It is the standard linear wildcard match that backtracks on the latest "*".
func matchSegments(pattern, name []string) bool {
	pi, ni := 0, 0
	star, match := -1, 0
	for ni < len(name) {
		switch {
		case pi < len(pattern) && pattern[pi] == "*":
			star = pi
			match = ni
			pi++
		case pi < len(pattern) && pattern[pi] == name[ni]:
			pi++
			ni++
		case star >= 0:
			// Backtrack: let the last "*" consume one more name segment.
			pi = star + 1
			match++
			ni = match
		default:
			return false
		}
	}
	for pi < len(pattern) && pattern[pi] == "*" {
		pi++
	}
	return pi == len(pattern)
}
