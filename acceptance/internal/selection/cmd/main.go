// Command cmd prints the acceptance tests that DATABRICKS_TEST_SELECT_CHANGED would run,
// either for the current branch or for the paths given as arguments, together with the
// score each test was picked by. It is a way to see what a change selects without running
// the suite. Run it from the repo root:
//
//	go run ./acceptance/internal/selection/cmd
//	go run ./acceptance/internal/selection/cmd -limit 5
//	go run ./acceptance/internal/selection/cmd acceptance/bundle/invariant/configs/job.yml.tmpl
//
// A bare path takes the status git reports for it: added for a path git does not track,
// modified otherwise. Prefix a path to force one ("A:path", "M:path", "R100:old:new").
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
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
	for _, test := range result.Selected {
		fmt.Printf("  %3d  %s\n", test.Score, test.Name())
	}
	if len(result.Selected) == 0 {
		fmt.Println("       none of the changed files belong to a test dir")
	}
}

// diffFromArgs renders command line arguments as `git diff --name-status` lines, so the
// same selection runs on them as on a real diff.
func diffFromArgs(args []string) string {
	lines := make([]string, 0, len(args))
	for _, arg := range args {
		status, rest, ok := strings.Cut(arg, ":")
		if !ok {
			status, rest = "", arg
		}
		// A rename carries both paths ("R100:old:new"), which git separates by tabs.
		paths := strings.Split(rest, ":")
		for i, path := range paths {
			paths[i] = strings.TrimPrefix(filepath.ToSlash(filepath.Clean(path)), "./")
		}
		if status == "" {
			status = gitStatus(paths[0])
		}
		lines = append(lines, status+"\t"+strings.Join(paths, "\t"))
	}
	return strings.Join(lines, "\n")
}

// gitStatus is the status to assume for a path given without one: the status git reports
// against the merge base if the path is changed there, otherwise "M" for a tracked path
// and "A" for one git does not know, so a bare argument also stands for a hypothetical
// change to an existing test or a brand new one.
func gitStatus(path string) string {
	out, err := exec.Command("git", "diff", "--name-status", "--merge-base", "-M", "origin/main", "--", path).Output()
	if err == nil {
		if fields := strings.Fields(string(out)); len(fields) > 0 {
			return fields[0]
		}
	}

	if exec.Command("git", "ls-files", "--error-unmatch", "--", path).Run() == nil {
		return "M"
	}
	return "A"
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
