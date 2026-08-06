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
dependencies = ["pyarrow==17.0.0", "requests>=2.0"]
`)
	// Env constrains pyarrow to ~=21.0.0 (admits [21.0.0, 21.1.0)); the user's
	// ==17.0.0 is provably outside it. requests is not constrained → no conflict.
	c := Constraints{
		RequiresPython: "==3.12.*",
		ConstraintDeps: []string{"pyarrow~=21.0.0", "numpy~=2.1.3"},
	}
	got := detectMergeWarnings(user, c)
	assert.Equal(t, []string{WarnUserConstraintConflict}, codes(got))
	assert.Contains(t, got[0].Message, "pyarrow")
}

func TestDetectMergeWarningsNoConflictWhenCompatible(t *testing.T) {
	user := []byte(`[project]
requires-python = "==3.12.*"
dependencies = ["pyarrow==21.0.3"]
`)
	// 21.0.3 is inside ~=21.0.0's [21.0.0, 21.1.0) range — compatible, no warning.
	c := Constraints{RequiresPython: "==3.12.*", ConstraintDeps: []string{"pyarrow~=21.0.0"}}
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

func TestDetectMergeWarningsIncludeGroupIndirection(t *testing.T) {
	// The user's pin is reachable only through an include-group, but MergeManaged
	// writes the env's pin into dev anyway — so without following the indirection the
	// merged file would carry two databricks-connect pins and no override warning.
	c := Constraints{RequiresPython: "==3.12.*", DatabricksConnect: "databricks-connect==17.0.0"}

	indirect := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = [{include-group = "spark"}]
spark = ["databricks-connect==16.1.0"]
`)
	assert.Equal(t, []string{WarnDBConnectPinOverridden}, codes(detectMergeWarnings(indirect, c)))

	// An include-group cycle must terminate rather than recurse forever, and the pin
	// behind it is still found.
	cyclic := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = [{include-group = "a"}]
a = [{include-group = "dev"}, "databricks-connect==16.1.0"]
`)
	assert.Equal(t, []string{WarnDBConnectPinOverridden}, codes(detectMergeWarnings(cyclic, c)))

	// PEP 735 normalizes group names the same way PEP 503 normalizes package names.
	renamed := []byte(`[project]
requires-python = "==3.12.*"

[dependency-groups]
dev = [{include-group = "My_Spark.Group"}]
"my-spark-group" = ["databricks-connect==16.1.0"]
`)
	assert.Equal(t, []string{WarnDBConnectPinOverridden}, codes(detectMergeWarnings(renamed, c)))

	// An included pin that already matches the env is not an override.
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
		{">=2.0", "==2.5", false, "cannot decide a lower-bound vs exact conservatively"},
		{"!=2.0", "==2.0", false, "!= is not modeled — never a conflict"},
		{">=2.0,<3.0", "==5.0", false, "multi-clause range is treated as unknown"},
		{"==2.*", "==2.0", false, "wildcards are unparsed — no conflict"},
		{"", "~=21.0.0", false, "no user spec — nothing to compare"},
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
