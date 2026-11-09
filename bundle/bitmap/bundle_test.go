package bitmap_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/bitmap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests need bundle.Telemetry, and the bundle package imports this one,
// so they live in an external test package.

func TestWalkSchema(t *testing.T) {
	schema, err := bitmap.WalkSchema(reflect.TypeFor[bundle.Telemetry]())
	require.NoError(t, err)

	for _, p := range schema {
		assert.NotEqual(t, "targets", p)
		assert.NotContains(t, p, "targets.")
		assert.NotContains(t, p, "environments.")
		assert.NotContains(t, p, "__locations")
	}

	assert.Contains(t, schema, "bundle.name")
	assert.Contains(t, schema, "resources.jobs.*.name")
	assert.Contains(t, schema, "resources.jobs.*.tasks[*].task_key")
	assert.Contains(t, schema, "variables.*.default")
	// Telemetry fields are namespaced under "telemetry." and, having no json
	// tag, keep their Go field name.
	assert.Contains(t, schema, "telemetry.SelectUsed")
	assert.Contains(t, schema, "telemetry.RunAsSet")
	assert.Contains(t, schema, "telemetry.ArtifactFilesIsSet")
	assert.NotContains(t, schema, "telemetry.ArtifactFilesIsSet[*]")

	// Each bool field, whatever its Go shape, is followed by its value entries.
	for _, p := range []string{
		"experimental.use_legacy_run_as",                      // bool, omitempty
		"resources.jobs.*.queue.enabled",                      // bool, no omitempty
		"presets.pipelines_development",                       // *bool
		"telemetry.SelectUsed",                                // telemetry bool
		"telemetry.RunAsSet",                                  // telemetry *bool
		"telemetry.ArtifactFilesIsSet",                        // telemetry BoolSet
		"resources.jobs.*.tasks[*].disable_auto_optimization", // bool in a list element
	} {
		i := slices.Index(schema, p)
		require.GreaterOrEqual(t, i, 0, p)
		assert.Equal(t, []string{p + "=true", p + "=false"}, schema[i+1:i+3], p)
	}
	// Non-bool fields have no value entries.
	assert.NotContains(t, schema, "bundle.name=true")
	assert.NotContains(t, schema, "variables.*.default=true")
}

func TestEmbeddedSchemaIsCompleteAndUnique(t *testing.T) {
	schema := bitmap.EmbeddedSchema()
	fresh, err := bitmap.WalkSchema(reflect.TypeFor[bundle.Telemetry]())
	require.NoError(t, err)

	_, added := bitmap.Merge(schema, fresh)
	assert.Empty(t, added, "run 'task generate-bitmap-schema'")

	// A duplicate would leave its first bit permanently 0.
	seen := map[string]bool{}
	for _, p := range schema {
		assert.False(t, seen[p], "duplicate schema entry %q", p)
		seen[p] = true
	}
}

func TestBitsTelemetry(t *testing.T) {
	schema := bitmap.EmbeddedSchema()
	for _, tc := range []struct {
		name string
		tm   bundle.Telemetry
		want []string
	}{
		// Every field of a zero struct is unmeasured; none may count as present.
		{"zero value", bundle.Telemetry{}, nil},
		{"bool false", bundle.Telemetry{SelectUsed: false}, nil},
		{"bool true", bundle.Telemetry{SelectUsed: true}, []string{"telemetry.SelectUsed", "telemetry.SelectUsed=true"}},
		{"*bool nil", bundle.Telemetry{RunAsSet: nil}, nil},
		{"*bool false", bundle.Telemetry{RunAsSet: new(false)}, []string{"telemetry.RunAsSet", "telemetry.RunAsSet=false"}},
		{"*bool true", bundle.Telemetry{RunAsSet: new(true)}, []string{"telemetry.RunAsSet", "telemetry.RunAsSet=true"}},
		{"BoolSet unset", bundle.Telemetry{ArtifactFilesIsSet: boolSet()}, nil},
		{"BoolSet true twice", bundle.Telemetry{ArtifactFilesIsSet: boolSet(true, true)}, []string{
			"telemetry.ArtifactFilesIsSet", "telemetry.ArtifactFilesIsSet=true",
		}},
		{"BoolSet both values", bundle.Telemetry{ArtifactFilesIsSet: boolSet(true, false)}, []string{
			"telemetry.ArtifactFilesIsSet", "telemetry.ArtifactFilesIsSet=false", "telemetry.ArtifactFilesIsSet=true",
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bits, err := bitmap.NewBuilder(schema).Bits(tc.tm)
			require.NoError(t, err)
			assert.Equal(t, tc.want, setPaths(schema, bits))
		})
	}
}

func boolSet(values ...bool) bundle.BoolSet {
	var s bundle.BoolSet
	for _, v := range values {
		s.SetBool(v)
	}
	return s
}

func BenchmarkWalkSchema(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, err := bitmap.WalkSchema(reflect.TypeFor[bundle.Telemetry]())
		if err != nil {
			b.Fatal(err)
		}
	}
}

// setPaths returns the schema paths whose bit is set, sorted.
func setPaths(schema []string, bits []bool) []string {
	var out []string
	for i, p := range schema {
		if bits[i] {
			out = append(out, p)
		}
	}
	slices.Sort(out)
	return out
}
