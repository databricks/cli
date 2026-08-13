// Command cmd prints the acceptance tests that DATABRICKS_TEST_SELECT_CHANGED would run,
// either for the current branch or for the changed paths given as arguments. It is a way
// to see what a change selects without running the suite. Run it from the repo root:
//
//	go run ./acceptance/internal/selection/cmd
//	go run ./acceptance/internal/selection/cmd -limit 5
//	go run ./acceptance/internal/selection/cmd acceptance/bundle/invariant/configs/job.yml.tmpl
//
// An argument may carry a git status ("A:path" for an added file, "R100:old:new" for a
// rename); without one the file counts as modified.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/acceptance/internal/selection"
)

func main() {
	limit := flag.Int("limit", 50, "maximum number of changed tests to select")
	root := flag.String("root", "acceptance", "path to the acceptance directory")
	flag.Parse()

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
	for _, name := range result.Names() {
		fmt.Println("   ", name)
	}
	if len(result.Tests) == 0 {
		fmt.Println("    none of the changed files belong to a test dir")
	}
}

// diffFromArgs renders command line arguments as `git diff --name-status` lines, so the
// same selection runs on them as on a real diff.
func diffFromArgs(args []string) string {
	lines := make([]string, 0, len(args))
	for _, arg := range args {
		// The status prefix is optional: "A:path", "R100:old:new", or a plain "path".
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
