package localenv

import (
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// requireValidTOML fails the test if b is not parseable TOML. String-only
// assertions miss structural corruption such as a table header emitted twice
// ("Key 'tool.uv' has already been defined"), which uv also rejects on sync.
func requireValidTOML(t *testing.T, b []byte) {
	t.Helper()
	var v map[string]any
	_, err := toml.Decode(string(b), &v)
	require.NoError(t, err, "merged output must be valid TOML:\n%s", b)
}

func testConstraints() Constraints {
	return Constraints{
		RequiresPython:    "==3.12.*",
		DatabricksConnect: "databricks-connect~=17.2.0",
		ConstraintDeps:    []string{"pydantic~=2.10.6", "anyio~=4.6.2"},
	}
}

func TestMergeReplacesRequiresPythonPreservingComments(t *testing.T) {
	in := []byte(`[project]
name = "demo"
# keep this comment
requires-python = ">=3.10"

[dependency-groups]
dev = [
    "databricks-connect~=16.0.0",
    "pytest~=8.0",
]
`)
	out, regions, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	assert.Contains(t, string(out), `requires-python = "==3.12.*"`)
	assert.Contains(t, string(out), "# keep this comment")
	assert.Contains(t, string(out), `"databricks-connect~=17.2.0",`)
	assert.Contains(t, string(out), `"pytest~=8.0",`)
	assert.Contains(t, regions, "requires-python")
	assert.Contains(t, regions, "databricks-connect")
	assert.Contains(t, regions, "tool.uv.constraint-dependencies")
	assert.Contains(t, string(out), "pydantic~=2.10.6")
}

func TestMergeIsIdempotent(t *testing.T) {
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
dev = [
    "databricks-connect~=16.0.0",
]
`)
	once, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	twice, _, err := MergeManaged(once, testConstraints())
	require.NoError(t, err)
	assert.Equal(t, string(once), string(twice))
}

func TestMergeInsertsRequiresPythonWhenMissing(t *testing.T) {
	in := []byte(`[project]
name = "demo"

[dependency-groups]
dev = ["databricks-connect~=16.0.0"]
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	assert.Contains(t, string(out), `requires-python = "==3.12.*"`)
}

func TestMergeReplacesExistingManagedToolUvBlock(t *testing.T) {
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
dev = ["databricks-connect~=16.0.0"]

` + managedMarkerStart + `
[tool.uv]
constraint-dependencies = [
    "stale~=1.0.0",
]
` + managedMarkerEnd + `
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	assert.NotContains(t, string(out), "stale~=1.0.0")
	assert.Contains(t, string(out), "pydantic~=2.10.6")
	// Only one managed block remains.
	assert.Equal(t, 1, countOccurrences(string(out), managedMarkerStart))
}

func TestMergePreservesCRLF(t *testing.T) {
	in := []byte("[project]\r\nrequires-python = \">=3.10\"\r\n\r\n[dependency-groups]\r\ndev = [\"databricks-connect~=16.0.0\"]\r\n")
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	assert.Contains(t, string(out), "\r\n")
	assert.Contains(t, string(out), `requires-python = "==3.12.*"`)
	// Merging the CRLF output again must be byte-identical (idempotent under \r\n).
	twice, _, err := MergeManaged(out, testConstraints())
	require.NoError(t, err)
	assert.Equal(t, string(out), string(twice))
}

func TestMergePreservesUserToolUvKeys(t *testing.T) {
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
dev = ["databricks-connect~=16.0.0"]

[tool.uv]
package = true
dev-dependencies = ["ruff"]
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "[tool.uv]")
	assert.Contains(t, s, "package = true")
	assert.Contains(t, s, `dev-dependencies = ["ruff"]`)
	assert.Contains(t, s, managedMarkerStart)
	assert.Contains(t, s, "pydantic~=2.10.6")
	// The user's keys must live outside the managed marker block.
	start := strings.Index(s, managedMarkerStart)
	require.GreaterOrEqual(t, start, 0)
	assert.NotContains(t, s[start:], "package = true")
	assert.NotContains(t, s[start:], `dev-dependencies = ["ruff"]`)
	// The result must be valid TOML: the managed constraint-dependencies nests
	// inside the user's [tool.uv] rather than emitting a second [tool.uv] header.
	requireValidTOML(t, out)
	assert.Equal(t, 1, countOccurrences(s, "[tool.uv]"))
	// Merge-twice is byte-identical (header-less managed region stays header-less).
	twice, _, err := MergeManaged(out, testConstraints())
	require.NoError(t, err)
	assert.Equal(t, s, string(twice))
	requireValidTOML(t, twice)
}

func TestMergeStripsStaleConstraintDepsFromUserToolUv(t *testing.T) {
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
dev = ["databricks-connect~=16.0.0"]

[tool.uv]
package = true
constraint-dependencies = ["old~=1.0"]
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "package = true")
	// The stale constraint must be gone from the user table; the managed block has the new deps.
	assert.NotContains(t, s, "old~=1.0")
	assert.Contains(t, s, "pydantic~=2.10.6")
	// Valid TOML with a single [tool.uv]: the managed deps nest in the user table.
	requireValidTOML(t, out)
	assert.Equal(t, 1, countOccurrences(s, "[tool.uv]"))
	// Merge-twice is byte-identical.
	twice, _, err := MergeManaged(out, testConstraints())
	require.NoError(t, err)
	assert.Equal(t, string(out), string(twice))
}

func TestMergeRemovesOwnedOnlyToolUv(t *testing.T) {
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
dev = ["databricks-connect~=16.0.0"]

[tool.uv]
constraint-dependencies = ["old~=1.0"]
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	s := string(out)
	assert.NotContains(t, s, "old~=1.0")
	assert.Contains(t, s, "pydantic~=2.10.6")
	// The plain table was removed and replaced by exactly one managed block.
	assert.Equal(t, 1, countOccurrences(s, "[tool.uv]"))
	assert.Equal(t, 1, countOccurrences(s, managedMarkerStart))
	requireValidTOML(t, out)
}

func TestMergeRemovesOwnedOnlyMultiLineToolUv(t *testing.T) {
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
dev = ["databricks-connect~=16.0.0"]

[tool.uv]
constraint-dependencies = [
    "old~=1.0",
]
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	s := string(out)
	assert.NotContains(t, s, "old~=1.0")
	assert.Contains(t, s, "pydantic~=2.10.6")
	// The multi-line owned-only table was removed whole, leaving exactly one
	// [tool.uv] (inside the managed block) and no stray empty header.
	assert.Equal(t, 1, countOccurrences(s, "[tool.uv]"))
	assert.Equal(t, 1, countOccurrences(s, managedMarkerStart))
	requireValidTOML(t, out)
	// Merge-twice is byte-identical.
	twice, _, err := MergeManaged(out, testConstraints())
	require.NoError(t, err)
	assert.Equal(t, string(out), string(twice))
}

func TestMergeReplacesSingleLineDevArray(t *testing.T) {
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
dev = ["databricks-connect~=16.0.0", "pytest~=8.0"]
`)
	out, regions, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	// Sibling element and single-line array layout are preserved.
	assert.Contains(t, string(out), `dev = ["databricks-connect~=17.2.0", "pytest~=8.0"]`)
	assert.Contains(t, regions, "databricks-connect")
}

func TestMergePreservesMultiLineTrailingComma(t *testing.T) {
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
dev = [
    "databricks-connect~=16.0.0",
]
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	// The trailing comma on the managed element is preserved.
	assert.Contains(t, string(out), `    "databricks-connect~=17.2.0",`)
}

func TestMergeInsertsDatabricksConnectMultiLine(t *testing.T) {
	// Existing project with a dev group that does not pin databricks-connect: the
	// pin must be inserted (mirroring greenfield) rather than left absent.
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
dev = [
    "pytest~=8.0",
]
`)
	out, regions, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, `"databricks-connect~=17.2.0",`)
	assert.Contains(t, s, `"pytest~=8.0",`, "existing element preserved")
	assert.Contains(t, regions, "databricks-connect")
	// Idempotent: a second merge finds the element and rewrites in place.
	out2, _, err := MergeManaged(out, testConstraints())
	require.NoError(t, err)
	assert.Equal(t, s, string(out2))
}

func TestMergeInsertsDatabricksConnectSingleLine(t *testing.T) {
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
dev = ["pytest~=8.0"]
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	assert.Contains(t, string(out), `dev = ["databricks-connect~=17.2.0", "pytest~=8.0"]`)
	out2, _, err := MergeManaged(out, testConstraints())
	require.NoError(t, err)
	assert.Equal(t, string(out), string(out2))
}

func TestMergeInsertsDatabricksConnectEmptyDev(t *testing.T) {
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
dev = []
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	assert.Contains(t, string(out), `dev = ["databricks-connect~=17.2.0"]`)
}

func TestMergeInsertsDevKeyWhenAbsent(t *testing.T) {
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
test = ["pytest~=8.0"]
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, `"databricks-connect~=17.2.0",`)
	assert.Contains(t, s, `test = ["pytest~=8.0"]`, "sibling group untouched")
	out2, _, err := MergeManaged(out, testConstraints())
	require.NoError(t, err)
	assert.Equal(t, s, string(out2))
}

func TestMergeInsertsDependencyGroupsWhenAbsent(t *testing.T) {
	in := []byte(`[project]
requires-python = ">=3.10"
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "[dependency-groups]")
	assert.Contains(t, s, `"databricks-connect~=17.2.0",`)
	out2, _, err := MergeManaged(out, testConstraints())
	require.NoError(t, err)
	assert.Equal(t, s, string(out2))
}

func TestMergeReplacesDatabricksConnectOnDevLine(t *testing.T) {
	// The pin shares the opening "dev = [" line of a multi-line array. It must be
	// rewritten in place, not duplicated by the insert path.
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
dev = ["databricks-connect~=16.0.0",
    "pytest~=8.0",
]
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	s := string(out)
	assert.Equal(t, 1, strings.Count(s, "databricks-connect"), "must not duplicate the pin")
	assert.Contains(t, s, "databricks-connect~=17.2.0")
	assert.NotContains(t, s, "databricks-connect~=16.0.0")
}

func TestMergeReplacesDatabricksConnectWithTrailingComment(t *testing.T) {
	// The pin element carries a trailing comment. It must be rewritten in place
	// (comment preserved), not duplicated.
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
dev = [
    "databricks-connect~=16.0.0", # keep me
    "pytest~=8.0",
]
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	s := string(out)
	assert.Equal(t, 1, strings.Count(s, "databricks-connect"), "must not duplicate the pin")
	assert.Contains(t, s, `"databricks-connect~=17.2.0", # keep me`)
}

func TestMergePreservesCommentMentioningDatabricksConnect(t *testing.T) {
	// A trailing comment that itself contains a quoted "databricks-connect..."
	// token must be preserved byte-for-byte; only the code element is rewritten.
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
dev = [
    "databricks-connect~=16.0.0", # was "databricks-connect~=16.0.0"
    "pytest~=8.0",
]
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	s := string(out)
	// Code element is rewritten; the comment mention is left verbatim.
	assert.Contains(t, s, `"databricks-connect~=17.2.0", # was "databricks-connect~=16.0.0"`)
	assert.Equal(t, 1, strings.Count(s, "databricks-connect~=16.0.0"), "only the comment mention of the old pin remains")
}

func TestMergeRewritesNonCanonicalDatabricksConnectSpelling(t *testing.T) {
	// A pin spelled with PEP 503-equivalent separators/case must be rewritten in
	// place, not left undetected so the insert path adds a conflicting second pin.
	for _, spelling := range []string{"databricks_connect", "Databricks-Connect", "databricks.connect"} {
		in := []byte("[project]\nrequires-python = \">=3.10\"\n\n[dependency-groups]\ndev = [\n    \"" + spelling + "~=16.0.0\",\n]\n")
		out, _, err := MergeManaged(in, testConstraints())
		require.NoError(t, err, spelling)
		s := string(out)
		assert.Equal(t, 1, strings.Count(s, `"databricks-connect~=17.2.0"`), "spelling %q must be rewritten in place, not duplicated:\n%s", spelling, s)
		assert.NotContains(t, s, "~=16.0.0", "old pin (any spelling) must be gone")
	}
}

func TestMergeInsertsCommaWhenLastElementHasNone(t *testing.T) {
	// TOML permits the final array element to omit its trailing comma. Inserting
	// after it must first add the separating comma, or the result is invalid TOML.
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
dev = [
    "pytest~=8.0"
]
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, `"pytest~=8.0",`, "previous last element gains a separating comma")
	assert.Contains(t, s, `"databricks-connect~=17.2.0",`)
	// Round-trips as valid TOML.
	out2, _, err := MergeManaged(out, testConstraints())
	require.NoError(t, err)
	assert.Equal(t, s, string(out2))
}

func TestMergeConstraintsOnlyDoesNotInsertDatabricksConnect(t *testing.T) {
	// Empty DatabricksConnect (constraints-only) must not add a pin.
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
dev = ["pytest~=8.0"]
`)
	c := testConstraints()
	c.DatabricksConnect = ""
	out, regions, err := MergeManaged(in, c)
	require.NoError(t, err)
	assert.NotContains(t, string(out), "databricks-connect")
	assert.NotContains(t, regions, "databricks-connect")
}

func TestRenderFreshPyproject(t *testing.T) {
	out := RenderFreshPyproject("demo", testConstraints())
	s := string(out)
	assert.Contains(t, s, `name = "demo"`)
	assert.Contains(t, s, `requires-python = "==3.12.*"`)
	assert.Contains(t, s, `"databricks-connect~=17.2.0",`)
	assert.Contains(t, s, managedMarkerStart)
	assert.Contains(t, s, managedMarkerEnd)
	assert.Contains(t, s, "pydantic~=2.10.6")
	// A cluster target (no EnvironmentVersion) writes no [tool.databricks.environment].
	assert.NotContains(t, s, "[tool.databricks.environment]")
	// A fresh render is itself a no-op under MergeManaged (already fully managed).
	merged, _, err := MergeManaged(out, testConstraints())
	require.NoError(t, err)
	assert.Equal(t, s, string(merged))
}

func TestRenderFreshPyprojectServerlessWritesEnvironment(t *testing.T) {
	c := testConstraints()
	c.EnvironmentVersion = "5"
	out := RenderFreshPyproject("demo", c)
	s := string(out)
	assert.Contains(t, s, "[tool.databricks.environment]")
	assert.Contains(t, s, `environment_version = "5"`)
	requireValidTOML(t, out)
	// A fresh render is itself a no-op under MergeManaged (already fully managed).
	merged, _, err := MergeManaged(out, c)
	require.NoError(t, err)
	assert.Equal(t, s, string(merged))
}

func TestMergeInsertsDatabricksEnvironmentWhenAbsent(t *testing.T) {
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
dev = ["databricks-connect~=16.0.0"]
`)
	c := testConstraints()
	c.EnvironmentVersion = "5"
	out, regions, err := MergeManaged(in, c)
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "[tool.databricks.environment]")
	assert.Contains(t, s, `environment_version = "5"`)
	assert.Contains(t, regions, "tool.databricks.environment")
	requireValidTOML(t, out)
	// Idempotent.
	twice, _, err := MergeManaged(out, c)
	require.NoError(t, err)
	assert.Equal(t, s, string(twice))
}

func TestMergeReplacesExistingEnvironmentVersionPreservingComment(t *testing.T) {
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
dev = ["databricks-connect~=16.0.0"]

[tool.databricks.environment]
environment_version = "4"  # pinned
`)
	c := testConstraints()
	c.EnvironmentVersion = "5"
	out, regions, err := MergeManaged(in, c)
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, `environment_version = "5"  # pinned`)
	assert.NotContains(t, s, `environment_version = "4"`)
	assert.Contains(t, regions, "tool.databricks.environment")
	// The table is refreshed in place, not duplicated.
	assert.Equal(t, 1, countOccurrences(s, "[tool.databricks.environment]"))
	requireValidTOML(t, out)
}

func TestMergeInsertsEnvironmentVersionKeyWhenTableExists(t *testing.T) {
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
dev = ["databricks-connect~=16.0.0"]

[tool.databricks.environment]
# user note
`)
	c := testConstraints()
	c.EnvironmentVersion = "5"
	out, _, err := MergeManaged(in, c)
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, `environment_version = "5"`)
	assert.Contains(t, s, "# user note")
	assert.Equal(t, 1, countOccurrences(s, "[tool.databricks.environment]"))
	requireValidTOML(t, out)
}

func TestMergeEnvironmentNoopForClusterTarget(t *testing.T) {
	// A cluster target leaves EnvironmentVersion empty: the section is never
	// written, and an existing one is left untouched rather than removed.
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
dev = ["databricks-connect~=16.0.0"]

[tool.databricks.environment]
environment_version = "4"
`)
	c := testConstraints() // EnvironmentVersion == "": cluster target.
	out, regions, err := MergeManaged(in, c)
	require.NoError(t, err)
	assert.Contains(t, string(out), `environment_version = "4"`)
	assert.NotContains(t, regions, "tool.databricks.environment")
}

func TestMergeAddsEnvironmentToPreFeatureManagedFile(t *testing.T) {
	// The common upgrade path: a file a pre-feature CLI wrote for a serverless
	// target already carries the managed [tool.uv] marker block but no
	// [tool.databricks.environment] section. Re-running the new CLI must add the
	// section, keep exactly one managed block, and stay idempotent.
	in := []byte(`[project]
name = "demo"
version = "0.0.0"
requires-python = "==3.12.*"

[dependency-groups]
dev = [
    "databricks-connect~=17.2.0",
]

` + managedMarkerStart + `
[tool.uv]
constraint-dependencies = [
    "pydantic~=2.10.6",
    "anyio~=4.6.2",
]
` + managedMarkerEnd + `
`)
	c := testConstraints()
	c.EnvironmentVersion = "5"
	out, regions, err := MergeManaged(in, c)
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "[tool.databricks.environment]")
	assert.Contains(t, s, `environment_version = "5"`)
	assert.Contains(t, regions, "tool.databricks.environment")
	// The pre-existing managed [tool.uv] block is neither duplicated nor disturbed.
	assert.Equal(t, 1, countOccurrences(s, managedMarkerStart))
	assert.Equal(t, 1, countOccurrences(s, "[tool.databricks.environment]"))
	requireValidTOML(t, out)
	// Idempotent on the upgraded file.
	twice, _, err := MergeManaged(out, c)
	require.NoError(t, err)
	assert.Equal(t, s, string(twice))
}

func TestMergeStripsMultiLineConstraintDepsWithBracketInFirstElement(t *testing.T) {
	// The user's stale constraint-dependencies is a multi-line array whose FIRST
	// element line contains a "]" inside an extras spec. A naive
	// strings.Contains(line, "]") check would misread this as single-line and
	// strip only the first element, orphaning the rest and producing invalid TOML.
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
dev = ["databricks-connect~=16.0.0"]

[tool.uv]
package = true
constraint-dependencies = ["requests[security]~=2.0",
    "old-dep~=1.0",
]
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	s := string(out)
	// The whole stale array is gone (both the bracket-bearing first element and
	// the continuation), replaced by the managed deps.
	assert.NotContains(t, s, "requests[security]")
	assert.NotContains(t, s, "old-dep~=1.0")
	assert.Contains(t, s, "package = true")
	assert.Contains(t, s, "pydantic~=2.10.6")
	requireValidTOML(t, out)
	assert.Equal(t, 1, countOccurrences(s, "[tool.uv]"))
	// Idempotent.
	twice, _, err := MergeManaged(out, testConstraints())
	require.NoError(t, err)
	assert.Equal(t, s, string(twice))
}

func TestBracketDepthDeltaIgnoresStringsAndComments(t *testing.T) {
	cases := map[string]int{
		`constraint-dependencies = [`:                    1,  // opens, no close
		`constraint-dependencies = ["a", "b"]`:           0,  // balanced single-line
		`constraint-dependencies = ["requests[sec]~=2",`: 1,  // ] inside string ignored
		`    "old~=1.0",`:                                0,  // element line
		`]`:                                              -1, // close
		`]  # trailing note ]`:                           -1, // ] in comment ignored
		`constraint-dependencies = ["x"]  # [note]`:      0,  // comment ] ignored
	}
	for line, want := range cases {
		assert.Equal(t, want, bracketDepthDelta(line), "line: %q", line)
	}
}

func TestMergeConsolidatesDatabricksConnectFromSiblingGroup(t *testing.T) {
	// databricks-connect is fully owned by setup-local in the install flow: a stray pin
	// in a sibling group (docs) makes uv unsatisfiable, so it is removed and the managed
	// pin lives only in dev. This deliberately overrides the "sibling groups untouched"
	// rule that scopes mergeDatabricksConnect, but only for databricks-connect.
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
docs = ["databricks-connect~=14.3"]
dev = [
    "databricks-connect~=16.0.0",
]
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	s := string(out)
	// docs consolidated (its db-connect removed); dev updated to the managed pin.
	assert.Contains(t, s, `docs = []`)
	assert.NotContains(t, s, `databricks-connect~=14.3`)
	assert.Contains(t, s, `"databricks-connect~=17.2.0",`)
	requireValidTOML(t, out)
}

func TestMergeConsolidatesDatabricksConnectFromProjectDeps(t *testing.T) {
	// The reported bug: a template ships databricks-connect in [project].dependencies
	// while the dev group carries a different pin, so uv cannot co-resolve them. The
	// merge removes the [project].dependencies pin and manages db-connect only in dev.
	in := []byte(`[project]
requires-python = ">=3.10"
dependencies = [
    "databricks-dlt",
    "pytest",
    "databricks-connect==15.1.*",
]

[dependency-groups]
dev = [
    "databricks-connect~=16.0",
]
`)
	out, regions, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	s := string(out)
	assert.NotContains(t, s, "databricks-connect==15.1.*", "the stray project.dependencies pin is removed")
	assert.Contains(t, s, `"databricks-dlt",`, "sibling dependencies are preserved")
	assert.Contains(t, s, `"pytest",`)
	assert.Contains(t, s, `"databricks-connect~=17.2.0",`, "dev carries the single managed pin")
	assert.Contains(t, regions, regionDatabricksConnect)
	requireValidTOML(t, out)

	// Idempotent: a second merge finds nothing to remove and produces identical bytes.
	out2, _, err := MergeManaged(out, testConstraints())
	require.NoError(t, err)
	assert.Equal(t, string(out), string(out2))
}

func TestMergeConsolidatesDatabricksConnectEmptiesSingleLineArray(t *testing.T) {
	// Removing the only element of a single-line array leaves a valid empty array,
	// not a dangling comma.
	in := []byte(`[project]
requires-python = ">=3.10"

[project.optional-dependencies]
spark = ["databricks-connect==15.0.0"]

[dependency-groups]
dev = ["databricks-connect~=16.0"]
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "spark = []", "the optional-dependency extra is emptied")
	assert.NotContains(t, s, "databricks-connect==15.0.0")
	requireValidTOML(t, out)
}

func TestMergeConsolidatesLeavesOtherElementsInArray(t *testing.T) {
	// Only the databricks-connect element is removed from a mixed single-line array;
	// the surviving elements and the array structure stay intact.
	in := []byte(`[project]
requires-python = ">=3.10"
dependencies = ["numpy", "databricks-connect==15.1.*", "pytest"]

[dependency-groups]
dev = ["databricks-connect~=16.0"]
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, `dependencies = ["numpy", "pytest"]`)
	assert.NotContains(t, s, "databricks-connect==15.1.*")
	requireValidTOML(t, out)
}

func TestMergeKeepsCompatibleDatabricksConnectPins(t *testing.T) {
	// Only a databricks-connect pin that provably cannot co-resolve with the managed
	// pin is removed. A pin that overlaps it, carries no version, or is marker-gated
	// resolves fine, so it is left in place — the merge does not silently rewrite a
	// user declaration (including [project].dependencies wheel metadata) that isn't
	// broken. env pin here is ~=17.2.0.
	in := []byte(`[project]
requires-python = ">=3.10"
dependencies = [
    "databricks-connect>=15",
    "databricks-connect ; python_version < '3.13'",
]

[project.optional-dependencies]
extra = ["databricks-connect"]

[dependency-groups]
dev = ["databricks-connect~=16.0"]
test = ["databricks-connect>=15,<20"]
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, `"databricks-connect>=15",`, "an overlapping pin co-resolves and is kept")
	assert.Contains(t, s, "python_version < '3.13'", "a marker-gated pin is not compared and is kept")
	assert.Contains(t, s, `extra = ["databricks-connect"]`, "an unversioned pin is kept")
	assert.Contains(t, s, `test = ["databricks-connect>=15,<20"]`, "an overlapping ranged pin in another group is kept")
	assert.Contains(t, s, `"databricks-connect~=17.2.0"`, "the dev pin is still updated to the managed version")
	requireValidTOML(t, out)
}

func TestMergeKeepsEnvEqualPinInProjectDeps(t *testing.T) {
	// A [project].dependencies pin that already equals the managed version is not
	// disjoint from it, so it is left in place — the only declaration in the wheel
	// metadata is never silently deleted, and no consolidation warning is emitted.
	in := []byte(`[project]
requires-python = ">=3.10"
dependencies = ["databricks-connect~=17.2.0"]
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	assert.Contains(t, string(out), `dependencies = ["databricks-connect~=17.2.0"]`, "the env-equal pin is kept")
	requireValidTOML(t, out)
}

func TestMergeKeepsClosingBracketOnItsOwnLine(t *testing.T) {
	// Removing the last element of a multi-line array with no trailing comma on it must
	// not pull the closing "]" up onto the previous element's line (a common uv-init shape).
	in := []byte(`[project]
requires-python = ">=3.10"
dependencies = [
    "databricks-dlt",
    "pytest",
    "databricks-connect==15.1.*"
]

[dependency-groups]
dev = ["databricks-connect~=16.0"]
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	s := string(out)
	assert.NotContains(t, s, "databricks-connect==15.1.*")
	assert.Contains(t, s, "\n    \"pytest\",\n]", "the closing bracket keeps its own line and the trailing comma survives")
	assert.NotContains(t, s, `"pytest"]`, "the bracket is not pulled up")
	requireValidTOML(t, out)
}

func TestMergeConstraintsOnlyLeavesDatabricksConnectUntouched(t *testing.T) {
	// In constraints-only mode (empty DatabricksConnect) databricks-connect is not
	// managed at all: no pin is inserted and no stray is removed, anywhere.
	in := []byte(`[project]
requires-python = ">=3.10"
dependencies = ["databricks-connect==15.1.*"]

[dependency-groups]
dev = ["databricks-connect~=16.0"]
docs = ["databricks-connect~=14.3"]
`)
	c := Constraints{RequiresPython: "==3.12.*", ConstraintDeps: []string{"pydantic~=2.10.6"}}
	out, regions, err := MergeManaged(in, c)
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, "databricks-connect==15.1.*", "project.dependencies pin left untouched")
	assert.Contains(t, s, "databricks-connect~=16.0", "dev pin left untouched")
	assert.Contains(t, s, "databricks-connect~=14.3", "sibling group left untouched")
	assert.NotContains(t, regions, regionDatabricksConnect)
	requireValidTOML(t, out)
}

func TestMergeConsolidatesStrayAfterCommentedElement(t *testing.T) {
	// A stray databricks-connect element whose token carries a leading comment (from
	// the previous element's trailing comment, or the opening "[" line) must still be
	// removed. dbconnectElementPin scans the token per line so the leading comment does
	// not hide the quoted requirement on the next line.
	cases := map[string]string{
		"trailing comment on previous element": `[project]
requires-python = ">=3.10"
dependencies = [
    "pytest",  # test runner
    "databricks-connect==15.1.*",
]

[dependency-groups]
dev = ["databricks-connect~=16.0"]
`,
		"comment on the opening bracket line": `[project]
requires-python = ">=3.10"
dependencies = [  # runtime deps
    "databricks-connect==15.1.*",
    "pytest",
]

[dependency-groups]
dev = ["databricks-connect~=16.0"]
`,
	}
	// A comment on a surviving element (or the opening line) must be preserved; the
	// removal only drops the databricks-connect element, not the neighbouring comment.
	wantComment := map[string]string{
		"trailing comment on previous element": "# test runner",
		"comment on the opening bracket line":  "# runtime deps",
	}
	for name, in := range cases {
		out, _, err := MergeManaged([]byte(in), testConstraints())
		require.NoError(t, err, name)
		s := string(out)
		assert.NotContains(t, s, "databricks-connect==15.1.*", name)
		assert.Contains(t, s, `"pytest",`, name)
		assert.Contains(t, s, `"databricks-connect~=17.2.0"`, name)
		assert.Contains(t, s, wantComment[name], name+": neighbouring comment must be preserved")
		requireValidTOML(t, out)
	}
}

func TestMergeConsolidatesSecondDevPinAfterCommentedManagedPin(t *testing.T) {
	// The dev-group keepFirst dedup must also see through a comment: the managed pin
	// carries a trailing comment, and the second databricks-connect on the next line
	// must still be removed rather than shadowed by that comment.
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
dev = [
    "databricks-connect~=16.0",  # managed by setup-local
    "databricks-connect==15.0.0",
]
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, `"databricks-connect~=17.2.0",`, "the managed pin is kept and updated")
	assert.NotContains(t, s, "databricks-connect==15.0.0", "the second pin is removed")
	assert.Contains(t, s, "# managed by setup-local", "the managed pin's comment is preserved")
	requireValidTOML(t, out)
}

func TestMergeConsolidationSkipsIncludeGroupAndLiteralStrings(t *testing.T) {
	// The removal matches double-quoted databricks-connect elements only, mirroring the
	// rewrite: a PEP 735 include-group reference and a single-quoted pin are left alone.
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
dev = [{include-group = "spark"}]
spark = ['databricks-connect==15.0.0']
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, `{include-group = "spark"}`, "the include-group reference is preserved")
	assert.Contains(t, s, `'databricks-connect==15.0.0'`, "the single-quoted pin is not removed")
	requireValidTOML(t, out)
}

func TestMergeDatabricksConnectDoesNotClobberComment(t *testing.T) {
	// A databricks-connect token inside a trailing comment is user content and
	// must not be rewritten.
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
dev = ["pytest"] # keep "databricks-connect~=14.3" for docs
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	s := string(out)
	// The comment's databricks-connect~=14.3 is preserved verbatim.
	assert.Contains(t, s, `# keep "databricks-connect~=14.3" for docs`)
	requireValidTOML(t, out)
}

func TestMergeRequiresPythonPreservesInlineComment(t *testing.T) {
	in := []byte(`[project]
requires-python = ">=3.10" # maintained by platform team

[dependency-groups]
dev = ["databricks-connect~=16.0.0"]
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	s := string(out)
	assert.Contains(t, s, `requires-python = "==3.12.*" # maintained by platform team`)
	requireValidTOML(t, out)
}

func TestMergeHandlesTableHeaderInlineComments(t *testing.T) {
	// Table headers may carry a trailing comment. requires-python under a
	// commented [project] must still be updated, and the [dependency-groups] end
	// bound must not run past a commented sibling header into another table's dev.
	in := []byte(`[project] # package metadata
requires-python = ">=3.10"

[dependency-groups]
dev = ["databricks-connect~=16.0.0"]

[tool.custom] # user table
dev = ["databricks-connect==1.0.0"] # must not be managed
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	s := string(out)
	// [project].requires-python was found and updated despite the header comment.
	assert.Contains(t, s, `requires-python = "==3.12.*"`)
	// The real dev group was updated...
	assert.Contains(t, s, `"databricks-connect~=17.2.0"`)
	// ...but the lookalike dev under [tool.custom] was left untouched.
	assert.Contains(t, s, `dev = ["databricks-connect==1.0.0"] # must not be managed`)
	requireValidTOML(t, out)
}

func TestHeaderName(t *testing.T) {
	assert.Equal(t, "[project]", headerName("[project]"))
	assert.Equal(t, "[project]", headerName("  [project] # note"))
	assert.Equal(t, "[tool.uv]", headerName("[tool.uv]#x"))
	// Array-of-tables headers are distinct from their parent table.
	assert.Equal(t, "[[tool.uv.index]]", headerName("[[tool.uv.index]]"))
	assert.Equal(t, "[[tool.uv.index]]", headerName("  [[tool.uv.index]] # note"))
	assert.Empty(t, headerName("requires-python = \"3.12\""))
	assert.Empty(t, headerName("dev = [\"a\"]"))
}

func TestMergeToolUvWithArrayOfTablesChild(t *testing.T) {
	// A [tool.uv] table followed by its [[tool.uv.index]] array-of-tables child:
	// the managed constraint block must attach to [tool.uv], not leak into the
	// index item, and the result must be valid TOML.
	in := []byte(`[project]
requires-python = ">=3.10"

[dependency-groups]
dev = ["databricks-connect~=16.0.0"]

[tool.uv]

[[tool.uv.index]]
name = "internal"
url = "https://packages.example/simple"
`)
	out, _, err := MergeManaged(in, testConstraints())
	require.NoError(t, err)
	s := string(out)
	requireValidTOML(t, out)
	// The index array-of-tables is preserved intact.
	assert.Contains(t, s, `[[tool.uv.index]]`)
	assert.Contains(t, s, `name = "internal"`)
	assert.Contains(t, s, "pydantic~=2.10.6")
	// Merge-twice is byte-identical.
	twice, _, err := MergeManaged(out, testConstraints())
	require.NoError(t, err)
	assert.Equal(t, s, string(twice))
}

func TestMergeBailsOnMultilineString(t *testing.T) {
	// A TOML multi-line string can contain lines that look like headers/keys/
	// brackets; the line-based merge can't track it, so it must refuse rather than
	// risk corrupting the file. The body here contains a fake table header and a
	// fake constraint-dependencies line.
	in := []byte(`[project]
requires-python = ">=3.10"

[tool.uv]
note = """
[project]
constraint-dependencies = ["not really"]
"""

[dependency-groups]
dev = ["databricks-connect~=16.0.0"]
`)
	_, _, err := MergeManaged(in, testConstraints())
	require.Error(t, err)
	assert.ErrorIs(t, err, errMultilineString)
}

func TestMergeBailsWhenNoProjectTable(t *testing.T) {
	// A partial existing file with no [project] table can't receive requires-python;
	// merging it would silently skip the pin, so it must error instead.
	in := []byte(`[dependency-groups]
dev = ["databricks-connect~=16.0.0"]

[tool.uv]
constraint-dependencies = ["old~=1.0"]
`)
	_, _, err := MergeManaged(in, testConstraints())
	require.Error(t, err)
	assert.ErrorIs(t, err, errNoProjectTable)
}

func countOccurrences(s, substr string) int {
	return strings.Count(s, substr)
}
