package acceptance_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSubsetDisabled(t *testing.T) {
	var s subsetSelector
	assert.Empty(t, s.skipReason("bundle/foo", nil))
	assert.Empty(t, s.skipReason("bundle/foo", []string{"DATABRICKS_BUNDLE_ENGINE=direct"}))
}

func TestSubsetPctBoundaries(t *testing.T) {
	all := subsetSelector{enabled: true, pct: 100, seed: "seed"}
	none := subsetSelector{enabled: true, pct: 0, seed: "seed"}

	for _, dir := range []string{"bundle/a", "bundle/b", "cmd/c"} {
		assert.Empty(t, all.skipReason(dir, nil), "pct=100 must run everything")
		assert.NotEmpty(t, none.skipReason(dir, nil), "pct=0 must skip everything")
	}
}

func TestSubsetChangedAlwaysRuns(t *testing.T) {
	// pct=0 would skip everything, but a changed dir is always kept.
	s := subsetSelector{
		enabled: true,
		pct:     0,
		seed:    "seed",
		changed: map[string][]string{"bundle/changed": nil},
	}
	assert.Empty(t, s.skipReason("bundle/changed", []string{"DATABRICKS_BUNDLE_ENGINE=direct"}))
	assert.NotEmpty(t, s.skipReason("bundle/other", nil))
}

func TestSubsetChangedVariantFilter(t *testing.T) {
	// A changed dir restricted to a specific config only keeps matching variants.
	s := subsetSelector{
		enabled: true,
		pct:     0,
		seed:    "seed",
		changed: map[string][]string{"bundle/invariant/x": {"INPUT_CONFIG=job.yml.tmpl"}},
	}
	assert.Empty(t, s.skipReason("bundle/invariant/x", []string{"INPUT_CONFIG=job.yml.tmpl"}))
	assert.NotEmpty(t, s.skipReason("bundle/invariant/x", []string{"INPUT_CONFIG=pipeline.yml.tmpl"}))
}

func TestSubsetChangedVariantFilterAlternatives(t *testing.T) {
	// Filters naming the same key are alternatives, so the variants of both configs are
	// kept rather than neither.
	s := subsetSelector{
		enabled: true,
		pct:     0,
		seed:    "seed",
		changed: map[string][]string{"bundle/invariant/x": {"INPUT_CONFIG=job.yml.tmpl", "INPUT_CONFIG=pipeline.yml.tmpl"}},
	}
	assert.Empty(t, s.skipReason("bundle/invariant/x", []string{"INPUT_CONFIG=job.yml.tmpl"}))
	assert.Empty(t, s.skipReason("bundle/invariant/x", []string{"INPUT_CONFIG=pipeline.yml.tmpl"}))
	assert.NotEmpty(t, s.skipReason("bundle/invariant/x", []string{"INPUT_CONFIG=other.yml.tmpl"}))
}

func TestSubsetSeedDeterministic(t *testing.T) {
	// Same seed selects the same subset; the decision does not depend on ordering.
	a := subsetSelector{enabled: true, pct: 50, seed: "abc"}
	b := subsetSelector{enabled: true, pct: 50, seed: "abc"}
	for _, dir := range []string{"bundle/a", "bundle/b", "bundle/c", "bundle/d"} {
		assert.Equal(t, a.skipReason(dir, nil), b.skipReason(dir, nil))
	}
}

func TestSubsetSeedChangesSelection(t *testing.T) {
	// Different seeds should produce different selections across a set of dirs.
	var dirs []string
	for i := range 100 {
		dirs = append(dirs, "bundle/dir"+string(rune('a'+i%26))+string(rune('0'+i/26)))
	}
	s1 := subsetSelector{enabled: true, pct: 50, seed: "seed-1"}
	s2 := subsetSelector{enabled: true, pct: 50, seed: "seed-2"}
	diff := 0
	for _, dir := range dirs {
		if (s1.skipReason(dir, nil) == "") != (s2.skipReason(dir, nil) == "") {
			diff++
		}
	}
	assert.NotZero(t, diff, "different seeds should select different subsets")
}

func TestSubsetVariantsIndependent(t *testing.T) {
	// Two variants of one dir are decided independently: at least one differs from the
	// other across a spread of dirs (they are not forced to the same outcome).
	s := subsetSelector{enabled: true, pct: 50, seed: "seed"}
	sawDifferent := false
	for i := range 50 {
		dir := "bundle/d" + string(rune('a'+i%26))
		a := s.skipReason(dir, []string{"DATABRICKS_BUNDLE_ENGINE=direct"}) == ""
		b := s.skipReason(dir, []string{"DATABRICKS_BUNDLE_ENGINE=terraform"}) == ""
		if a != b {
			sawDifferent = true
			break
		}
	}
	assert.True(t, sawDifferent, "variants of the same dir should be decided independently")
}

func TestSubsetApproximatesPct(t *testing.T) {
	// Over many subtests the selected fraction should be close to pct.
	s := subsetSelector{enabled: true, pct: 30, seed: "seed"}
	run := 0
	total := 1000
	for i := range total {
		id := "bundle/pkg" + string(rune('a'+i%26)) + "/case" + string(rune('0'+i/26%10))
		if s.skipReason(id, []string{"V=" + string(rune('a'+i%7))}) == "" {
			run++
		}
	}
	// Allow a generous margin; this guards against gross bias, not exact proportion.
	assert.Greater(t, run, 200)
	assert.Less(t, run, 400)
}
