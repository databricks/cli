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
	// 21.0.3 is inside ~=21.0.0's [21.0, 22.0) range — compatible, no warning.
	c := Constraints{RequiresPython: "==3.12.*", ConstraintDeps: []string{"pyarrow~=21.0.0"}}
	assert.Empty(t, detectMergeWarnings(user, c))
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
	}
	for _, tc := range cases {
		assert.Equalf(t, tc.disjoint, rangesDisjoint(tc.user, tc.env),
			"rangesDisjoint(%q,%q): %s", tc.user, tc.env, tc.why)
	}
}
