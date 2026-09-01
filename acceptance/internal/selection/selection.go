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
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/databricks/cli/acceptance/internal"
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

// Test is one test the selection picked: a whole test dir, or one variant of it when only
// some of its variants changed.
type Test struct {
	// Dir is the test dir, relative to acceptance/.
	Dir string

	// Filter is the KEY=value the variant is selected by, empty when every variant of the
	// dir runs. Each variant is picked, and scored, on its own: adding one invariant config
	// makes that config's variant new without touching the others.
	Filter string

	// Score is why the test was picked; see the score constants.
	Score int
}

// Name is the test as go test names it, without the variants the selection says nothing
// about (see MatchesFilters).
func (t Test) Name() string {
	if t.Filter == "" {
		return t.Dir
	}
	return t.Dir + "/" + t.Filter
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
// acceptance harness looks tests up by. A nil slice means every variant of the dir runs.
func (r Result) Tests() map[string][]string {
	tests := make(map[string][]string, len(r.Selected))
	for _, test := range r.Selected {
		if test.Filter == "" {
			tests[test.Dir] = nil
			continue
		}
		if filters, ok := tests[test.Dir]; !ok || filters != nil {
			tests[test.Dir] = append(filters, test.Filter)
		}
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

// MatchesFilters reports whether envset satisfies every KEY=value filter. A filter matches
// unless its key is present in envset with a different value. Filters that share a key are
// alternatives: the environment has one value per key, so requiring each of them separately
// would match nothing, while a selection naming several values of a key (one INPUT_CONFIG
// per changed invariant config) means any of them.
func MatchesFilters(envset, filters []string) bool {
	envMap := make(map[string]string, len(envset))
	for _, kv := range envset {
		key, value, _ := strings.Cut(kv, "=")
		envMap[key] = value
	}

	expected := make(map[string][]string, len(filters))
	for _, filter := range filters {
		key, value, _ := strings.Cut(filter, "=")
		expected[key] = append(expected[key], value)
	}

	for key, values := range expected {
		if actual, ok := envMap[key]; ok && !slices.Contains(values, actual) {
			return false
		}
	}
	return true
}

// Variants returns the env sets dir runs with, read from its materialized config
// (out.test.toml), which already has the excluded matrix values removed. Excludes that name a
// combination of variables are not recorded there, so a returned variant can still be one the
// harness skips.
//
// A dir with no readable materialized config has no matrix, which expands to a single variant
// with no env set, so every filter matches it. That is the conservative answer: it can only
// select more tests, never fewer, and the harness parses the same file strictly when it runs a
// test, so a malformed one is reported there.
func Variants(root, dir string) [][]string {
	var config internal.TestConfig

	contents, err := os.ReadFile(filepath.Join(root, dir, internal.MaterializedConfigFile))
	if err == nil {
		_, _ = toml.Decode(string(contents), &config)
	}

	return internal.ExpandEnvMatrix(config.EnvMatrix, nil, nil)
}

// variantIndex answers whether a test dir runs a given variant, reading each dir's
// materialized config at most once.
type variantIndex struct {
	root    string
	envsets map[string][][]string
}

// has reports whether dir runs a variant matching filter.
func (v *variantIndex) has(dir, filter string) bool {
	envsets, ok := v.envsets[dir]
	if !ok {
		envsets = Variants(v.root, dir)
		v.envsets[dir] = envsets
	}

	for _, envset := range envsets {
		if MatchesFilters(envset, []string{filter}) {
			return true
		}
	}
	return false
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
func FromGit(root string, testDirs map[string]bool, limit int) (Result, error) {
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

	return FromDiff(root, strings.TrimSpace(string(out)), testDirs, limit), nil
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
	// filters holds one record per variant the dir is restricted to. Empty means every
	// variant runs.
	filters []variantFilter

	// allVariants is set by a change to the dir itself, as opposed to a change to an
	// invariant config the dir is generated from. It clears filters and keeps a later
	// config change from narrowing the dir back down to one config.
	allVariants bool

	// newTest is set when the dir's script is new, so the whole test is new. moved is set
	// when the script arrived as an identical rename, so the test only changed location.
	newTest bool
	moved   bool

	// nonRenameChange is set by a change that is not a rename, so a dir that both moved and
	// changed is not scored as a pure move.
	nonRenameChange bool

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

func (d *changedDir) score(newVariant bool) int {
	if d.moved && !d.nonRenameChange {
		// The files of a moved dir all arrive as renames. Moving a test does not change what
		// it does, so those renames do not also count as changes.
		return scoreMoved
	}

	score := 0
	if d.newTest || newVariant {
		score += scoreNewTest
	}
	if d.fixture {
		score += scoreChange
	}
	if d.generated {
		score += scoreGenerated
	}
	if d.moved {
		// A dir that moved and changed is scored for both.
		score += scoreMoved
	}
	return score
}

// variantFilter is one variant of a dir, selected because the invariant config it is
// generated from changed.
type variantFilter struct {
	// env is the KEY=value the variant is selected by.
	env string

	// newVariant is set when the config is new, so this variant of the dir is new while its
	// other variants are not.
	newVariant bool
}

// changedDirs maps a test dir, relative to acceptance/, to how it changed.
type changedDirs map[string]*changedDir

// addFilter restricts the dir to one more variant, unless a change to the dir itself
// already runs every variant. Repeating a variant (a config and its setup script) keeps the
// stronger record.
func (d *changedDir) addFilter(env string, newVariant bool) {
	if d.allVariants {
		return
	}
	for i, filter := range d.filters {
		if filter.env == env {
			d.filters[i].newVariant = filter.newVariant || newVariant
			return
		}
	}
	d.filters = append(d.filters, variantFilter{env: env, newVariant: newVariant})
}

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

// markChanged records a changed file against its test dir. The dir runs in full, and the file
// counts as a fixture unless it is output the test generates.
func markChanged(dirs changedDirs, dir, path string) *changedDir {
	d := dirs.get(dir)
	// A change to the dir itself runs every variant, so any variant filter it picked up from a
	// config change is dropped and a later one is ignored.
	d.allVariants = true
	d.filters = nil
	if isGeneratedFile(path, dir) {
		d.generated = true
	} else {
		d.fixture = true
	}
	return d
}

// FromDiff selects among testDirs the tests touched by `git diff --name-status` output,
// keeping at most limit of them in the order documented on the score constants. root is the
// acceptance directory, read for the variants each test dir runs.
func FromDiff(root, diff string, testDirs map[string]bool, limit int) Result {
	dirs := changedDirs{}
	variants := &variantIndex{root: root, envsets: map[string][][]string{}}

	for line := range strings.SplitSeq(diff, "\n") {
		// A rename line carries both paths ("R100\told\tnew"); the last field is the
		// path that exists now.
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			continue
		}
		status := fields[0]
		path := fields[len(fields)-1]

		// A rename also changes the dir the file left: it lost an input and may no longer
		// pass. It only counts while that test dir still exists, so moving a whole dir does
		// not select the dir it came from.
		if strings.HasPrefix(status, "R") && len(fields) >= 3 {
			if source := testDirForFile(fields[1], testDirs); source != "" {
				markChanged(dirs, source, fields[1])
			}
		}

		// A changed invariant config re-enables every invariant subdir, but only for the
		// variants generated from that config.
		if configName := invariantConfigName(path); configName != "" {
			filter := "INPUT_CONFIG=" + configName
			for dir := range testDirs {
				if !strings.HasPrefix(dir, invariantDirPrefix) {
					continue
				}
				// A dir that does not run this config would be selected with every
				// variant skipped, spending the limit on a test that runs nothing.
				// A deleted config is gone from every matrix, so this drops it too.
				if !variants.has(dir, filter) {
					continue
				}
				d := dirs.get(dir)
				// The config is the fixture these dirs are generated from. A new config
				// adds one variant of each dir, so only that variant is new; the
				// -init.sh / -cleanup.sh companions of a config change how an existing
				// variant runs.
				d.fixture = true
				isNewConfig := status == "A" && strings.HasSuffix(path, configName)
				d.addFilter(filter, isNewConfig)
			}
			continue
		}

		// out.test.toml under the invariant tree regenerates automatically when
		// INPUT_CONFIG changes; ignore it so a regenerated matrix does not unlock all
		// variants of every invariant subdir. The test.toml beside it is hand-written, so
		// it is a fixture like any other.
		if strings.HasPrefix(path, acceptanceDirPrefix+invariantDirPrefix) {
			if filepath.Base(path) == internal.MaterializedConfigFile {
				continue
			}
		}

		dir := testDirForFile(path, testDirs)
		if dir == "" {
			continue
		}

		d := markChanged(dirs, dir, path)
		if !strings.HasPrefix(status, "R") {
			d.nonRenameChange = true
		}
		// The status of the dir's script says how the dir itself changed.
		if strings.HasSuffix(path, "/script") {
			switch status {
			case "A":
				d.newTest = true
			case "R100":
				// Only an identical rename is a pure move. A rename that also rewrote the
				// script is a change like any other.
				d.moved = true
			}
		}
	}

	tests := make([]Test, 0, len(dirs))
	for _, dir := range slices.Sorted(maps.Keys(dirs)) {
		d := dirs[dir]
		if len(d.filters) == 0 {
			tests = append(tests, Test{Dir: dir, Score: d.score(false)})
			continue
		}
		for _, filter := range d.filters {
			tests = append(tests, Test{Dir: dir, Filter: filter.env, Score: d.score(filter.newVariant)})
		}
	}

	// Highest score first, then by name, so the limit cuts the same set whatever order the
	// diff happens to list the changed files in.
	slices.SortFunc(tests, func(a, b Test) int {
		if byScore := b.Score - a.Score; byScore != 0 {
			return byScore
		}
		return strings.Compare(a.Name(), b.Name())
	})

	dropped := max(len(tests)-limit, 0)
	tests = tests[:len(tests)-dropped]
	return Result{Selected: tests, Dropped: dropped, Limit: limit}
}
