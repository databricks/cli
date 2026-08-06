package localenv

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// codes extracts the warning codes in order for concise assertions.
func codes(ws []Warning) []string {
	out := make([]string, 0, len(ws))
	for _, w := range ws {
		out = append(out, w.Code)
	}
	return out
}

func TestDetectMergeWarningsGreenfieldAndEmpty(t *testing.T) {
	c := Constraints{RequiresPython: "==3.12.*", DatabricksConnect: "databricks-connect~=18.0.0"}
	// No pre-existing file: nothing of the user's to override.
	assert.Nil(t, detectMergeWarnings(nil, c))
	assert.Nil(t, detectMergeWarnings([]byte{}, c))
	// Unparseable TOML is best-effort: no warnings, no panic.
	assert.Nil(t, detectMergeWarnings([]byte("this is : not valid toml ["), c))
}

func TestDetectMergeWarningsOverriddenPins(t *testing.T) {
	user := []byte(`[project]
name = "demo"
requires-python = ">=3.10"

[dependency-groups]
dev = ["databricks-connect~=16.1.0"]
`)
	c := Constraints{RequiresPython: "==3.12.*", DatabricksConnect: "databricks-connect~=18.0.0"}
	got := detectMergeWarnings(user, c)
	assert.Equal(t, []string{WarnRequiresPythonOverridden, WarnDBConnectPinOverridden}, codes(got))
}

func TestDetectMergeWarningsNoOverrideWhenMatching(t *testing.T) {
	// User already matches the env pins exactly — the merge is a no-op, so no
	// override warnings.
	user := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = ["databricks-connect~=18.0.0"]
`)
	c := Constraints{RequiresPython: "==3.12.*", DatabricksConnect: "databricks-connect~=18.0.0"}
	assert.Empty(t, detectMergeWarnings(user, c))
}

func TestDetectMergeWarningsConstraintsOnlyIgnoresDBConnect(t *testing.T) {
	// In constraints-only mode c.DatabricksConnect is empty; the user's dev pin is
	// left untouched, so it must not produce an override warning.
	user := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = ["databricks-connect~=16.1.0"]
`)
	c := Constraints{RequiresPython: "==3.12.*", DatabricksConnect: ""}
	assert.Empty(t, detectMergeWarnings(user, c))
}

func TestDetectMergeWarningsUserConstraintConflict(t *testing.T) {
	user := []byte(`[project]
requires-python = "==3.12.*"
dependencies = ["pyarrow==21.0.0", "requests>=2.0"]
`)
	// The env constraints use the upper-bound shape the published artifacts actually
	// publish ("pyarrow<19", "pandas<3" — see acceptance/localenv/*/test.toml), not a
	// shape chosen to suit the detector. The user's ==21.0.0 is provably above the
	// ceiling. requests is unconstrained, so it yields nothing.
	c := Constraints{
		RequiresPython: "==3.12.*",
		ConstraintDeps: []string{"pyarrow<19", "pandas<3"},
	}
	got := detectMergeWarnings(user, c)
	assert.Equal(t, []string{WarnUserConstraintConflict}, codes(got))
	assert.Contains(t, got[0].Message, "pyarrow")
}

func TestDetectMergeWarningsNoConflictWhenCompatible(t *testing.T) {
	user := []byte(`[project]
requires-python = "==3.12.*"
dependencies = ["pyarrow==18.1.0", "pandas==2.2.0"]
`)
	// Both pins sit inside the env's ceilings, so there is nothing to report.
	c := Constraints{RequiresPython: "==3.12.*", ConstraintDeps: []string{"pyarrow<19", "pandas<3"}}
	assert.Empty(t, detectMergeWarnings(user, c))
}

func TestDetectMergeWarningsIncludeGroupDoesNotSuppress(t *testing.T) {
	// A PEP 735 {include-group = ...} table is a legal dev-group entry that uv
	// supports. It must not fail the decode and take every unrelated warning with
	// it: the requires-python override and the pyarrow conflict below do not depend
	// on the dev group at all, and MergeManaged rewrites all three regions anyway.
	user := []byte(`[project]
requires-python = ">=3.9"
dependencies = ["pyarrow==19.0.0"]

[dependency-groups]
dev = ["databricks-connect~=16.1.0", {include-group = "test"}]
test = []
`)
	c := Constraints{
		RequiresPython:    "==3.12.*",
		DatabricksConnect: "databricks-connect~=17.2.0",
		ConstraintDeps:    []string{"pyarrow~=21.0.0"},
	}
	assert.Equal(t, []string{
		WarnRequiresPythonOverridden,
		WarnDBConnectPinOverridden,
		WarnUserConstraintConflict,
	}, codes(detectMergeWarnings(user, c)))
}

func TestDetectMergeWarningsIncludeGroupIsDuplicatedNotOverridden(t *testing.T) {
	// MergeManaged only rewrites a pin sitting in dev's own array. A pin reached
	// through an include-group is left alone and the env's pin is inserted alongside
	// it, so the merged file carries two pins for one package and uv cannot resolve.
	// That is a different condition from an override and needs its own code —
	// reporting "was replaced" here would state something untrue and hide a hard
	// resolution failure behind a reassuring advisory.
	c := Constraints{RequiresPython: "==3.12.*", DatabricksConnect: "databricks-connect==17.0.0"}

	indirect := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = [{include-group = "spark"}]
spark = ["databricks-connect==16.1.0"]
`)
	got := detectMergeWarnings(indirect, c)
	assert.Equal(t, []string{WarnDBConnectPinDuplicated}, codes(got))
	// The message must not claim a replacement that did not happen.
	assert.NotContains(t, got[0].Message, "was replaced")

	// A pin directly in dev *is* rewritten in place, so that stays an override.
	direct := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = ["databricks-connect==16.1.0"]
`)
	assert.Equal(t, []string{WarnDBConnectPinOverridden}, codes(detectMergeWarnings(direct, c)))

	// With pins both in dev and behind an include-group, the merge rewrites dev's,
	// so the direct case wins and only one warning is reported.
	both := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = ["databricks-connect==16.1.0", {include-group = "spark"}]
spark = ["databricks-connect==15.0.0"]
`)
	assert.Equal(t, []string{WarnDBConnectPinOverridden}, codes(detectMergeWarnings(both, c)))

	// An include-group cycle must terminate rather than recurse forever, and the pin
	// behind it is still found.
	cyclic := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = [{include-group = "a"}]
a = [{include-group = "dev"}, "databricks-connect==16.1.0"]
`)
	assert.Equal(t, []string{WarnDBConnectPinDuplicated}, codes(detectMergeWarnings(cyclic, c)))

	// PEP 735 normalizes group names the same way PEP 503 normalizes package names.
	renamed := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = [{include-group = "My_Spark.Group"}]
"my-spark-group" = ["databricks-connect==16.1.0"]
`)
	assert.Equal(t, []string{WarnDBConnectPinDuplicated}, codes(detectMergeWarnings(renamed, c)))

	// An included pin that already matches the env needs no reconciliation.
	matching := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = [{include-group = "spark"}]
spark = ["databricks-connect==17.0.0"]
`)
	assert.Empty(t, detectMergeWarnings(matching, c))

	// A dangling reference is not an error; there is simply no pin to compare.
	dangling := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = [{include-group = "absent"}]
`)
	assert.Empty(t, detectMergeWarnings(dangling, c))
}

func TestDetectMergeWarningsConflictInDependencyGroup(t *testing.T) {
	// uv applies constraint-dependencies to the whole resolution, so a pin in the dev
	// group breaks `uv sync` exactly like one in [project].dependencies. The dev group
	// is where this command's audience keeps its pins, so it must be scanned too.
	user := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = ["pyarrow==17.0.0"]
`)
	c := Constraints{RequiresPython: "==3.12.*", ConstraintDeps: []string{"pyarrow<19"}}
	assert.Empty(t, detectMergeWarnings(user, c), "17.0.0 satisfies <19 — no conflict")

	conflicting := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = ["pyarrow==21.0.0"]
`)
	assert.Equal(t, []string{WarnUserConstraintConflict},
		codes(detectMergeWarnings(conflicting, c)), "21.0.0 is outside <19")

	// A pin behind an include-group is part of the resolution too.
	included := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = [{include-group = "data"}]
data = ["pyarrow==21.0.0"]
`)
	assert.Equal(t, []string{WarnUserConstraintConflict}, codes(detectMergeWarnings(included, c)))
}

func TestConstraintConflictsDuplicateEnvEntriesAreOrderIndependent(t *testing.T) {
	// Constraint entries for one package compose as a conjunction. Joining them
	// yields a multi-clause spec, which is an unknown range — so neither ordering
	// decides a conflict against just one of the clauses.
	user := []string{"pyarrow==19.0.0"}
	assert.Empty(t, constraintConflicts(user, []string{"pyarrow<21", "pyarrow>=20"}))
	assert.Empty(t, constraintConflicts(user, []string{"pyarrow>=20", "pyarrow<21"}))
}

func TestSplitDepSpecExtrasAndMarkers(t *testing.T) {
	// Extras never narrow the version range, so they are stripped and the pin
	// underneath is still compared.
	name, spec, ok := splitDepSpec("pyarrow[compute,parquet]==17.0.0")
	assert.True(t, ok)
	assert.Equal(t, "pyarrow", name)
	assert.Equal(t, "==17.0.0", spec)

	// A marker makes the requirement conditional on the resolving interpreter,
	// which we do not evaluate — so it is skipped rather than compared, with or
	// without extras in front of it.
	_, _, ok = splitDepSpec(`pyarrow==17.0.0; python_version < "3.12"`)
	assert.False(t, ok)
	_, _, ok = splitDepSpec(`pyarrow[compute]==17.0.0; python_version < "3.12"`)
	assert.False(t, ok)

	// PEP 503 normalization applies to the name on both sides of the lookup.
	name, spec, ok = splitDepSpec("PyArrow_Extra ==1.0")
	assert.True(t, ok)
	assert.Equal(t, "pyarrow-extra", name)
	assert.Equal(t, "==1.0", spec)
}

func TestDetectMergeWarningsConflictThroughExtras(t *testing.T) {
	user := []byte(`[project]
dependencies = ["pyarrow[compute]==17.0.0"]
`)
	c := Constraints{ConstraintDeps: []string{"pyarrow~=21.0.0"}}
	got := detectMergeWarnings(user, c)
	assert.Equal(t, []string{WarnUserConstraintConflict}, codes(got))
}

func TestRangesDisjoint(t *testing.T) {
	cases := []struct {
		user, env string
		disjoint  bool
		why       string
	}{
		{"==17.0.0", "~=21.0.0", true, "exact pin below the compatible range"},
		{"==21.0.3", "~=21.0.0", false, "exact pin inside ~=21.0.0 == >=21.0.0,==21.0.*"},
		{"==21.5.0", "~=21.0.0", true, "~=21.0.0 fixes 21.0.*, so 21.5.0 is outside it"},
		{"==22.0.0", "~=21.0.0", true, "exact pin above the compatible upper bound"},
		{"==1.0", "==2.0", true, "two different exact pins"},
		{"==2.0", "==2.0", false, "identical exact pins"},
		{"==2", "==2.0.0", false, "trailing zeros denote the same release"},
		{">=2.0", "==2.5", false, "2.5 is above the floor — overlapping"},
		{"!=2.0", "==2.0", false, "!= is not modeled — never a conflict"},
		{">=2.0,<3.0", "==5.0", false, "multi-clause range is treated as unknown"},
		{"==2.*", "==2.0", false, "wildcards are unparsed — no conflict"},
		{"", "~=21.0.0", false, "no user spec — nothing to compare"},

		// Opposite-direction bounds. This is the shape the published constraint
		// artifacts use (pyarrow<19, pandas<3), so failing to decide it would make the
		// warning miss the common case and under-report the metric silently.
		{">=20", "<19", true, "floor above the ceiling"},
		{">=4", "<3", true, "floor above the ceiling, single segment"},
		{">=19", "<20", false, "ranges overlap on [19, 20)"},
		{">2.0", "<2.0", true, "strict bounds meeting at one excluded point"},
		{">=2.0", "<2.0", true, "inclusive floor at an exclusive ceiling"},
		{">=2.0", "<=2.0", false, "both include the single shared version 2.0"},
		{">2.0", "<=2.0", true, "the shared endpoint is excluded by the floor"},
		{"<19", "<3", false, "same direction always shares the low end"},
		{">=20", ">=3", false, "same direction always shares the high end"},

		// ~= against a bound or another ~=. The ceiling depends on how many segments
		// the base has: ~=21.0 expands to ">=21.0, ==21.*" -> [21.0, 22.0), whereas
		// ~=21.0.0 expands to ">=21.0.0, ==21.0.*" -> [21.0.0, 21.1.0).
		{"~=17.0", "~=21.0", true, "[17.0, 18.0) is entirely below [21.0, 22.0)"},
		{"~=21.0", "~=21.1", false, "[21.1, 22.0) is a subset of [21.0, 22.0)"},
		{"~=21.0.0", "~=21.1.0", true, "[21.0.0, 21.1.0) ends where [21.1.0, 21.2.0) begins"},
		{"~=21.0", "~=21.0", false, "identical compatible ranges"},
		{"~=17.0", "<19", false, "[17.0, 18.0) lies below the ceiling 19"},
		{"~=21.0", "<19", true, "floor 21.0 is above the ceiling 19"},
		{"~=21.0", ">=22", true, "ceiling 22.0 is exclusive, so it meets the floor 22"},
		{"~=21.0", ">=21.5", false, "21.5 is inside [21.0, 22.0)"},
		// PEP 440 requires two release segments after "~=", so a single-segment base
		// has no defined range. It must stay undecidable rather than being read as a
		// proof of disjointness: 2.0 and 2.31.0 both plainly satisfy any reading of ~=2.
		{"~=2", "==2.0", false, "single-segment ~= is malformed — undecidable, not disjoint"},
		{"==2.0", "~=2", false, "same, with the malformed base on the env side"},
		{"~=2", "==2.31.0", false, "single-segment ~= never decides a conflict"},
		{"~=2.0", "==2.31.0", false, "valid ~=2.0 admits 2.31.0 (>=2.0, ==2.*)"},
		{"~=2.1", "==2.0.5", true, "valid ~=2.1 excludes 2.0.5 (below the floor)"},
	}
	for _, tc := range cases {
		assert.Equalf(t, tc.disjoint, rangesDisjoint(tc.user, tc.env),
			"rangesDisjoint(%q,%q): %s", tc.user, tc.env, tc.why)
	}
}
