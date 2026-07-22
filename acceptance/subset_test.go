package acceptance_test

import (
	"hash/fnv"
	"os"
	"strconv"
	"strings"
	"testing"
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
	SubsetSeedEnvVar = "TESTS_SELECT_SUBSET_SEED"
)

// subsetSelector decides, per subtest, whether it runs under TESTS_SELECT_SUBSET_PCT.
// A subtest runs if it is an added/modified test on this branch (always kept, reusing
// the same change detection as SkipLocalWithChanged), or if its seeded hash falls
// under the percentage. Added/modified tests thus count against the percentage only in
// the sense that they occupy the run; they are never dropped by it.
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
// When enabled, added/modified tests are detected once (reusing the change detection
// behind SkipLocalWithChanged) so they are always kept.
func newSubsetSelector(t *testing.T, overwrite, forcerun bool, testDirs map[string]bool) subsetSelector {
	raw := os.Getenv(SubsetPctEnvVar)
	if raw == "" || overwrite || forcerun {
		return subsetSelector{}
	}

	pct, err := strconv.Atoi(raw)
	if err != nil || pct < 0 || pct > 100 {
		t.Fatalf("Invalid %s=%q, expected an integer 0..100", SubsetPctEnvVar, raw)
	}

	return subsetSelector{
		enabled: true,
		pct:     pct,
		seed:    os.Getenv(SubsetSeedEnvVar),
		changed: selectChangedLocalTests(testDirs),
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

// isChanged reports whether the subtest belongs to an added/modified test dir. For an
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

// envMatchesFilters reports whether envset satisfies every KEY=value filter, mirroring
// the skip semantics of checkEnvFilters: a filter matches unless its key is present in
// envset with a different value.
func envMatchesFilters(envset, filters []string) bool {
	envMap := make(map[string]string, len(envset))
	for _, kv := range envset {
		key, value, _ := strings.Cut(kv, "=")
		envMap[key] = value
	}
	for _, filter := range filters {
		key, expected, _ := strings.Cut(filter, "=")
		if actual, ok := envMap[key]; ok && actual != expected {
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
