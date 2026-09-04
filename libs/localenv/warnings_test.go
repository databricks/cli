package localenv

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// detectWarnings runs the detector the way mergePlan does, taking the databricks-connect
// plan from the merge itself rather than hand-supplied values. Tests must not assert
// against a different notion of what the merge rewrites or removes than production uses —
// that divergence is the bug this wiring exists to prevent.
func detectWarnings(userPyproject []byte, c Constraints) []Warning {
	return detectMergeWarnings(userPyproject, c, planDBConnect(userPyproject, c), false)
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

func TestDetectMergeWarningsSkipConstraintsSuppressesConstraintWarnings(t *testing.T) {
	// Under --no-constraints the merge leaves requires-python and the constraint
	// block unmanaged, so neither the requires-python-overridden warning nor a
	// constraint conflict is reported — but the orthogonal databricks-connect
	// override warning still is.
	user := []byte(`[project]
name = "demo"
requires-python = ">=3.10"

[dependency-groups]
dev = ["databricks-connect~=16.1.0", "pydantic~=1.0"]
`)
	c := Constraints{
		RequiresPython:    "==3.12.*",
		DatabricksConnect: "databricks-connect~=18.0.0",
		ConstraintDeps:    []string{"pydantic~=2.10.6"},
	}
	got := detectMergeWarnings(user, c, planDBConnect(user, c), true)
	assert.Equal(t, []string{WarnDBConnectPinOverridden}, codes(got))
}

func TestDetectMergeWarningsWithMultilineString(t *testing.T) {
	user := []byte(`[project]
requires-python = ">=3.10"
description = """
[dependency-groups]
dev = ["databricks-connect==1.0.0"]
"""

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

func TestDetectMergeWarningsStaleEnvironmentVersionOnClusterTarget(t *testing.T) {
	// A cluster target (c.EnvironmentVersion == "") that finds a leftover serverless
	// environment_version warns that the value is now stale.
	user := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = ["databricks-connect~=18.0.0"]

[tool.databricks.environment]
environment_version = "5"
`)
	c := Constraints{RequiresPython: "==3.12.*", DatabricksConnect: "databricks-connect~=18.0.0", EnvironmentVersion: ""}
	got := detectWarnings(user, c)
	assert.Equal(t, []string{WarnStaleEnvironmentVersion}, codes(got))
	assert.Contains(t, got[0].Message, `"5"`)
}

func TestDetectMergeWarningsNoStaleWarningForServerlessTarget(t *testing.T) {
	// A serverless target manages the section (refreshes it), so an existing value
	// is not stale and must not warn.
	user := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = ["databricks-connect~=18.0.0"]

[tool.databricks.environment]
environment_version = "4"
`)
	c := Constraints{RequiresPython: "==3.12.*", DatabricksConnect: "databricks-connect~=18.0.0", EnvironmentVersion: "5"}
	assert.NotContains(t, codes(detectWarnings(user, c)), WarnStaleEnvironmentVersion)
}

func TestDetectMergeWarningsNoStaleWarningWhenSectionAbsent(t *testing.T) {
	// A cluster target with no existing section has nothing to go stale.
	user := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = ["databricks-connect~=18.0.0"]
`)
	c := Constraints{RequiresPython: "==3.12.*", DatabricksConnect: "databricks-connect~=18.0.0", EnvironmentVersion: ""}
	assert.NotContains(t, codes(detectWarnings(user, c)), WarnStaleEnvironmentVersion)
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

func TestDetectMergeWarningsStandalonePysparkConflict(t *testing.T) {
	// databricks-connect vendors its own pyspark, so a standalone pyspark the user
	// declares collides with it in a shared environment. The warning fires whenever
	// the env manages databricks-connect, independent of the pyspark version pinned,
	// and sits after the databricks-connect warnings but before constraint conflicts.
	user := []byte(`[project]
requires-python = "==3.12.*"
dependencies = ["pyspark>=3.5.0", "pyarrow==21.0.0"]

[dependency-groups]
dev = ["databricks-connect~=16.1.0"]
`)
	c := Constraints{
		RequiresPython:    "==3.12.*",
		DatabricksConnect: "databricks-connect~=18.0.0",
		ConstraintDeps:    []string{"pyarrow<19"},
	}
	got := detectWarnings(user, c)
	assert.Equal(t, []string{
		WarnDBConnectPinOverridden,
		WarnStandalonePysparkConflict,
		WarnUserConstraintConflict,
	}, codes(got))
	assert.Contains(t, got[1].Message, "pyspark")
	assert.Contains(t, got[1].Message, "databricks-connect")
}

func TestStandalonePysparkIgnoredInConstraintsOnly(t *testing.T) {
	// In constraints-only mode the env does not install databricks-connect, so there
	// is no vendored pyspark to collide with — a standalone pyspark is fine.
	user := []byte(`[project]
requires-python = "==3.12.*"
dependencies = ["pyspark>=3.5.0"]
`)
	c := Constraints{RequiresPython: "==3.12.*", DatabricksConnect: ""}
	assert.Empty(t, detectWarnings(user, c))
}

func TestStandalonePysparkReportedOnce(t *testing.T) {
	// pyspark declared in both [project].dependencies and a dependency group is one
	// collision to fix, reported once so it does not inflate the code histogram.
	user := []byte(`[project]
requires-python = "==3.12.*"
dependencies = ["PySpark == 4.2.0"]

[dependency-groups]
dev = ["pyspark", "databricks-connect~=18.0.0"]
`)
	c := Constraints{RequiresPython: "==3.12.*", DatabricksConnect: "databricks-connect~=18.0.0"}
	assert.Equal(t, []string{WarnStandalonePysparkConflict}, codes(detectWarnings(user, c)))
}

func TestStandalonePysparkFiresInConstraintsOnlyWhenUserPinsDBConnect(t *testing.T) {
	// Constraints-only mode does not manage databricks-connect, but if the user's own
	// pyproject pins it the merge leaves that pin in place (mergeDatabricksConnect is a
	// no-op on an empty managed value), so uv still installs databricks-connect and a
	// standalone pyspark still collides. The warning must fire — matching the validate
	// hard-fail, which keys on the installed venv rather than the mode.
	user := []byte(`[project]
requires-python = "==3.12.*"
dependencies = ["pyspark>=3.5.0", "databricks-connect~=18.0.0"]
`)
	c := Constraints{RequiresPython: "==3.12.*", DatabricksConnect: ""}
	assert.Equal(t, []string{WarnStandalonePysparkConflict}, codes(detectWarnings(user, c)))
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

func TestDetectMergeWarningsStrayPinsAreConsolidated(t *testing.T) {
	// A databricks-connect pin in another group — reached through an include-group or
	// held directly in a sibling group — is removed by the consolidation pass so the
	// managed pin is the only one left. That is reported as consolidation, not as an
	// override (nothing in dev was rewritten) and not as a duplicate (the pin does not
	// survive the merge).
	c := Constraints{RequiresPython: "==3.12.*", DatabricksConnect: "databricks-connect==17.0.0"}

	indirect := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = [{include-group = "spark"}]
spark = ["databricks-connect==16.1.0"]
`)
	got := detectWarnings(indirect, c)
	assert.Equal(t, []string{WarnDBConnectConsolidated}, codes(got))
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
	// removes the included one, so both an override and a consolidation are reported.
	both := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = ["databricks-connect==16.1.0", {include-group = "spark"}]
spark = ["databricks-connect==15.0.0"]
`)
	assert.Equal(t, []string{WarnDBConnectPinOverridden, WarnDBConnectConsolidated},
		codes(detectWarnings(both, c)))

	// Consolidation actually fixes the project: re-running on the merged file finds a
	// single managed pin and nothing to warn about (the spark stray is gone).
	merged, _, err := MergeManaged(both, c, false)
	require.NoError(t, err)
	assert.Empty(t, detectWarnings(merged, c))

	// An include-group cycle must terminate rather than recurse forever, and the pin
	// behind it is still removed.
	cyclic := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = [{include-group = "a"}]
a = [{include-group = "dev"}, "databricks-connect==16.1.0"]
`)
	assert.Equal(t, []string{WarnDBConnectConsolidated}, codes(detectWarnings(cyclic, c)))

	// A quoted TOML group key is outside the line-based removal's reach, so the pin
	// survives beside the managed one and is reported as a duplicate — uv still cannot
	// resolve it, so the user must not lose the signal.
	quotedKey := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = [{include-group = "my-spark-group"}]
"my-spark-group" = ["databricks-connect==16.1.0"]
`)
	assert.Equal(t, []string{WarnDBConnectPinDuplicated}, codes(detectWarnings(quotedKey, c)))

	// A stray pin that already matches the env is not disjoint from it, so the gate
	// leaves it in place and there is nothing to warn about.
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

func TestDetectMergeWarningsNonLiteralDevGroupIsConsolidated(t *testing.T) {
	// MergeManaged finds the dev array with devKeyRe (`^\s*dev\s*=`), so a group spelled
	// "Dev" is never rewritten: the merge adds its own dev key. The consolidation pass
	// then removes the stray pin from "Dev", so it is reported as consolidated, not as
	// an override (nothing was replaced in place).
	c := Constraints{RequiresPython: "==3.12.*", DatabricksConnect: "databricks-connect==17.0.0"}
	user := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
Dev = ["databricks-connect==16.1.0"]
`)
	got := detectWarnings(user, c)
	assert.Equal(t, []string{WarnDBConnectConsolidated}, codes(got))
	assert.NotContains(t, got[0].Message, "is replaced")

	merged, _, err := MergeManaged(user, c, false)
	require.NoError(t, err)
	assert.NotContains(t, string(merged), `databricks-connect==16.1.0`, "the stray pin is removed, not retained")
}

func TestDBConnectWarningsFollowWhatTheMergeDoes(t *testing.T) {
	// The warnings must be driven by the merge's own answer, not by a second
	// implementation of what it edits. The merge rewrites and removes only
	// double-quoted elements the line-based passes reach, so a pin the passes cannot
	// touch survives beside the managed pin and is a duplicate (uv cannot resolve),
	// while a pin they can reach is consolidated away.
	c := Constraints{RequiresPython: "==3.12.*", DatabricksConnect: "databricks-connect==17.0.0"}

	// Pins the line-based passes cannot reach survive the merge — a duplicate.
	survives := map[string]string{
		// A TOML literal string is a string, but not one the double-quoted matcher sees.
		"single-quoted element": `[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = ['databricks-connect==16.1.0']
`,
		// A dotted key defines the group outside a [dependency-groups] table header, so
		// the line-based removal never scans it.
		"top-level dotted key": `dependency-groups.dev = ["databricks-connect==16.1.0"]

[project]
requires-python = "==3.12.*"
`,
	}
	for name, body := range survives {
		got := detectWarnings([]byte(body), c)
		assert.Equal(t, []string{WarnDBConnectPinDuplicated}, codes(got), name)
		assert.NotContains(t, got[0].Message, "is replaced", name)

		// The merged file really does carry both pins, which is what the code reports.
		merged, _, err := MergeManaged([]byte(body), c, false)
		require.NoError(t, err, name)
		assert.Contains(t, string(merged), "databricks-connect==16.1.0", name)
		assert.Contains(t, string(merged), "databricks-connect==17.0.0", name)
	}

	// A double-quoted pin in a normalization-equal group ("Dev") is reached by the
	// removal pass, so it is consolidated away rather than left as a duplicate.
	capitalized := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
Dev = ["databricks-connect==16.1.0"]
`)
	got := detectWarnings(capitalized, c)
	assert.Equal(t, []string{WarnDBConnectConsolidated}, codes(got))
	merged, _, err := MergeManaged(capitalized, c, false)
	require.NoError(t, err)
	assert.NotContains(t, string(merged), "databricks-connect==16.1.0")
	assert.Contains(t, string(merged), "databricks-connect==17.0.0")
}

func TestDBConnectDuplicateOnlyWhenRangesCannotBothHold(t *testing.T) {
	// A pin the merge cannot remove (single-quoted, so the double-quoted-only passes do
	// not reach it) survives beside the managed pin. Two pins for one package are only a
	// problem when nothing satisfies both: uv resolves an overlapping pair without
	// complaint, so there is nothing to reconcile and no warning to give.
	c := Constraints{RequiresPython: "==3.12.*", DatabricksConnect: "databricks-connect~=17.2.0"}

	overlapping := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = ['databricks-connect>=16']
`)
	assert.Empty(t, detectWarnings(overlapping, c), ">=16 and ~=17.2.0 both hold at 17.2.x")

	disjoint := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = ['databricks-connect==15.0.0']
`)
	assert.Equal(t, []string{WarnDBConnectPinDuplicated}, codes(detectWarnings(disjoint, c)),
		"==15.0.0 is outside ~=17.2.0, so uv cannot resolve")
}

func TestDBConnectWarningsSecondDirectPinIsConsolidated(t *testing.T) {
	// MergeManaged rewrites only the first databricks-connect element in the dev array;
	// the consolidation pass then removes any further one, so a second pin is reported
	// as consolidated rather than left as a duplicate uv cannot resolve.
	c := Constraints{RequiresPython: "==3.12.*", DatabricksConnect: "databricks-connect==17.0.0"}
	user := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = ["databricks-connect==17.0.0", "databricks-connect==16.0.0"]
`)
	// The first pin already matches the env, so there is nothing to override; the second
	// is removed and reported as consolidated.
	assert.Equal(t, []string{WarnDBConnectConsolidated}, codes(detectWarnings(user, c)))
}

func TestConsolidatedWarningNamesLocationAndCoexistsWithOverride(t *testing.T) {
	// A stray pin in [project].dependencies is consolidated, and the message names where
	// it came from so the user can see what the merge edited.
	c := Constraints{RequiresPython: "==3.12.*", DatabricksConnect: "databricks-connect~=17.2.0"}
	projectDeps := []byte(`[project]
requires-python = "==3.12.*"
dependencies = ["databricks-connect==15.1.*"]
`)
	got := detectWarnings(projectDeps, c)
	require.Equal(t, []string{WarnDBConnectConsolidated}, codes(got))
	assert.Contains(t, got[0].Message, "[project].dependencies")

	// An override in dev and consolidations across an optional extra and a sibling group
	// coexist, in a deterministic order (override first, then removals in resolution
	// order: optional-dependencies before dependency-groups).
	mixed := []byte(`[project]
requires-python = "==3.12.*"

[project.optional-dependencies]
extra = ["databricks-connect==14.0.0"]

[dependency-groups]
dev = ["databricks-connect==16.1.0"]
docs = ["databricks-connect==15.0.0"]
`)
	want := codes(detectWarnings(mixed, c))
	assert.Equal(t, []string{WarnDBConnectPinOverridden, WarnDBConnectConsolidated, WarnDBConnectConsolidated}, want)
	// The order must be stable across runs (group names come from a Go map).
	for range 200 {
		assert.Equal(t, want, codes(detectWarnings(mixed, c)))
	}
}

func TestConstraintsOnlyEmitsNoDBConnectWarnings(t *testing.T) {
	// With an empty DatabricksConnect (constraints-only mode) databricks-connect is left
	// untouched wherever it sits, so none of the db-connect warnings fire.
	c := Constraints{RequiresPython: "==3.12.*", DatabricksConnect: ""}
	user := []byte(`[project]
requires-python = "==3.12.*"
dependencies = ["databricks-connect==15.1.*"]

[dependency-groups]
dev = ["databricks-connect~=16.0"]
docs = ["databricks-connect==15.0.0"]
`)
	assert.Empty(t, detectWarnings(user, c))
}

func TestUserConstraintConflictFiresForWildcardPin(t *testing.T) {
	// A prefix-match wildcard pin ("==18.*") must be modeled so a conflict with the env's
	// constraint-dependencies is reported rather than silently missed.
	user := []byte(`[project]
requires-python = "==3.12.*"
dependencies = ["pyarrow==18.*"]
`)
	c := Constraints{RequiresPython: "==3.12.*", ConstraintDeps: []string{"pyarrow~=19.0"}}
	got := detectWarnings(user, c)
	assert.Equal(t, []string{WarnUserConstraintConflict}, codes(got))
	assert.Contains(t, got[0].Message, "pyarrow")
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
		{"", "~=21.0.0", false, "no user spec — nothing to compare"},

		// PEP 440 prefix-match wildcards ("==X.Y.*"). The clause admits the whole X.Y
		// series: [X.Y, X.(Y+1)). Only "==" carries a wildcard; "!=X.*" and any other
		// operator stay undecidable (parseClause refuses them), never a false conflict.
		{"==15.1.*", "~=17.0", true, "the 15.1 series is entirely below [17.0, 18.0)"},
		{"==17.0.*", "~=17.0", false, "the 17.0 series sits inside [17.0, 18.0)"},
		{"==17.*", "~=17.2.0", false, "the 17.* series spans the ~=17.2.0 range"},
		{"==2.*", "==3.0", true, "the 2.* series ends at 3, which the exact 3.0 exceeds"},
		{"==2.*", "==2.0", false, "2.0 is inside the 2.* series [2, 3)"},
		{"!=2.*", "==2.0", false, "!=X.* excludes a whole series — undecidable, never a conflict"},

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
