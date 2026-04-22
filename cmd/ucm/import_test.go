package ucm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmd_Import_HappyPathTerraformEngine(t *testing.T) {
	h := newVerbHarness(t)

	stdout, stderr, err := runVerb(t, validFixtureDir(t), "import", "catalog", "team_alpha", "--auto-approve")
	t.Logf("stdout=%q stderr=%q", stdout, stderr)

	require.NoError(t, err)
	assert.Contains(t, stdout, "Imported catalog.team_alpha (team_alpha)")
	assert.Equal(t, 1, h.tf.RenderCalls)
	assert.Equal(t, 1, h.tf.InitCalls)
	assert.Equal(t, 1, h.tf.ImportCalls)
	assert.Equal(t, "databricks_catalog.team_alpha", h.tf.LastImportAddress)
	assert.Equal(t, "team_alpha", h.tf.LastImportId)
}

func TestCmd_Import_SchemaDefaultKeyIsLastPathSegment(t *testing.T) {
	h := newVerbHarness(t)

	_, _, err := runVerb(t, validFixtureDir(t), "import", "schema", "team_alpha.bronze", "--auto-approve")

	require.NoError(t, err)
	assert.Equal(t, 1, h.tf.ImportCalls)
	assert.Equal(t, "databricks_schema.bronze", h.tf.LastImportAddress)
	assert.Equal(t, "team_alpha.bronze", h.tf.LastImportId)
}

func TestCmd_Import_KeyFlagOverridesDefault(t *testing.T) {
	h := newVerbHarness(t)

	_, _, err := runVerb(t, validFixtureDir(t), "import", "catalog", "team_alpha", "--key", "team_alpha", "--auto-approve")

	require.NoError(t, err)
	assert.Equal(t, "databricks_catalog.team_alpha", h.tf.LastImportAddress)
}

func TestCmd_Import_UnknownKindFails(t *testing.T) {
	_ = newVerbHarness(t)

	_, _, err := runVerb(t, validFixtureDir(t), "import", "table", "foo", "--auto-approve")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported kind")
}

func TestCmd_Import_RequiresDeclaredResource(t *testing.T) {
	_ = newVerbHarness(t)

	stdout, stderr, err := runVerb(t, validFixtureDir(t), "import", "catalog", "missing_catalog", "--auto-approve")
	t.Logf("stdout=%q stderr=%q", stdout, stderr)

	require.Error(t, err)
	assert.Contains(t, stderr, "not declared in ucm.yml")
}

func TestCmd_Import_PropagatesImportError(t *testing.T) {
	h := newVerbHarness(t)
	h.tf.ImportErr = assertSentinel

	_, _, err := runVerb(t, validFixtureDir(t), "import", "catalog", "team_alpha", "--auto-approve")

	require.Error(t, err)
	assert.Equal(t, 1, h.tf.ImportCalls)
}
