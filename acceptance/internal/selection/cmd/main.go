// Command cmd prints the acceptance tests that DATABRICKS_TEST_SELECT_CHANGED would run,
// either for the current branch or for the paths given as arguments, together with the
// score each test was picked by. It is a way to see what a change selects without running
// the suite. Run it from the repo root:
//
//	go run ./acceptance/internal/selection/cmd
//	go run ./acceptance/internal/selection/cmd -limit 5
//	go run ./acceptance/internal/selection/cmd acceptance/bundle/invariant/configs/job.yml.tmpl
//
// With no arguments the selection comes from git, exactly as the test runner gets it. Each
// argument stands for a changed file instead, and counts as modified unless it carries a
// status ("A:path" for an added file, "R100:old:new" for a rename).
//
// Each line is a name go test accepts, so a selected test can be run as printed:
//
//	go test ./acceptance -run 'TestAccept/bundle/invariant/no_drift/DATABRICKS_BUNDLE_ENGINE=direct/INPUT_CONFIG=job.yml.tmpl'
//
// The variants come from each test's materialized config (out.test.toml), which already has
// the excluded matrix values removed. Excludes that name a combination of variables are not
// recorded there, so a listed variant can still turn out to be one the harness skips.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/databricks/cli/acceptance/internal"
	"github.com/databricks/cli/acceptance/internal/selection"
)

func main() {
	limit := flag.Int("limit", 50, "maximum number of changed tests to select")
	root := flag.String("root", "acceptance", "path to the acceptance directory")
	flag.Parse()

	if *limit <= 0 {
		fatalf("-limit must be a positive integer, got %d", *limit)
	}

	dirs, err := selection.FindTestDirs(*root)
	if err != nil {
		fatalf("cannot list test dirs in %s: %s", *root, err)
	}
	testDirs := make(map[string]bool, len(dirs))
	for _, dir := range dirs {
		testDirs[dir] = true
	}

	var result selection.Result
	if args := flag.Args(); len(args) > 0 {
		result = selection.FromDiff(diffFromArgs(args), testDirs, *limit)
	} else {
		result, err = selection.FromGit(testDirs, *limit)
		if err != nil {
			fatalf("%s", err)
		}
	}

	fmt.Println(result.Counts())
	for _, test := range result.Selected {
		for _, name := range variantNames(*root, test) {
			fmt.Printf("  %3d  %s\n", test.Score, name)
		}
	}
	if len(result.Selected) == 0 {
		fmt.Println("       none of the changed files belong to a test dir")
	}
}

// variantNames returns the go test names of the selected test: one per variant of its dir
// that the selection covers, or the dir alone when it has no variants.
func variantNames(root string, test selection.Test) []string {
	var filters []string
	if test.Filter != "" {
		filters = []string{test.Filter}
	}

	var names []string
	for _, envset := range internal.ExpandEnvMatrix(envMatrix(root, test.Dir), nil, nil) {
		if len(envset) == 0 {
			// The harness runs a test without variants as a plain subtest of its dir.
			return []string{test.Dir}
		}
		if selection.MatchesFilters(envset, filters) {
			names = append(names, test.Dir+"/"+strings.Join(envset, "/"))
		}
	}
	return names
}

// envMatrix reads the variant matrix of a test from its materialized config, the same file
// the harness generates so that inherited settings are visible.
func envMatrix(root, dir string) map[string][]string {
	path := filepath.Join(root, dir, internal.MaterializedConfigFile)
	contents, err := os.ReadFile(path)
	if err != nil {
		// A test dir without a materialized config has no variants.
		return nil
	}

	var config internal.TestConfig
	if _, err := toml.Decode(string(contents), &config); err != nil {
		fatalf("cannot parse %s: %s", path, err)
	}
	return config.EnvMatrix
}

// diffFromArgs renders command line arguments as `git diff --name-status` lines, so the
// same selection runs on them as on a real diff.
func diffFromArgs(args []string) string {
	lines := make([]string, 0, len(args))
	for _, arg := range args {
		status, rest, ok := strings.Cut(arg, ":")
		if !ok {
			status, rest = "M", arg
		}
		// A rename carries both paths ("R100:old:new"), which git separates by tabs.
		paths := strings.Split(rest, ":")
		for i, path := range paths {
			paths[i] = strings.TrimPrefix(filepath.ToSlash(filepath.Clean(path)), "./")
		}
		lines = append(lines, status+"\t"+strings.Join(paths, "\t"))
	}
	return strings.Join(lines, "\n")
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
