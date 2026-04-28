package ucm

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runValidate is a thin wrapper around runVerbInDir that runs `validate` from
// fixtureDir. The fixture's ucm.yml is read in place; the cwd is restored on
// test cleanup. Returns stdout, the cmdio-captured stderr buffer, and the
// cobra.Execute error.
func runValidate(t *testing.T, fixtureDir string, extraArgs ...string) (string, string, error) {
	t.Helper()
	return runVerbInDir(t, fixtureDir, append([]string{"validate"}, extraArgs...)...)
}

func TestCmd_Validate_ValidFixturePasses(t *testing.T) {
	stdout, stderr, err := runValidate(t, filepath.Join("testdata", "valid"))
	t.Logf("stdout=%q", stdout)
	t.Logf("stderr=%q", stderr)
	require.NoError(t, err)
	assert.Contains(t, stdout, "Validation OK!")
}

func TestCmd_Validate_MissingTagFixtureFails(t *testing.T) {
	stdout, stderr, err := runValidate(t, filepath.Join("testdata", "missing_tag"))
	require.Error(t, err)
	// Trailer summarises the count; per-diagnostic lines are streamed to stderr.
	assert.Contains(t, stdout, "Found ")
	assert.Contains(t, stdout, "error")
	assert.Contains(t, stderr, "requires tag")
}

func TestCmd_Validate_JSONModeProducesValidJSON(t *testing.T) {
	stdout, _, err := runValidate(t, filepath.Join("testdata", "valid"), "--output", "json")
	require.NoError(t, err)

	var tree map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &tree))
	_, ok := tree["resources"]
	assert.True(t, ok, "JSON output should contain resources subtree")
	assert.NotContains(t, stdout, "Validation OK!")
}

func TestCmd_Validate_JSONModeOnErrorFixture(t *testing.T) {
	stdout, _, err := runValidate(t, filepath.Join("testdata", "missing_tag"), "--output", "json")
	require.Error(t, err)
	// Cobra may append nothing else with SilenceUsage, so stdout is pure JSON.
	assert.Contains(t, stdout, `"resources"`)
	assert.Contains(t, stdout, `"catalogs"`)
}

func TestCmd_Validate_IncludeLocationsOffOmitsLocationsKey(t *testing.T) {
	stdout, _, err := runValidate(t, filepath.Join("testdata", "valid"), "--output", "json")
	require.NoError(t, err)

	var tree map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &tree))
	_, ok := tree["__locations"]
	assert.False(t, ok, "default validate JSON should not contain __locations")
}

func TestCmd_Validate_IncludeLocationsOnAddsLocationsKey(t *testing.T) {
	stdout, _, err := runValidate(t, filepath.Join("testdata", "valid"), "--output", "json", "--include-locations")
	require.NoError(t, err)

	var tree map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &tree))
	locs, ok := tree["__locations"].(map[string]any)
	require.True(t, ok, "expected __locations in JSON output: %s", stdout)
	assert.Contains(t, locs, "files")
	assert.Contains(t, locs, "locations")
}

func TestCmd_Validate_NestedFixturePasses(t *testing.T) {
	stdout, stderr, err := runValidate(t, filepath.Join("testdata", "nested"))
	t.Logf("stdout=%q", stdout)
	t.Logf("stderr=%q", stderr)
	require.NoError(t, err)
	assert.Contains(t, stdout, "Validation OK!")
}

func TestCmd_Validate_CollisionFixtureFails(t *testing.T) {
	_, stderr, err := runValidate(t, filepath.Join("testdata", "collision"))
	require.Error(t, err)
	assert.Contains(t, stderr, "declared both as a flat entry and nested")
}

func TestCmd_Validate_InheritOptOutFailsTagRule(t *testing.T) {
	_, stderr, err := runValidate(t, filepath.Join("testdata", "inherit_opt_out"))
	require.Error(t, err)
	assert.Contains(t, stderr, "requires tag")
}

func TestCmd_Schema_ProducesValidJSON(t *testing.T) {
	stdout, _, err := runVerbInDir(t, t.TempDir(), "schema")
	require.NoError(t, err)

	// The output must parse as JSON and declare an object at the root.
	var schema map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &schema))

	// Should at minimum advertise the ucm root in the $defs tree.
	assert.True(t, strings.Contains(stdout, "resources.Catalog"), "schema should describe Catalog")
	assert.True(t, strings.Contains(stdout, "resources.TagValidationRule"), "schema should describe TagValidationRule")
}
