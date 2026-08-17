package acceptance_test

import (
	"hash/fnv"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// TESTS_SELECT_SUBSET_PCT/SEED select a random subset of subtests to run, so a CI
// cell can cover a fraction of the suite in less time. CI sets these only on the
// windows/macOS pull_request cells (see .github/workflows/push.yml); on Linux they
// are unset and the full suite runs.
const (
	// SubsetPctEnvVar is an integer 0..100: the percentage of subtests to run.
	// Unset (or empty) disables subsetting entirely.
	SubsetPctEnvVar = "TESTS_SELECT_SUBSET_PCT"

	// SubsetSeedEnvVar seeds the selection. CI sets it to the HEAD commit hash, so
	// a new commit reshuffles the subset while a retry of the same commit repeats it.
	// If unset, a random seed is generated and logged so the run can be reproduced.
	SubsetSeedEnvVar = "TESTS_SELECT_SUBSET_SEED"

	// subsetChangedLimit caps how many changed tests the subset selector keeps on top of
	// its hash-selected fraction. Unlike selection.EnvVar it carries no count of its own,
	// and a PR that edits hundreds of test dirs must not turn the subset cells back into
	// a full run.
	subsetChangedLimit = 50
)

// subsetSelector decides, per subtest, whether it runs under TESTS_SELECT_SUBSET_PCT.
// A subtest runs if it is a changed test on this branch (always kept, reusing the same
// change detection as DATABRICKS_TEST_SELECT_CHANGED), or if its seeded hash falls
// under the percentage. The decision is independent per subtest, so changed tests run
// on top of the hash-selected subset rather than displacing anything.
type subsetSelector struct {
	enabled bool
	pct     int
	seed    string

	// changed maps test dir -> variant filters for added/modified tests (nil filters
	// means all variants of that dir are changed). Same shape as changedTests.
	changed map[string][]string
}

// newSubsetSelector reads the subset env vars. Subsetting is disabled in update mode
// and under -forcerun so that every output is regenerated and forced runs are honored.
// The caller assigns .changed (the changed tests to always keep) so that the change
// detection is shared with DATABRICKS_TEST_SELECT_CHANGED and runs at most once.
func newSubsetSelector(t *testing.T, overwrite, forcerun bool) subsetSelector {
	raw := os.Getenv(SubsetPctEnvVar)
	if raw == "" || overwrite || forcerun {
		return subsetSelector{}
	}

	pct, err := strconv.Atoi(raw)
	if err != nil || pct < 0 || pct > 100 {
		t.Fatalf("Invalid %s=%q, expected an integer 0..100", SubsetPctEnvVar, raw)
	}

	seed := os.Getenv(SubsetSeedEnvVar)
	if seed == "" {
		// Reproduce this run's subset by re-running with the logged seed.
		seed = uuid.NewString()
		t.Logf("%s not set, using random seed %q", SubsetSeedEnvVar, seed)
	}

	return subsetSelector{
		enabled: true,
		pct:     pct,
		seed:    seed,
	}
}

// skipReason returns a non-empty skip message if the subtest identified by dir and
// envset should be skipped by subsetting, or "" if it should run.
func (s subsetSelector) skipReason(dir string, envset []string) string {
	if !s.enabled {
		return ""
	}
	if s.isChanged(dir, envset) {
		return ""
	}
	id := dir
	if len(envset) > 0 {
		id += "/" + strings.Join(envset, "/")
	}
	if hashPercent(s.seed, id) < s.pct {
		return ""
	}
	return "Skipped by " + SubsetPctEnvVar
}

// isChanged reports whether the subtest belongs to a changed test dir. For an
// invariant dir re-enabled by a specific config change, only the matching variants
// count as changed.
func (s subsetSelector) isChanged(dir string, envset []string) bool {
	filters, ok := s.changed[dir]
	if !ok {
		return false
	}
	if filters == nil {
		return true
	}
	return envMatchesFilters(envset, filters)
}

// envMatchesFilters reports whether envset satisfies every KEY=value filter. A filter
// matches unless its key is present in envset with a different value. Filters that share a
// key are alternatives: the environment has one value per key, so requiring each of them
// separately would match nothing, while a selection that names several values of a key
// (e.g. one INPUT_CONFIG per changed invariant config) means any of them.
func envMatchesFilters(envset, filters []string) bool {
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

// hashPercent maps (seed, id) deterministically to 0..99. A subtest runs when this is
// below the percentage, so pct=100 runs everything and pct=0 runs nothing.
func hashPercent(seed, id string) int {
	h := fnv.New64a()
	h.Write([]byte(seed))
	h.Write([]byte{0})
	h.Write([]byte(id))
	return int(h.Sum64() % 100)
}
