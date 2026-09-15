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
// Each line is a `go test -run` pattern, so a selected test can be run as printed:
//
//	go test ./acceptance -run 'TestAccept/^bundle$/^invariant$/^no_drift$/^DATABRICKS_BUNDLE_ENGINE=direct$/^INPUT_CONFIG=job.yml.tmpl$'
//
// Every element is anchored, because a -run element is a regexp matched against one level of
// the test name: unanchored, `no_drift` would also run no_drift_extra, and `DMS=` would also
// run the DMS=true variant.
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
	"regexp"
	"slices"
	"strings"

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
		result = selection.FromDiff(*root, diffFromArgs(args), testDirs, *limit)
	} else {
		result, err = selection.FromGit(*root, testDirs, *limit)
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

// variantNames returns the go test names of the selected test. go test reads a name as a
// prefix, so a name is only as long as it needs to be: the dir alone when every variant of
// it runs, and otherwise up to the variant the selection names.
func variantNames(root string, test selection.Test) []string {
	if test.Filter == "" {
		return []string{runPattern(strings.Split(test.Dir, "/"))}
	}
	key, _, _ := strings.Cut(test.Filter, "=")

	var names []string
	for _, envset := range selection.Variants(root, test.Dir) {
		if !selection.MatchesFilters(envset, []string{test.Filter}) {
			continue
		}
		// Cut the name after the variable the filter names; the variants below it all run.
		for i, kv := range envset {
			if name, _, _ := strings.Cut(kv, "="); name == key {
				envset = envset[:i+1]
				break
			}
		}
		if name := runPattern(append(strings.Split(test.Dir, "/"), envset...)); !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	return names
}

// runPattern renders test name elements as a `go test -run` pattern. Each element of such a
// pattern is a regexp matched against one level of the test name, so every element is escaped
// and anchored: otherwise `bundle/deploy/immutable` would also run immutable-no-artifacts, and
// the dots in job.yml.tmpl would match any character. The pattern is left open at the end, so
// the variants below its last element still run.
func runPattern(elements []string) string {
	anchored := make([]string, 0, len(elements))
	for _, element := range elements {
		anchored = append(anchored, "^"+regexp.QuoteMeta(element)+"$")
	}
	return strings.Join(anchored, "/")
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
