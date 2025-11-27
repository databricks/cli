package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// errSkip indicates that files matching this rule should be skipped.
var errSkip = errors.New("skip")

// RuleFunc is a function that processes a list of changed files.
// Returns:
//   - packages: list of packages to unit test (empty means test all)
//   - acceptancePrefixes: list of acceptance test prefixes (empty means test all)
//   - error: errSkip to skip these files, or nil/other error
type RuleFunc func(files []string) (packages, acceptancePrefixes []string, err error)

func main() {
	baseRef := os.Getenv("GITHUB_BASE_REF")
	if baseRef == "" {
		baseRef = "HEAD"
	}

	headRef := os.Getenv("GITHUB_HEAD_REF")
	if headRef == "" {
		headRef = "HEAD"
	}

	// Accept CLI arguments for testing
	if len(os.Args) == 3 {
		headRef = os.Args[1]
		baseRef = os.Args[2]
	}

	changedFiles, err := GetChangedFiles(headRef, baseRef)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting changed files: %v\n", err)
		os.Exit(1)
	}

	packages, acceptancePrefixes := applyRules(changedFiles)

	// Output packages (space-separated, or empty for all)
	if len(packages) == 0 {
		fmt.Println("")
	} else {
		fmt.Println(strings.Join(packages, " "))
	}

	// Output acceptance test prefixes (space-separated, or empty for all)
	if len(acceptancePrefixes) == 0 {
		fmt.Println("")
	} else {
		fmt.Println(strings.Join(acceptancePrefixes, " "))
	}
}

// applyRules applies rules sequentially to all files until one returns a real error or real outputs.
func applyRules(changedFiles []string) ([]string, []string) {
	rules := AllRules()

	for _, rule := range rules {
		packages, prefixes, err := rule(changedFiles)
		if err != nil && err != errSkip {
			// Real error - exit
			fmt.Fprintf(os.Stderr, "Error applying rule: %v\n", err)
			os.Exit(1)
		}
		if err == errSkip {
			// Skip this rule, continue to next rule
			continue
		}
		return packages, prefixes
	}

	// No rule matched - test everything (empty means test all)
	return []string{}, []string{}
}
