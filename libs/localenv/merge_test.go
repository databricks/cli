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

func TestRenderFreshPyproject(t *testing.T) {
	out := RenderFreshPyproject("demo", testConstraints())
	s := string(out)
	assert.Contains(t, s, `name = "demo"`)
	assert.Contains(t, s, `requires-python = "==3.12.*"`)
	assert.Contains(t, s, `"databricks-connect~=17.2.0",`)
	assert.Contains(t, s, managedMarkerStart)
	assert.Contains(t, s, managedMarkerEnd)
	assert.Contains(t, s, "pydantic~=2.10.6")
	// A fresh render is itself a no-op under MergeManaged (already fully managed).
	merged, _, err := MergeManaged(out, testConstraints())
	require.NoError(t, err)
	assert.Equal(t, s, string(merged))
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

func TestMergeDatabricksConnectOnlyTouchesDevGroup(t *testing.T) {
	// A databricks-connect pin in a sibling group (docs) must be left alone; only
	// the dev group's entry is managed.
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
	// docs untouched; dev updated to the managed pin.
	assert.Contains(t, s, `docs = ["databricks-connect~=14.3"]`)
	assert.Contains(t, s, `"databricks-connect~=17.2.0",`)
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
	assert.Empty(t, headerName("requires-python = \"3.12\""))
	assert.Empty(t, headerName("dev = [\"a\"]"))
}

func countOccurrences(s, substr string) int {
	return strings.Count(s, substr)
}
