package acceptance_test

import (
	"errors"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Cloud PR runs set DATABRICKS_TEST_SELECT_CHANGED=N to run only the acceptance
// tests this branch touches, at most N of them, instead of the full suite.
const (
	SelectChangedEnvVar = "DATABRICKS_TEST_SELECT_CHANGED"

	// Cap for runs that need change detection without setting the env var: the subset
	// selector keeps changed tests on top of its hash-selected fraction, so a PR that
	// edits hundreds of test dirs must not turn the subset cells back into a full run.
	subsetChangedLimit = 50

	invariantConfigsPrefix = "acceptance/bundle/invariant/configs/"
	invariantDirPrefix     = "bundle/invariant/"
)

// getSelectChangedLimit returns the number of changed tests to select, or 0 when
// DATABRICKS_TEST_SELECT_CHANGED is unset (feature off).
func getSelectChangedLimit(t *testing.T) int {
	raw := os.Getenv(SelectChangedEnvVar)
	if raw == "" {
		return 0
	}

	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		t.Fatalf("Invalid %s=%q, expected a positive integer", SelectChangedEnvVar, raw)
	}

	return limit
}

// testDirForFile maps a repo-relative changed file (e.g. acceptance/bundle/foo/script)
// to its owning test dir relative to acceptance/ (e.g. bundle/foo), or "" if the file
// is outside acceptance/ or not under any known test dir.
func testDirForFile(repoRelPath string, testDirs map[string]bool) string {
	parts := strings.Split(filepath.ToSlash(repoRelPath), "/")
	if len(parts) < 2 || parts[0] != "acceptance" {
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

// selectChangedTests returns a map of test dir → extra env filters for the tests this
// branch changed. A nil filter slice means all variants of that dir run; a non-nil
// slice restricts to variants matching those filters (applied by the caller via
// checkEnvFilters in the variant loop).
//
// --merge-base diffs the working tree against the merge base of HEAD and
// origin/main. This covers committed, staged, and unstaged changes alike —
// the working tree reflects all three. Untracked files (not yet git-added)
// are not visible to git diff and will not be re-enabled until staged or
// committed. The three-dot form origin/main...HEAD only covers committed
// changes and misses unstaged edits, which breaks the "touch a config, run
// the test" local dev workflow (same reason lintdiff.py uses --merge-base).
func selectChangedTests(t *testing.T, testDirs map[string]bool, limit int) map[string][]string {
	out, err := exec.Command("git", "diff", "--name-status", "--merge-base", "-M", "origin/main").Output()
	if err != nil {
		// A failed diff (most commonly a missing origin/main in a shallow CI
		// checkout) must not be silently treated as "nothing changed": that
		// disables change detection and lets newly added tests skip. Fail loudly.
		// Every caller (push.yml PR cells, integration runs) now fetches origin/main.
		stderr := ""
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			stderr = strings.TrimSpace(string(exitErr.Stderr))
		}
		t.Fatalf("git diff --merge-base origin/main failed: %v\n%s", err, stderr)
	}

	changed, dropped := classifyChangedTests(strings.TrimSpace(string(out)), testDirs, limit)

	// Log the outcome up front: which tests the diff picked, and how many the limit
	// cut, so a CI run shows what it is about to cover without reading every skip line.
	names := make([]string, 0, len(changed))
	for dir, filters := range changed {
		if filters != nil {
			dir += "[" + strings.Join(filters, ",") + "]"
		}
		names = append(names, dir)
	}
	slices.Sort(names)
	t.Logf("Selected %d changed tests (limit=%d, %d not selected): %s", len(names), limit, dropped, strings.Join(names, " "))

	return changed
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

	// added is set when the dir's script is new, moved when the script arrived as a
	// rename, and fixture when any file the test is made of changed (that is, anything
	// but the generated out* files). A dir with no fixture change is one whose golden
	// output was regenerated.
	added   bool
	moved   bool
	fixture bool
}

// Order the cap selects in. An added test is the most likely to be broken and a moved one
// the least, since its content did not change. A regenerated golden ranks below a changed
// fixture because it usually comes from a change elsewhere in the tree and lands on
// hundreds of dirs at once, which would otherwise fill the quota with tests this branch
// never edited.
const (
	rankAdded = iota
	rankFixture
	rankGenerated
	rankMoved
)

func (d *changedDir) rank() int {
	switch {
	case d.added:
		return rankAdded
	case d.moved:
		return rankMoved
	case d.fixture:
		return rankFixture
	default:
		return rankGenerated
	}
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
	return strings.HasPrefix(strings.TrimPrefix(path, "acceptance/"+dir+"/"), "out")
}

// classifyChangedTests maps `git diff --name-status` output to test dirs and keeps at most
// limit of them, in the order documented on the rank constants. It also returns how many
// changed dirs the limit dropped.
func classifyChangedTests(diff string, testDirs map[string]bool, limit int) (map[string][]string, int) {
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
				// The config is the fixture these dirs are generated from.
				d.fixture = true
				if !d.allVariants {
					d.filters = append(d.filters, "INPUT_CONFIG="+configName)
				}
			}
			continue
		}

		// test.toml and out.test.toml under the invariant tree regenerate
		// automatically when INPUT_CONFIG changes; ignore them so they don't
		// unlock all variants of every invariant subdir.
		if strings.HasPrefix(path, "acceptance/"+invariantDirPrefix) {
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
		if !isGeneratedFile(path, dir) {
			d.fixture = true
		}
		// The status of the dir's script says how the dir itself changed.
		if strings.HasSuffix(path, "/script") {
			switch {
			case status == "A":
				d.added = true
			case strings.HasPrefix(status, "R"):
				d.moved = true
			}
		}
	}

	// Sort by name first, then stably by rank, so dirs of equal rank stay alphabetical.
	selected := slices.Sorted(maps.Keys(dirs))
	slices.SortStableFunc(selected, func(a, b string) int {
		return dirs[a].rank() - dirs[b].rank()
	})

	dropped := max(len(selected)-limit, 0)
	selected = selected[:len(selected)-dropped]

	out := make(map[string][]string, len(selected))
	for _, dir := range selected {
		out[dir] = dirs[dir].filters
	}
	return out, dropped
}
