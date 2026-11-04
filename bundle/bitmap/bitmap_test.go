package bitmap

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/binary"
	"reflect"
	"slices"
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
	assert.Contains(t, schema, "telemetry.RunAsSet.True")
	assert.Contains(t, schema, "telemetry.RunAsSet.False")
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

func TestBitsTelemetry(t *testing.T) {
	schema := EmbeddedSchema()

	// A zero-valued struct sets no bits: Walk visits every tagless field, and
	// Bits must not treat that visit as presence.
	bits, err := Bits(dyn.NilValue, bundle.Telemetry{}, schema)
	require.NoError(t, err)
	assert.Empty(t, setPaths(schema, bits))

	var tm bundle.Telemetry
	tm.SelectUsed = true
	tm.RunAsSet.Set(false)
	bits, err = Bits(dyn.NilValue, tm, schema)
	require.NoError(t, err)
	assert.Equal(t, []string{
		"telemetry.RunAsSet",
		"telemetry.RunAsSet.False",
		"telemetry.SelectUsed",
	}, setPaths(schema, bits))
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
