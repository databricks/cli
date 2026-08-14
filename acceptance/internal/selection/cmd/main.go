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
		fmt.Printf("  %3d  %s\n", test.Score, test.Name())
	}
	if len(result.Selected) == 0 {
		fmt.Println("       none of the changed files belong to a test dir")
	}
}

// diffFromArgs renders command line arguments as `git diff --name-status` lines, so the
// same selection runs on them as on a real diff.
func diffFromArgs(args []string) string {
	changed := changedLines()

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
		if status != "" {
			lines = append(lines, status+"\t"+strings.Join(paths, "\t"))
			continue
		}
		// A path given without a status takes the line git reports for it, kept whole so a
		// rename keeps both its status and its source path. A path git does not report as
		// changed stands for a hypothetical change: modified when tracked, added when not.
		if line, ok := changed[paths[0]]; ok {
			lines = append(lines, line)
			continue
		}
		lines = append(lines, gitStatus(paths[0])+"\t"+paths[0])
	}
	return strings.Join(lines, "\n")
}

// changedLines maps each path this branch changed to its whole `git diff --name-status`
// line, keyed the way the diff names it now: the destination path of a rename, and the
// path itself otherwise. A path filter cannot be used for this, because git only pairs a
// rename when both of its paths are in the diff — asking about the destination alone
// reports an addition.
func changedLines() map[string]string {
	out, err := exec.Command("git", "diff", "--name-status", "--merge-base", "-M", "origin/main").Output()
	if err != nil {
		fatalf("git diff --merge-base origin/main failed: %s", err)
	}

	lines := map[string]string{}
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		if fields := strings.Split(line, "\t"); len(fields) >= 2 {
			lines[fields[len(fields)-1]] = line
		}
	}
	return lines
}

// gitStatus is the status to assume for a path git does not report as changed: "M" for a
// tracked path and "A" for one git does not know.
func gitStatus(path string) string {
	if exec.Command("git", "ls-files", "--error-unmatch", "--", path).Run() == nil {
		return "M"
	}
	return "A"
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
