package localenv

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// detectWarnings runs the detector the way mergePlan does, taking the replaced pin
// from the merge itself rather than a hand-supplied value. Tests must not assert
// against a different notion of what the merge rewrites than production uses — that
// divergence is the bug this wiring exists to prevent.
func detectWarnings(userPyproject []byte, c Constraints) []Warning {
	return detectMergeWarnings(userPyproject, c, replacedDBConnectPin(userPyproject, c))
}

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
	assert.Nil(t, detectWarnings(nil, c))
	assert.Nil(t, detectWarnings([]byte{}, c))
	// Unparseable TOML is best-effort: no warnings, no panic.
	assert.Nil(t, detectWarnings([]byte("this is : not valid toml ["), c))
}

func TestDetectMergeWarningsOverriddenPins(t *testing.T) {
	user := []byte(`[project]
name = "demo"
requires-python = ">=3.10"

[dependency-groups]
dev = ["databricks-connect~=16.1.0"]
`)
	c := Constraints{RequiresPython: "==3.12.*", DatabricksConnect: "databricks-connect~=18.0.0"}
	got := detectWarnings(user, c)
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
	assert.Empty(t, detectWarnings(user, c))
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
	assert.Empty(t, detectWarnings(user, c))
}

func TestDetectMergeWarningsUserConstraintConflict(t *testing.T) {
	user := []byte(`[project]
requires-python = "==3.12.*"
dependencies = ["pyarrow==21.0.0", "requests>=2.0"]
`)
	// The bound shape used here matches the acceptance fixtures, which is why it is
	// worth covering — but note the published artifacts are entirely "~=" (every entry
	// of serverless-v4 and -v5 at the time of writing), so "~=" is the shape that
	// actually has to work. TestRangesDisjoint covers the "~=" pairs directly.
	c := Constraints{
		RequiresPython: "==3.12.*",
		ConstraintDeps: []string{"pyarrow<19", "pandas<3"},
	}
	got := detectWarnings(user, c)
	assert.Equal(t, []string{WarnUserConstraintConflict}, codes(got))
	assert.Contains(t, got[0].Message, "pyarrow")

	// The same conflict through the "~=" shape the artifacts really publish.
	tilde := Constraints{RequiresPython: "==3.12.*", ConstraintDeps: []string{"pyarrow~=18.1.0"}}
	assert.Equal(t, []string{WarnUserConstraintConflict}, codes(detectWarnings(user, tilde)))
}

func TestDetectMergeWarningsNoConflictWhenCompatible(t *testing.T) {
	user := []byte(`[project]
requires-python = "==3.12.*"
dependencies = ["pyarrow==18.1.0", "pandas==2.2.0"]
`)
	// Both pins sit inside the env's ceilings, so there is nothing to report.
	c := Constraints{RequiresPython: "==3.12.*", ConstraintDeps: []string{"pyarrow<19", "pandas<3"}}
	assert.Empty(t, detectWarnings(user, c))
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
	}, codes(detectWarnings(user, c)))
}

func TestDetectMergeWarningsIncludeGroupIsDuplicatedNotOverridden(t *testing.T) {
	// MergeManaged only rewrites a pin sitting in dev's own array. A pin reached
	// through an include-group is left alone and the env's pin is inserted alongside
	// it, so the merged file carries two pins for one package and uv cannot resolve.
	// That is a different condition from an override and needs its own code —
	// reporting "is replaced" here would state something untrue and hide a hard
	// resolution failure behind a reassuring advisory.
	c := Constraints{RequiresPython: "==3.12.*", DatabricksConnect: "databricks-connect==17.0.0"}

	indirect := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = [{include-group = "spark"}]
spark = ["databricks-connect==16.1.0"]
`)
	got := detectWarnings(indirect, c)
	assert.Equal(t, []string{WarnDBConnectPinDuplicated}, codes(got))
	// The message must not claim a replacement that did not happen.
	assert.NotContains(t, got[0].Message, "is replaced")

	// A pin directly in dev *is* rewritten in place, so that stays an override.
	direct := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = ["databricks-connect==16.1.0"]
`)
	assert.Equal(t, []string{WarnDBConnectPinOverridden}, codes(detectWarnings(direct, c)))

	// With pins both in dev and behind an include-group, the merge rewrites dev's and
	// leaves the included one, so both conditions hold and both are reported. Only
	// flagging the override would go silent on the pin that still breaks resolution.
	both := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = ["databricks-connect==16.1.0", {include-group = "spark"}]
spark = ["databricks-connect==15.0.0"]
`)
	assert.Equal(t, []string{WarnDBConnectPinOverridden, WarnDBConnectPinDuplicated},
		codes(detectWarnings(both, c)))

	// The merge is idempotent, but the included pin it does not rewrite is not fixed
	// by re-running: the duplicate warning must persist for as long as the two pins do,
	// or the user loses the only signal about a project uv cannot resolve.
	merged, _, err := MergeManaged(both, c)
	require.NoError(t, err)
	assert.Equal(t, []string{WarnDBConnectPinDuplicated}, codes(detectWarnings(merged, c)))

	// An include-group cycle must terminate rather than recurse forever, and the pin
	// behind it is still found.
	cyclic := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = [{include-group = "a"}]
a = [{include-group = "dev"}, "databricks-connect==16.1.0"]
`)
	assert.Equal(t, []string{WarnDBConnectPinDuplicated}, codes(detectWarnings(cyclic, c)))

	// PEP 735 normalizes group names the same way PEP 503 normalizes package names.
	renamed := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = [{include-group = "My_Spark.Group"}]
"my-spark-group" = ["databricks-connect==16.1.0"]
`)
	assert.Equal(t, []string{WarnDBConnectPinDuplicated}, codes(detectWarnings(renamed, c)))

	// An included pin that already matches the env needs no reconciliation.
	matching := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = [{include-group = "spark"}]
spark = ["databricks-connect==17.0.0"]
`)
	assert.Empty(t, detectWarnings(matching, c))

	// A dangling reference is not an error; there is simply no pin to compare.
	dangling := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = [{include-group = "absent"}]
`)
	assert.Empty(t, detectWarnings(dangling, c))
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
	assert.Empty(t, detectWarnings(user, c), "17.0.0 satisfies <19 — no conflict")

	conflicting := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = ["pyarrow==21.0.0"]
`)
	assert.Equal(t, []string{WarnUserConstraintConflict},
		codes(detectWarnings(conflicting, c)), "21.0.0 is outside <19")

	// A pin behind an include-group is part of the resolution too.
	included := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = [{include-group = "data"}]
data = ["pyarrow==21.0.0"]
`)
	assert.Equal(t, []string{WarnUserConstraintConflict}, codes(detectWarnings(included, c)))
}

func TestDetectMergeWarningsSurvivesUnrelatedTOMLShapes(t *testing.T) {
	// BurntSushi reports a per-key type mismatch as a whole-document error. Each shape
	// below is legal in a real pyproject.toml but does not fit a stricter struct, and
	// none of them is read by the requires-python check — so dropping every warning on
	// one of them would silently hide an override the merge does perform.
	c := Constraints{RequiresPython: "==3.12.*", DatabricksConnect: "databricks-connect~=18.0.0"}
	for name, body := range map[string]string{
		// PDM/mkdocs style: a dependency group declared as a sub-table.
		"group sub-table": `[project]
requires-python = ">=3.9"

[dependency-groups]
dev = ["databricks-connect~=16.1.0"]

[dependency-groups.docs]
mkdocs = "*"
`,
		// PEP 621 permits a dependency to be a table in some tool dialects.
		"table dependency": `[project]
requires-python = ">=3.9"
dependencies = ["pyarrow==21.0.0", {name = "x"}]

[dependency-groups]
dev = ["databricks-connect~=16.1.0"]
`,
		"group is a string": `[project]
requires-python = ">=3.9"

[dependency-groups]
dev = "oops"
`,
	} {
		got := codes(detectWarnings([]byte(body), c))
		assert.Contains(t, got, WarnRequiresPythonOverridden, "%s: requires-python override must survive", name)
	}

	// A genuine syntax error still yields nothing: there are no fields to compare.
	assert.Empty(t, detectWarnings([]byte("this is : not valid toml ["), c))
}

func TestDetectMergeWarningsScansEveryGroupUVLocks(t *testing.T) {
	// uv applies constraint-dependencies to the whole resolution and locks every
	// declared group, so a conflicting pin outside dev fails `uv sync` with
	// "requirements are unsatisfiable" just as a [project] one does. Scanning only dev
	// would stay silent on exactly that failure.
	c := Constraints{RequiresPython: "==3.12.*", ConstraintDeps: []string{"pyarrow<19"}}

	nonDevGroup := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
qa = ["pyarrow==21.0.0"]
`)
	assert.Equal(t, []string{WarnUserConstraintConflict}, codes(detectWarnings(nonDevGroup, c)),
		"a pin in a group uv locks conflicts even though the group is not dev")

	optionalExtra := []byte(`[project]
requires-python = "==3.12.*"

[project.optional-dependencies]
extra = ["pyarrow==21.0.0"]
`)
	assert.Equal(t, []string{WarnUserConstraintConflict}, codes(detectWarnings(optionalExtra, c)),
		"uv resolves extras alongside the base dependencies")
}

func TestDetectMergeWarningsReportsOneConflictPerRequirement(t *testing.T) {
	// The same pin may be declared in several places uv locks. It is one conflict to
	// fix, and repeating it would inflate the histogram consumers build from the codes.
	user := []byte(`[project]
requires-python = "==3.12.*"
dependencies = ["pyarrow==21.0.0"]

[dependency-groups]
dev = ["pyarrow==21.0.0", {include-group = "g"}]
g = ["pyarrow==21.0.0"]
`)
	c := Constraints{RequiresPython: "==3.12.*", ConstraintDeps: []string{"pyarrow<19"}}
	assert.Equal(t, []string{WarnUserConstraintConflict}, codes(detectWarnings(user, c)))

	// Two genuinely different conflicting pins for one package are each reported.
	two := []byte(`[project]
requires-python = "==3.12.*"
dependencies = ["pyarrow==21.0.0"]

[dependency-groups]
dev = ["pyarrow==22.0.0"]
`)
	assert.Equal(t, []string{WarnUserConstraintConflict, WarnUserConstraintConflict},
		codes(detectWarnings(two, c)))
}

func TestDetectMergeWarningsIsDeterministic(t *testing.T) {
	// Group names are read from a Go map. Two keys that normalize to the same PEP 735
	// group, or several groups holding conflicting pins, must not let map iteration
	// order decide which warning is reported.
	c := Constraints{RequiresPython: "==3.12.*", ConstraintDeps: []string{"pyarrow<19", "numpy<2"}}
	user := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = ["pyarrow==21.0.0"]
Dev = ["numpy==9.9.9"]
`)
	want := codes(detectWarnings(user, c))
	assert.Len(t, want, 2, "both groups are locked by uv, so both conflicts are reported")
	for range 200 {
		assert.Equal(t, want, codes(detectWarnings(user, c)))
	}
}

func TestDetectMergeWarningsNonLiteralDevGroupIsNotAnOverride(t *testing.T) {
	// MergeManaged finds the dev array with devKeyRe (`^\s*dev\s*=`), so a group spelled
	// "Dev" is never rewritten: the merge adds its own dev key and the user's pin stays.
	// Claiming an override here would describe a replacement that did not happen, and
	// staying silent would hide a file uv rejects outright.
	c := Constraints{RequiresPython: "==3.12.*", DatabricksConnect: "databricks-connect==17.0.0"}
	user := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
Dev = ["databricks-connect==16.1.0"]
`)
	got := detectWarnings(user, c)
	assert.Equal(t, []string{WarnDBConnectPinDuplicated}, codes(got))
	assert.NotContains(t, got[0].Message, "is replaced")

	merged, _, err := MergeManaged(user, c)
	require.NoError(t, err)
	assert.Contains(t, string(merged), `"databricks-connect==16.1.0"`, "the user's pin is retained, not replaced")
}

func TestDBConnectOverrideFollowsWhatTheMergeRewrites(t *testing.T) {
	// The override warning must be driven by the merge's own answer, not by a second
	// implementation of "is this pin in the dev array". MergeManaged rewrites only
	// double-quoted elements, so each spelling below is left in place beside the managed
	// pin — reporting it as replaced would be a false claim about the user's file.
	c := Constraints{RequiresPython: "==3.12.*", DatabricksConnect: "databricks-connect==17.0.0"}
	for name, body := range map[string]string{
		// A TOML literal string is a string, but not one replaceDbconnectElement matches.
		"single-quoted element": `[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = ['databricks-connect==16.1.0']
`,
		// A dotted key defines the same table but no line matches devKeyRe.
		"top-level dotted key": `dependency-groups.dev = ["databricks-connect==16.1.0"]

[project]
requires-python = "==3.12.*"
`,
		// devKeyRe is literal, so a normalization-equal key is a different array.
		"capitalized group": `[project]
requires-python = "==3.12.*"

[dependency-groups]
Dev = ["databricks-connect==16.1.0"]
`,
	} {
		got := detectWarnings([]byte(body), c)
		assert.Equal(t, []string{WarnDBConnectPinDuplicated}, codes(got), name)
		assert.NotContains(t, got[0].Message, "is replaced", name)

		// The merged file really does carry both pins, which is what the code reports.
		merged, _, err := MergeManaged([]byte(body), c)
		require.NoError(t, err, name)
		assert.Contains(t, string(merged), "databricks-connect==16.1.0", name)
		assert.Contains(t, string(merged), "databricks-connect==17.0.0", name)
	}
}

func TestDBConnectDuplicateOnlyWhenRangesCannotBothHold(t *testing.T) {
	// Two pins for one package are only a problem when nothing satisfies both. uv
	// resolves an overlapping pair without complaint, so there is nothing to reconcile
	// and no warning to give.
	c := Constraints{RequiresPython: "==3.12.*", DatabricksConnect: "databricks-connect~=17.2.0"}

	overlapping := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = [{include-group = "spark"}]
spark = ["databricks-connect>=16"]
`)
	assert.Empty(t, detectWarnings(overlapping, c), ">=16 and ~=17.2.0 both hold at 17.2.x")

	disjoint := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = [{include-group = "spark"}]
spark = ["databricks-connect==15.0.0"]
`)
	assert.Equal(t, []string{WarnDBConnectPinDuplicated}, codes(detectWarnings(disjoint, c)),
		"==15.0.0 is outside ~=17.2.0, so uv cannot resolve")
}

func TestDBConnectWarningsSecondDirectPinIsRetained(t *testing.T) {
	// MergeManaged replaces only the first databricks-connect element in the dev array,
	// so a second one survives and leaves two pins uv cannot resolve.
	c := Constraints{RequiresPython: "==3.12.*", DatabricksConnect: "databricks-connect==17.0.0"}
	user := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = ["databricks-connect==17.0.0", "databricks-connect==16.0.0"]
`)
	// The first pin already matches the env, so there is nothing to override — but the
	// second still has to be reconciled by hand.
	assert.Equal(t, []string{WarnDBConnectPinDuplicated}, codes(detectWarnings(user, c)))
}

func TestConstraintConflictsDuplicateEnvEntriesAreOrderIndependent(t *testing.T) {
	// Constraint entries for one package compose as a conjunction, so they are
	// intersected rather than letting one entry win. 19.0.0 is outside [20, 21)
	// whichever order the artifact lists the bounds in.
	user := []string{"pyarrow==19.0.0"}
	assert.Equal(t, []string{WarnUserConstraintConflict},
		codes(constraintConflicts(user, []string{"pyarrow<21", "pyarrow>=20"})))
	assert.Equal(t, []string{WarnUserConstraintConflict},
		codes(constraintConflicts(user, []string{"pyarrow>=20", "pyarrow<21"})))

	// A pin inside the intersection is not a conflict, again either way round.
	inside := []string{"pyarrow==20.5.0"}
	assert.Empty(t, constraintConflicts(inside, []string{"pyarrow<21", "pyarrow>=20"}))
	assert.Empty(t, constraintConflicts(inside, []string{"pyarrow>=20", "pyarrow<21"}))
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
	got := detectWarnings(user, c)
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
		{">=2.0,<3.0", "==5.0", true, "5.0 is above the compound range's ceiling"},
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
