package dbconnect

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func countOccurrences(s, substr string) int {
	return strings.Count(s, substr)
}
