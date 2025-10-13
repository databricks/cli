/*
Start the test runner (gotestsum) as usual.

Download https://github.com/databricks/cli/blob/ciconfig/known_failures.txt in parallel.

If download was successful by the time test runner finishes and test runner finishes with non-zero code,
analyze failures in the test output identified by --jsonfile option. If all failures are expected (listed in known_failures.txt)
then the failure is masked, the process exits with 0.
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
	req, err := http.NewRequestWithContext(ctx, "GET", repoConfigURL, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

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
	defer file.Close()

	scanner := bufio.NewScanner(file)
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

		if result.Action == "fail" {
			matchedRule := config.matches(result.Package, result.Test)
			if matchedRule != "" {
				fmt.Printf("%s %s failure is allowed, matches rule %q\n", result.Package, result.Test, matchedRule)
			} else {
				fmt.Printf("%s %s failure is not allowed\n", result.Package, result.Test)
				unexpectedFailures[key] = true
			}
		} else if result.Action == "pass" && unexpectedFailures[key] {
			fmt.Printf("%s %s passed on retry\n", result.Package, result.Test)
			// We run gotestsum with --rerun-fails, so we need to account for intermittent failures
			delete(unexpectedFailures, key)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("testrunner: error reading JSON file: %v\n", err)
		return originalExitCode
	}

	if len(unexpectedFailures) == 0 {
		return 0
	} else {
		fmt.Printf("testrunner: %d test failures were not expected\n", len(unexpectedFailures))
		return originalExitCode
	}
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
// - Both package and testcase can be '*' meaning any package or any testcase
// - Both can end with '/' which means it's a prefix match
//
// Examples:
//   "libs/ *"           - all packages starting with "libs/" and all testcases are allowed to fail
//   "* TestAccept/"     - all testcases starting with "TestAccept/" are allowed to fail
//   "bundle TestDeploy" - exact match for package "bundle" and testcase "TestDeploy"
//
// Parse errors for individual lines are logged but do not abort processing.

type Config struct {
	rules []ConfigRule
}

func (c *Config) matches(packageName, testName string) string {
	for _, rule := range c.rules {
		if rule.matches(packageName, testName) {
			return rule.OriginalLine
		}
	}
	return ""
}

type ConfigRule struct {
	PackagePattern string
	TestPattern    string
	PackagePrefix  bool
	TestPrefix     bool
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
			fmt.Printf("Error parsing config line %d: %q - %v", lineNum, line, err)
			continue
		}

		config.rules = append(config.rules, rule)
	}

	return config, scanner.Err()
}

// parsePattern returns the pattern and whether it's a prefix match.
func parsePattern(pattern string) (string, bool) {
	if pattern == "*" {
		return "", true
	}
	return strings.CutSuffix(pattern, "/")
}

func parseConfigRule(line, originalLine string) (ConfigRule, error) {
	parts := strings.Fields(line)
	if len(parts) != 2 {
		return ConfigRule{}, fmt.Errorf("expected 2 fields, got %d", len(parts))
	}

	packagePattern := parts[0]
	testPattern := parts[1]

	rule := ConfigRule{
		PackagePattern: packagePattern,
		TestPattern:    testPattern,
		OriginalLine:   strings.TrimSpace(originalLine),
	}

	// Check for wildcard or prefix
	rule.PackagePattern, rule.PackagePrefix = parsePattern(packagePattern)
	rule.TestPattern, rule.TestPrefix = parsePattern(testPattern)

	return rule, nil
}

func (r ConfigRule) matches(packageName, testName string) bool {
	// Check package pattern
	var packageMatch bool
	if r.PackagePrefix {
		packageMatch = matchesPathPrefix(packageName, r.PackagePattern)
	} else {
		packageMatch = packageName == r.PackagePattern
	}

	if !packageMatch {
		return false
	}

	// Check test pattern
	if r.TestPrefix {
		return matchesPathPrefix(testName, r.TestPattern) || matchesPathPrefix(r.TestPattern, testName)
	} else {
		return testName == r.TestPattern || matchesPathPrefix(r.TestPattern, testName)
	}
}

// matchesPathPrefix returns true if s matches pattern or starts with pattern + "/"
// If pattern is empty (wildcard "*"), it matches any string
func matchesPathPrefix(s, prefix string) bool {
	if prefix == "" {
		return true
	}
	if s == prefix {
		return true
	}
	return strings.HasPrefix(s, prefix+"/")
}
