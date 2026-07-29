package bitmap

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeAppendOnly(t *testing.T) {
	old := []string{"a", "b", "c"}
	// fresh drops "b" (removed field) and adds "d" and "e".
	fresh := []string{"a", "c", "d", "e"}

	merged, added := Merge(old, fresh)

	// Removed fields stay to keep bit positions stable; new ones are appended.
	assert.Equal(t, []string{"a", "b", "c", "d", "e"}, merged)
	assert.Equal(t, []string{"d", "e"}, added)
}

func TestMergeNoChange(t *testing.T) {
	old := []string{"a", "b"}
	merged, added := Merge(old, []string{"a", "b"})
	assert.Equal(t, old, merged)
	assert.Empty(t, added)
}

func TestWalkSchemaPrunesTargets(t *testing.T) {
	schema, err := WalkSchema()
	require.NoError(t, err)
	require.NotEmpty(t, schema)

	for _, p := range schema {
		assert.NotEqual(t, "targets", p)
		assert.NotContains(t, p, "targets.")
		assert.NotContains(t, p, "environments.")
		assert.NotContains(t, p, "__locations")
	}

	assert.Contains(t, schema, "bundle.name")
	assert.Contains(t, schema, "resources.jobs.*.name")
}

func TestWalkSchemaEmitsValueBitForBool(t *testing.T) {
	schema, err := WalkSchema()
	require.NoError(t, err)

	// bundle.force is a bool: it gets both a presence entry and a #true value
	// entry, and they are adjacent.
	i := indexOf(schema, "bundle.force")
	require.GreaterOrEqual(t, i, 0)
	assert.Equal(t, "bundle.force"+valueSuffix, schema[i+1])
}

func TestBitsBoolTriState(t *testing.T) {
	schema := []string{
		"bundle.force",
		"bundle.force" + valueSuffix,
		"bundle.deployment.lock.enabled",
		"bundle.deployment.lock.enabled" + valueSuffix,
	}

	// lock.enabled is a *bool: unset here, so both its bits stay 0.
	// bundle.force is a plain bool set to true: presence and value both 1.
	var cfg config.Root
	cfg.Bundle.Force = true

	bits, err := Bits(cfg, schema)
	require.NoError(t, err)
	assert.Equal(t, []bool{true, true, false, false}, bits)
}

func TestBitsPointerBoolFalse(t *testing.T) {
	schema := []string{
		"bundle.deployment.lock.enabled",
		"bundle.deployment.lock.enabled" + valueSuffix,
	}

	// A *bool set to false: presence 1, value 0 (distinguishable from unset 0,0).
	no := false
	var cfg config.Root
	cfg.Bundle.Deployment.Lock.Enabled = &no

	bits, err := Bits(cfg, schema)
	require.NoError(t, err)
	assert.Equal(t, []bool{true, false}, bits)
}

func indexOf(s []string, v string) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}

func TestBitsSetsLeafAndPrefixes(t *testing.T) {
	schema := []string{
		"bundle",
		"bundle.name",
		"resources",
		"resources.jobs",
		"resources.jobs.*",
		"resources.jobs.*.name",
		"resources.jobs.*.tags",
		"resources.jobs.*.tags.*",
		"workspace.host",
	}

	var cfg config.Root
	cfg.Bundle.Name = "test-bundle"
	cfg.Resources.Jobs = map[string]*resources.Job{
		"my_job": {
			JobSettings: jobs.JobSettings{
				Name: "My Job",
				Tags: map[string]string{"team": "data"},
			},
		},
	}

	bits, err := Bits(cfg, schema)
	require.NoError(t, err)

	set := map[string]bool{}
	for i, p := range schema {
		set[p] = bits[i]
	}

	assert.True(t, set["bundle"])
	assert.True(t, set["bundle.name"])
	assert.True(t, set["resources"])
	assert.True(t, set["resources.jobs"])
	assert.True(t, set["resources.jobs.*"])
	assert.True(t, set["resources.jobs.*.name"])
	assert.True(t, set["resources.jobs.*.tags"])
	assert.True(t, set["resources.jobs.*.tags.*"])

	// Not set in the config.
	assert.False(t, set["workspace.host"])
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	bits := []bool{true, false, false, true, true, false, false, false, true}

	encoded, err := Encode(bits, ContextFullBundle)
	require.NoError(t, err)

	decoded, context, err := Decode(encoded)
	require.NoError(t, err)
	assert.Equal(t, ContextFullBundle, context)
	assert.Equal(t, bits, decoded)
}

func TestEncodeEmpty(t *testing.T) {
	encoded, err := Encode(nil, ContextFullBundle)
	require.NoError(t, err)

	decoded, _, err := Decode(encoded)
	require.NoError(t, err)
	assert.Empty(t, decoded)
}

func TestDecodeBadMagic(t *testing.T) {
	// Valid base64 + deflate but not our payload.
	_, _, err := Decode("not-valid-base64!!!")
	assert.Error(t, err)
}
