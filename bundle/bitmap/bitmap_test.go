package bitmap

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/binary"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/dyn"
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

func TestWalkSchema(t *testing.T) {
	schema, err := WalkSchema(reflect.TypeFor[bundle.Telemetry]())
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
	assert.Contains(t, schema, "telemetry.ArtifactFilesIsSet[*]")

	// Each bool field, whatever its Go shape, is followed by its value entries.
	for _, p := range []string{
		"experimental.use_legacy_run_as",                      // bool, omitempty
		"resources.jobs.*.queue.enabled",                      // bool, no omitempty
		"presets.pipelines_development",                       // *bool
		"telemetry.SelectUsed",                                // telemetry bool
		"telemetry.RunAsSet",                                  // telemetry *bool
		"telemetry.ArtifactFilesIsSet[*]",                     // telemetry []bool element
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
	schema := EmbeddedSchema()
	fresh, err := WalkSchema(reflect.TypeFor[bundle.Telemetry]())
	require.NoError(t, err)

	_, added := Merge(schema, fresh)
	assert.Empty(t, added, "run 'task generate-bitmap-schema'")

	// A duplicate would leave its first bit permanently 0.
	seen := map[string]bool{}
	for _, p := range schema {
		assert.False(t, seen[p], "duplicate schema entry %q", p)
		seen[p] = true
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

func TestBitsPresence(t *testing.T) {
	root, diags := config.LoadFromBytes("databricks.yml", []byte(`
bundle:
  name: test-bundle
variables:
  scalar:
    default: abc
  complex:
    type: complex
    default:
      key: value
resources:
  jobs:
    my_job:
      name: My Job
      timeout_seconds: 0
      email_notifications: {}
      tags:
        "a.b": c
      tasks: []
`))
	require.Empty(t, diags)

	schema := EmbeddedSchema()
	bits, err := Bits(root.Value(), bundle.Telemetry{}, schema)
	require.NoError(t, err)

	assert.Equal(t, []string{
		"bundle",
		"bundle.name",
		"resources",
		"resources.jobs",
		"resources.jobs.*",
		// Empty container: present, with no children.
		"resources.jobs.*.email_notifications",
		"resources.jobs.*.name",
		// A map key with a dot still maps to the wildcard.
		"resources.jobs.*.tags",
		"resources.jobs.*.tags.*",
		// Empty list: present, with no elements.
		"resources.jobs.*.tasks",
		// Explicit zero: present.
		"resources.jobs.*.timeout_seconds",
		"variables",
		"variables.*",
		// Interface-typed field: present; its contents are not descended into.
		"variables.*.default",
		"variables.*.type",
	}, setPaths(schema, bits))
}

func TestBitsSkipsPrunedTargets(t *testing.T) {
	root, diags := config.LoadFromBytes("databricks.yml", []byte(`
bundle:
  name: test-bundle
targets:
  dev:
    workspace:
      host: https://host.test
`))
	require.Empty(t, diags)

	schema := EmbeddedSchema()
	bits, err := Bits(root.Value(), bundle.Telemetry{}, schema)
	require.NoError(t, err)
	assert.Equal(t, []string{"bundle", "bundle.name"}, setPaths(schema, bits))
}

func TestBitsBoolValues(t *testing.T) {
	schema := EmbeddedSchema()
	for _, tc := range []struct {
		name string
		yaml string
		want []string
	}{
		{
			"bool true",
			"resources: {jobs: {j: {tasks: [{disable_auto_optimization: true}]}}}",
			[]string{
				"resources", "resources.jobs", "resources.jobs.*", "resources.jobs.*.tasks", "resources.jobs.*.tasks[*]",
				"resources.jobs.*.tasks[*].disable_auto_optimization", "resources.jobs.*.tasks[*].disable_auto_optimization=true",
			},
		},
		{
			"bool with omitempty, false",
			"resources: {jobs: {j: {tasks: [{disable_auto_optimization: false}]}}}",
			[]string{
				"resources", "resources.jobs", "resources.jobs.*", "resources.jobs.*.tasks", "resources.jobs.*.tasks[*]",
				"resources.jobs.*.tasks[*].disable_auto_optimization", "resources.jobs.*.tasks[*].disable_auto_optimization=false",
			},
		},
		{
			"bool without omitempty, false",
			"resources: {jobs: {j: {queue: {enabled: false}}}}",
			[]string{
				"resources", "resources.jobs", "resources.jobs.*", "resources.jobs.*.queue",
				"resources.jobs.*.queue.enabled", "resources.jobs.*.queue.enabled=false",
			},
		},
		{
			"*bool true",
			"presets: {pipelines_development: true}",
			[]string{"presets", "presets.pipelines_development", "presets.pipelines_development=true"},
		},
		{
			"*bool false",
			"presets: {pipelines_development: false}",
			[]string{"presets", "presets.pipelines_development", "presets.pipelines_development=false"},
		},
		{
			"omitted",
			"presets: {name_prefix: dev_}",
			[]string{"presets", "presets.name_prefix"},
		},
		{
			"unresolved reference has no value bit",
			`presets: {pipelines_development: "${var.flag}"}`,
			[]string{"presets", "presets.pipelines_development"},
		},
		{
			"different values under a wildcard set both",
			"resources: {jobs: {a: {queue: {enabled: true}}, b: {queue: {enabled: false}}}}",
			[]string{
				"resources", "resources.jobs", "resources.jobs.*", "resources.jobs.*.queue",
				"resources.jobs.*.queue.enabled", "resources.jobs.*.queue.enabled=false", "resources.jobs.*.queue.enabled=true",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, diags := config.LoadFromBytes("databricks.yml", []byte(tc.yaml))
			require.Empty(t, diags)
			bits, err := Bits(root.Value(), bundle.Telemetry{}, schema)
			require.NoError(t, err)
			assert.Equal(t, tc.want, setPaths(schema, bits))
		})
	}
}

func TestBitsTelemetry(t *testing.T) {
	schema := EmbeddedSchema()
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
		{"[]bool empty", bundle.Telemetry{ArtifactFilesIsSet: []bool{}}, nil},
		{"[]bool all true", bundle.Telemetry{ArtifactFilesIsSet: []bool{true, true}}, []string{
			"telemetry.ArtifactFilesIsSet", "telemetry.ArtifactFilesIsSet[*]", "telemetry.ArtifactFilesIsSet[*]=true",
		}},
		{"[]bool mixed", bundle.Telemetry{ArtifactFilesIsSet: []bool{true, false}}, []string{
			"telemetry.ArtifactFilesIsSet", "telemetry.ArtifactFilesIsSet[*]",
			"telemetry.ArtifactFilesIsSet[*]=false", "telemetry.ArtifactFilesIsSet[*]=true",
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bits, err := Bits(dyn.NilValue, tc.tm, schema)
			require.NoError(t, err)
			assert.Equal(t, tc.want, setPaths(schema, bits))
		})
	}
}

func TestBitsRejectsUnsupportedTelemetryField(t *testing.T) {
	_, err := Bits(dyn.NilValue, struct{ Count int }{}, EmbeddedSchema())
	assert.ErrorContains(t, err, "telemetry field Count has unsupported type int")
}

func TestParseSchema(t *testing.T) {
	// A CRLF checkout of schema.txt (Windows) must parse to the same clean field
	// paths as an LF one, so Merge's dedup matches and the schema does not double.
	assert.Equal(t, []string{"variables", "bundle.name"}, parseSchema([]byte("variables\r\nbundle.name\r\n")))
	assert.Equal(t, []string{"variables", "bundle.name"}, parseSchema([]byte("variables\nbundle.name\n")))

	// Blank lines (leading, internal, trailing) are dropped so they can never
	// shift bit indices in the append-only schema.
	assert.Equal(t, []string{"variables", "bundle.name"}, parseSchema([]byte("\nvariables\n\nbundle.name\n\n")))
	assert.Empty(t, parseSchema([]byte("")))
	assert.Empty(t, parseSchema([]byte("\n\n")))
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	// Sizes around byte boundaries exercise the MSB-first packing and padding.
	for _, n := range []int{0, 1, 7, 8, 9, 16, 17} {
		bits := make([]bool, n)
		for i := range bits {
			bits[i] = i%3 == 0
		}
		for _, context := range []uint16{ContextFullBundle, 0xBEEF} {
			encoded, err := Encode(bits, context)
			require.NoError(t, err)

			decoded, gotContext, err := Decode(encoded)
			require.NoError(t, err)
			assert.Equal(t, context, gotContext)
			assert.Len(t, decoded, n)
			if n > 0 {
				assert.Equal(t, bits, decoded)
			}
		}
	}
}

func TestEncodeBitOrder(t *testing.T) {
	encoded, err := Encode([]bool{true, false, false, false, false, false, false, true, true}, ContextFullBundle)
	require.NoError(t, err)

	compressed, err := base64.StdEncoding.DecodeString(encoded)
	require.NoError(t, err)
	var raw bytes.Buffer
	_, err = raw.ReadFrom(flate.NewReader(bytes.NewReader(compressed)))
	require.NoError(t, err)

	assert.Equal(t, []byte{'D', 'B', 'T', 'B', 0, 0, 0, 9, 0, 0, 0b1000_0001, 0b1000_0000}, raw.Bytes())
}

// deflateBase64 wraps a raw payload the way Encode does, so Decode's header
// checks can be exercised on hand-built payloads.
func deflateBase64(t *testing.T, raw []byte) string {
	var buf bytes.Buffer
	w, err := flate.NewWriter(&buf, flate.BestCompression)
	require.NoError(t, err)
	_, err = w.Write(raw)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func header(magic string, size uint32) []byte {
	raw := make([]byte, headerSize)
	copy(raw, magic)
	binary.BigEndian.PutUint32(raw[4:], size)
	return raw
}

func TestDecodeErrors(t *testing.T) {
	for _, tc := range []struct {
		name    string
		encoded string
		err     string
	}{
		{"bad base64", "not-valid-base64!!!", "base64"},
		{"not deflate", base64.StdEncoding.EncodeToString([]byte{0xff, 0xff}), "inflate"},
		{"short header", deflateBase64(t, []byte("DBTB")), "payload too short"},
		{"bad magic", deflateBase64(t, append(header("XXXX", 1), 0x80)), "bad magic"},
		{"missing bitmap bytes", deflateBase64(t, header(magic, 9)), "payload size mismatch"},
		{"extra bitmap bytes", deflateBase64(t, append(header(magic, 1), 0x80, 0x00)), "payload size mismatch"},
		{"size near 2^32", deflateBase64(t, header(magic, 0xFFFFFFFF)), "payload size mismatch"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := Decode(tc.encoded)
			assert.ErrorContains(t, err, tc.err)
		})
	}
}

func BenchmarkBits(b *testing.B) {
	schema := EmbeddedSchema()
	root, diags := config.LoadFromBytes("databricks.yml", []byte(`
bundle:
  name: bench
resources:
  jobs:
    j:
      name: My Job
      tags: {team: data, env: prod}
      tasks:
        - task_key: a
          notebook_task: {notebook_path: /nb}
        - task_key: b
          new_cluster: {num_workers: 2}
`))
	require.Empty(b, diags)
	var tm bundle.Telemetry
	tm.SelectUsed = true
	b.ReportAllocs()
	for b.Loop() {
		_, err := Bits(root.Value(), tm, schema)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWalkSchema(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, err := WalkSchema(reflect.TypeFor[bundle.Telemetry]())
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestText(t *testing.T) {
	schema := []string{
		"bundle", "bundle.name",
		"flag", "flag=true", "flag=false",
		"jobs.*.enabled", "jobs.*.enabled=true", "jobs.*.enabled=false",
		"ref", "ref=true", "ref=false",
		"later",
	}
	for _, tc := range []struct {
		name string
		set  []string
		want []string
	}{
		{"nothing set", nil, nil},
		{"field", []string{"bundle", "bundle.name"}, []string{"1 bundle", "1 bundle.name"}},
		{"bool true", []string{"flag", "flag=true"}, []string{"T flag"}},
		{"bool false", []string{"flag", "flag=false"}, []string{"F flag"}},
		{"both values under a wildcard", []string{"jobs.*.enabled", "jobs.*.enabled=true", "jobs.*.enabled=false"}, []string{"T jobs.*.enabled", "F jobs.*.enabled"}},
		{"unresolved reference", []string{"ref"}, []string{"1 ref"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := append([]string{"size=12 context=0"}, tc.want...)
			assert.Equal(t, want, Text(schema, bitsFor(schema, tc.set), ContextFullBundle))
		})
	}

	// A bitmap from an older CLI has fewer bits; the rest of the schema is unset.
	assert.Equal(t, []string{"size=1 context=7", "1 bundle"}, Text(schema, []bool{true}, 7))
}

func TestToJSON(t *testing.T) {
	schema := []string{
		"bundle", "bundle.name",
		"jobs", "jobs.*", "jobs.*.tasks", "jobs.*.tasks[*]",
		"jobs.*.tasks[*].enabled", "jobs.*.tasks[*].enabled=true", "jobs.*.tasks[*].enabled=false",
		"later",
	}
	tasks := func(leaf any) map[string]any {
		return map[string]any{"jobs": map[string]any{"*": map[string]any{"tasks": leaf}}}
	}
	enabled := tasks(map[string]any{"[*]": map[string]any{"enabled": true}})
	for _, tc := range []struct {
		name                  string
		set                   []string
		presence, isTrue, isF map[string]any
	}{
		{"nothing set", nil, map[string]any{}, map[string]any{}, map[string]any{}},
		{
			"a parent with set children is a subtree",
			[]string{"bundle", "bundle.name"},
			map[string]any{"bundle": map[string]any{"name": true}},
			map[string]any{},
			map[string]any{},
		},
		{
			"a set field without set children is a leaf",
			[]string{"jobs", "jobs.*", "jobs.*.tasks"},
			tasks(true),
			map[string]any{},
			map[string]any{},
		},
		{
			"bool false",
			[]string{"jobs", "jobs.*", "jobs.*.tasks", "jobs.*.tasks[*]", "jobs.*.tasks[*].enabled", "jobs.*.tasks[*].enabled=false"},
			enabled,
			map[string]any{},
			enabled,
		},
		{
			"both values under a wildcard",
			[]string{"jobs", "jobs.*", "jobs.*.tasks", "jobs.*.tasks[*]", "jobs.*.tasks[*].enabled", "jobs.*.tasks[*].enabled=true", "jobs.*.tasks[*].enabled=false"},
			enabled, enabled, enabled,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ToJSON(schema, bitsFor(schema, tc.set), ContextFullBundle)
			assert.Equal(t, JSON{Size: len(schema), Presence: tc.presence, True: tc.isTrue, False: tc.isF}, got)
		})
	}

	// A bitmap from an older CLI has fewer bits; the rest of the schema is unset.
	got := ToJSON(schema, []bool{true}, 7)
	assert.Equal(t, JSON{Size: 1, Context: 7, Presence: map[string]any{"bundle": true}, True: map[string]any{}, False: map[string]any{}}, got)
}

// TestToJSONIsLossless checks that every entry of the embedded schema maps to
// its own tree node, so the JSON form keeps every set bit.
func TestToJSONIsLossless(t *testing.T) {
	schema := EmbeddedSchema()
	bits := make([]bool, len(schema))
	for i := range bits {
		bits[i] = true
	}
	got := ToJSON(schema, bits, ContextFullBundle)

	var presence, trueValues, falseValues int
	for _, p := range schema {
		switch {
		case strings.HasSuffix(p, trueSuffix):
			trueValues++
		case strings.HasSuffix(p, falseSuffix):
			falseValues++
		default:
			presence++
		}
	}
	// "telemetry" namespaces the telemetry fields but is not a schema entry.
	assert.Equal(t, presence+1, countNodes(got.Presence))
	// A bool field has no children, so each value entry is a leaf.
	assert.Equal(t, trueValues, countLeaves(got.True))
	assert.Equal(t, falseValues, countLeaves(got.False))
}

func countNodes(tree map[string]any) int {
	n := len(tree)
	for _, v := range tree {
		if child, ok := v.(map[string]any); ok {
			n += countNodes(child)
		}
	}
	return n
}

func countLeaves(tree map[string]any) int {
	n := 0
	for _, v := range tree {
		if child, ok := v.(map[string]any); ok {
			n += countLeaves(child)
		} else {
			n++
		}
	}
	return n
}

// bitsFor returns bits with the given schema paths set.
func bitsFor(schema, set []string) []bool {
	bits := make([]bool, len(schema))
	for _, p := range set {
		bits[slices.Index(schema, p)] = true
	}
	return bits
}
