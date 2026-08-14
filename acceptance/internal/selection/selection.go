// Package selection picks the acceptance tests a branch changed, so a run can cover what a
// PR touches instead of the whole suite. The acceptance harness uses it for
// DATABRICKS_TEST_SELECT_CHANGED; the cmd subpackage exposes the same selection as a
// command, to inspect what a given change would run.
package selection

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const (
	// EnvVar holds the number of changed tests to select. Unset means no selection.
	EnvVar = "DATABRICKS_TEST_SELECT_CHANGED"

	// EntryPointScript is the file whose presence marks a directory as a test case.
	EntryPointScript = "script"

	// acceptanceDirPrefix is where the test dirs live relative to the repo root, since
	// git reports repo-relative paths while test dirs are named relative to acceptance/.
	acceptanceDirPrefix = "acceptance/"

	invariantConfigsPrefix = acceptanceDirPrefix + "bundle/invariant/configs/"
	invariantDirPrefix     = "bundle/invariant/"
)

// Test is one test the selection picked.
type Test struct {
	// Dir is the test dir, relative to acceptance/.
	Dir string

	// Filters restricts the run to the variants matching these KEY=value filters. A nil
	// slice means every variant of the dir runs.
	Filters []string

	// Score is why the test was picked; see the score constants.
	Score int
}

// Name is the test dir with its variant filters, as the log and the command print it.
func (t Test) Name() string {
	if t.Filters == nil {
		return t.Dir
	}
	return t.Dir + "[" + strings.Join(t.Filters, ",") + "]"
}

// Result is the outcome of a selection.
type Result struct {
	// Selected lists the picked tests, highest score first and alphabetical within a
	// score.
	Selected []Test

	// Dropped counts the changed tests that did not fit Limit.
	Dropped int

	// Limit is the cap this selection was made with.
	Limit int
}

// Tests maps each selected test dir to the variant filters it runs with, the form the
// acceptance harness looks tests up by.
func (r Result) Tests() map[string][]string {
	tests := make(map[string][]string, len(r.Selected))
	for _, test := range r.Selected {
		tests[test.Dir] = test.Filters
	}
	return tests
}

// Counts says how many tests were selected and how many the limit cut.
func (r Result) Counts() string {
	return fmt.Sprintf("Selected %d changed tests (limit=%d, %d not selected)", len(r.Selected), r.Limit, r.Dropped)
}

// Summary is the whole outcome on one line, for the test log.
func (r Result) Summary() string {
	names := make([]string, 0, len(r.Selected))
	for _, test := range r.Selected {
		names = append(names, test.Name())
	}
	return r.Counts() + ": " + strings.Join(names, " ")
}

// ParseLimit reads the number of tests to select from a raw EnvVar value. An empty value
// yields 0, which means no selection.
func ParseLimit(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}

	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		return 0, fmt.Errorf("invalid %s=%q, expected a positive integer", EnvVar, raw)
	}

	return limit, nil
}

// FindTestDirs returns every test dir under root, named relative to root with forward
// slashes, sorted.
func FindTestDirs(root string) ([]string, error) {
	var dirs []string

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || entry.Name() != EntryPointScript {
			return nil
		}
		rel, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		dirs = append(dirs, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}

	slices.Sort(dirs)
	return dirs, nil
}

// FromGit selects among testDirs the tests this branch changed, at most limit of them.
//
// --merge-base diffs the working tree against the merge base of HEAD and
// origin/main. This covers committed, staged, and unstaged changes alike —
// the working tree reflects all three. Untracked files (not yet git-added)
// are not visible to git diff and will not be re-enabled until staged or
// committed. The three-dot form origin/main...HEAD only covers committed
// changes and misses unstaged edits, which breaks the "touch a config, run
// the test" local dev workflow (same reason lintdiff.py uses --merge-base).
func FromGit(testDirs map[string]bool, limit int) (Result, error) {
	out, err := exec.Command("git", "diff", "--name-status", "--merge-base", "-M", "origin/main").Output()
	if err != nil {
		// A failed diff (most commonly a missing origin/main in a shallow CI checkout)
		// must not be silently treated as "nothing changed": that disables change
		// detection and lets newly added tests skip. Every caller (push.yml PR cells,
		// integration runs) fetches origin/main, so the caller should fail loudly.
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			if stderr := strings.TrimSpace(string(exitErr.Stderr)); stderr != "" {
				return Result{}, fmt.Errorf("git diff --merge-base origin/main failed: %w: %s", err, stderr)
			}
		}
		return Result{}, fmt.Errorf("git diff --merge-base origin/main failed: %w", err)
	}

	return FromDiff(strings.TrimSpace(string(out)), testDirs, limit), nil
}

// testDirForFile maps a repo-relative changed file (e.g. acceptance/bundle/foo/script)
// to its owning test dir relative to acceptance/ (e.g. bundle/foo), or "" if the file
// is outside acceptance/ or not under any known test dir.
func testDirForFile(repoRelPath string, testDirs map[string]bool) string {
	parts := strings.Split(filepath.ToSlash(repoRelPath), "/")
	if len(parts) < 2 || parts[0]+"/" != acceptanceDirPrefix {
		return ""
	}
	// Longest ancestor first so nested tests map to the innermost test dir.
	for depth := len(parts); depth > 1; depth-- {
		candidate := strings.Join(parts[1:depth], "/")
		if testDirs[candidate] {
			return candidate
		}
	}
	return ""
}

// changedDir records how one test dir changed and which of its variants should run.
type changedDir struct {
	// filters restricts the run to the variants matching these KEY=value filters.
	// Empty means every variant of the dir runs.
	filters []string

	// allVariants is set by a change to the dir itself, as opposed to a change to an
	// invariant config the dir is generated from. It clears filters and keeps a later
	// config change from narrowing the dir back down to one config.
	allVariants bool

	// newTest is set when the test itself is new: the dir's script is new, or a new
	// invariant config adds a variant of the dir. moved is set when the script arrived as
	// a rename, so the test only changed location. score treats them as exclusive.
	newTest bool
	moved   bool

	// fixture is set when a file the test is made of changed, generated when a file the
	// test produces changed (out*). A dir with only generated changes is one whose golden
	// output was regenerated.
	fixture   bool
	generated bool
}

// The cap takes the highest scoring dirs, and the scores add up, so a dir that changed in
// several ways outranks one that changed in a single way. A new test is the most likely to
// be broken, while a dir where nothing but the golden output changed scores lowest: that
// usually follows a change elsewhere in the tree and lands on hundreds of dirs at once,
// which would otherwise fill the quota with tests this branch never edited.
const (
	scoreNewTest   = 5
	scoreChange    = 5
	scoreGenerated = 1
	scoreMoved     = 1
)

func (d *changedDir) score() int {
	score := 0
	switch {
	case d.newTest:
		score += scoreNewTest
	case d.moved:
		// The files of a moved dir all arrive as renames. Moving a test does not change
		// what it does, so those renames do not also count as changes. A dir that is new
		// and moved at once (a renamed invariant dir picking up a new config) is scored as
		// new, since being new says more about it than the move does.
		return scoreMoved
	}
	if d.fixture {
		score += scoreChange
	}
	if d.generated {
		score += scoreGenerated
	}
	return score
}

// changedDirs maps a test dir, relative to acceptance/, to how it changed.
type changedDirs map[string]*changedDir

func (c changedDirs) get(dir string) *changedDir {
	if d, ok := c[dir]; ok {
		return d
	}
	d := &changedDir{}
	c[dir] = d
	return d
}

// invariantConfigName returns the config a changed file under acceptance/bundle/invariant/
// configs/ belongs to (job.yml.tmpl for both job.yml.tmpl and job.yml.tmpl-init.sh), or ""
// for any other path.
func invariantConfigName(path string) string {
	if !strings.HasPrefix(path, invariantConfigsPrefix) {
		return ""
	}
	name := strings.TrimPrefix(path, invariantConfigsPrefix)
	// Strip -init.sh / -cleanup.sh suffixes to get the base config name.
	if i := strings.Index(name, "-"); i > 0 && strings.HasSuffix(name, ".sh") {
		name = name[:i]
	}
	if !strings.HasSuffix(name, ".yml.tmpl") {
		return ""
	}
	return name
}

// isGeneratedFile reports whether path is a file the test generates rather than a fixture
// the test is made of. A file counts as generated when its path relative to the test dir
// starts with "out" (output.txt, out.requests.txt, out.test.toml) — the same rule the
// harness uses to split inputs from outputs when it copies a test dir, so a nested file
// such as subdir/outer.py stays a fixture.
func isGeneratedFile(path, dir string) bool {
	return strings.HasPrefix(strings.TrimPrefix(path, acceptanceDirPrefix+dir+"/"), "out")
}

// FromDiff selects among testDirs the tests touched by `git diff --name-status` output,
// keeping at most limit of them in the order documented on the score constants.
func FromDiff(diff string, testDirs map[string]bool, limit int) Result {
	dirs := changedDirs{}

	for line := range strings.SplitSeq(diff, "\n") {
		// A rename line carries both paths ("R100\told\tnew"); the last field is the
		// path that exists now.
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			continue
		}
		status := fields[0]
		path := fields[len(fields)-1]

		// A changed invariant config re-enables every invariant subdir, but only for the
		// variants generated from that config.
		if configName := invariantConfigName(path); configName != "" {
			for dir := range testDirs {
				if !strings.HasPrefix(dir, invariantDirPrefix) {
					continue
				}
				d := dirs.get(dir)
				// The config is the fixture these dirs are generated from, and a new
				// config adds a variant of each of them. Its -init.sh / -cleanup.sh
				// companions change how an existing variant runs, so only the config
				// itself counts as a new test.
				d.fixture = true
				if status == "A" && strings.HasSuffix(path, configName) {
					d.newTest = true
				}
				filter := "INPUT_CONFIG=" + configName
				switch {
				case d.allVariants:
					// A change to the dir itself already runs every variant.
				case len(d.filters) == 0:
					d.filters = []string{filter}
				case d.filters[0] != filter:
					// The harness requires every filter to match (see checkEnvFilters), so
					// two different INPUT_CONFIG values would skip every variant and the
					// dir would run nothing. Run all of its variants instead.
					d.allVariants = true
					d.filters = nil
				}
			}
			continue
		}

		// test.toml and out.test.toml under the invariant tree regenerate
		// automatically when INPUT_CONFIG changes; ignore them so they don't
		// unlock all variants of every invariant subdir.
		if strings.HasPrefix(path, acceptanceDirPrefix+invariantDirPrefix) {
			if name := filepath.Base(path); name == "test.toml" || name == "out.test.toml" {
				continue
			}
		}

		dir := testDirForFile(path, testDirs)
		if dir == "" {
			continue
		}

		d := dirs.get(dir)
		d.allVariants = true
		d.filters = nil
		if isGeneratedFile(path, dir) {
			d.generated = true
		} else {
			d.fixture = true
		}
		// The status of the dir's script says how the dir itself changed.
		if strings.HasSuffix(path, "/script") {
			switch {
			case status == "A":
				d.newTest = true
			case strings.HasPrefix(status, "R"):
				d.moved = true
			}
		}
	}

	// Sort by name first, then stably by descending score, so dirs that score the same
	// stay alphabetical.
	selected := slices.Sorted(maps.Keys(dirs))
	slices.SortStableFunc(selected, func(a, b string) int {
		return dirs[b].score() - dirs[a].score()
	})

	dropped := max(len(selected)-limit, 0)
	selected = selected[:len(selected)-dropped]

	tests := make([]Test, 0, len(selected))
	for _, dir := range selected {
		tests = append(tests, Test{Dir: dir, Filters: dirs[dir].filters, Score: dirs[dir].score()})
	}
	return Result{Selected: tests, Dropped: dropped, Limit: limit}
}
