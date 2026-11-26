package bitmap

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/binary"
	"math/rand/v2"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

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

// markedBits returns the bits a Builder sets for cfg, without telemetry.
func markedBits(t *testing.T, cfg dyn.Value, schema []string) []bool {
	t.Helper()
	b := NewBuilder(schema)
	b.Mark(cfg)
	bits, err := b.Bits(struct{}{})
	require.NoError(t, err)
	return bits
}

func loadValue(t *testing.T, yaml string) dyn.Value {
	t.Helper()
	root, diags := config.LoadFromBytes("databricks.yml", []byte(yaml))
	require.Empty(t, diags)
	return root.Value()
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
	// A variable whose name spells another schema path ("*.description")
	// counts as a variable, not as that path.
	odd, diags := config.LoadFromBytes("databricks.yml", []byte(`
variables:
  "*.description":
    default: x
`))
	require.Empty(t, diags)
	schema0 := EmbeddedSchema()
	assert.Equal(t, []string{"variables", "variables.*", "variables.*.default"}, setPaths(schema0, markedBits(t, odd.Value(), schema0)))

	schema := EmbeddedSchema()
	bits := markedBits(t, root.Value(), schema)

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

func TestBitsSkipsPrunedFields(t *testing.T) {
	root, diags := config.LoadFromBytes("databricks.yml", []byte(`
bundle:
  name: test-bundle
  target: dev
  environment: dev
targets:
  dev:
    workspace:
      host: https://host.test
`))
	require.Empty(t, diags)

	schema := EmbeddedSchema()
	bits := markedBits(t, root.Value(), schema)
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
			name: "bool true",
			yaml: "resources: {jobs: {j: {tasks: [{disable_auto_optimization: true}]}}}",
			want: []string{
				"resources", "resources.jobs", "resources.jobs.*", "resources.jobs.*.tasks", "resources.jobs.*.tasks[*]",
				"resources.jobs.*.tasks[*].disable_auto_optimization", "resources.jobs.*.tasks[*].disable_auto_optimization=true",
			},
		},
		{
			name: "bool with omitempty, false",
			yaml: "resources: {jobs: {j: {tasks: [{disable_auto_optimization: false}]}}}",
			want: []string{
				"resources", "resources.jobs", "resources.jobs.*", "resources.jobs.*.tasks", "resources.jobs.*.tasks[*]",
				"resources.jobs.*.tasks[*].disable_auto_optimization", "resources.jobs.*.tasks[*].disable_auto_optimization=false",
			},
		},
		{
			name: "bool without omitempty, false",
			yaml: "resources: {jobs: {j: {queue: {enabled: false}}}}",
			want: []string{
				"resources", "resources.jobs", "resources.jobs.*", "resources.jobs.*.queue",
				"resources.jobs.*.queue.enabled", "resources.jobs.*.queue.enabled=false",
			},
		},
		{
			name: "*bool true",
			yaml: "presets: {pipelines_development: true}",
			want: []string{"presets", "presets.pipelines_development", "presets.pipelines_development=true"},
		},
		{
			name: "*bool false",
			yaml: "presets: {pipelines_development: false}",
			want: []string{"presets", "presets.pipelines_development", "presets.pipelines_development=false"},
		},
		{
			name: "omitted",
			yaml: "presets: {name_prefix: dev_}",
			want: []string{"presets", "presets.name_prefix"},
		},
		{
			name: "unresolved reference has no value bit",
			yaml: `presets: {pipelines_development: "${var.flag}"}`,
			want: []string{"presets", "presets.pipelines_development"},
		},
		{
			name: "different values under a wildcard set both",
			yaml: "resources: {jobs: {a: {queue: {enabled: true}}, b: {queue: {enabled: false}}}}",
			want: []string{
				"resources", "resources.jobs", "resources.jobs.*", "resources.jobs.*.queue",
				"resources.jobs.*.queue.enabled", "resources.jobs.*.queue.enabled=false", "resources.jobs.*.queue.enabled=true",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, diags := config.LoadFromBytes("databricks.yml", []byte(tc.yaml))
			require.Empty(t, diags)
			bits := markedBits(t, root.Value(), schema)
			assert.Equal(t, tc.want, setPaths(schema, bits))
		})
	}
}

func TestBuilderMarkChanges(t *testing.T) {
	schema := EmbeddedSchema()
	job := []string{"resources", "resources.jobs", "resources.jobs.*"}
	for _, tc := range []struct {
		name          string
		before, after string
		want          []string
	}{
		{
			name:   "nothing changed",
			before: `resources: {jobs: {a: {name: x, max_concurrent_runs: 1}}}`,
			after:  `resources: {jobs: {a: {name: x, max_concurrent_runs: 1}}}`,
			want:   nil,
		},
		{
			name:   "a new resource is marked whole",
			before: `resources: {jobs: {a: {name: x}}}`,
			after:  `resources: {jobs: {a: {name: x}, b: {name: y, max_concurrent_runs: 1}}}`,
			want:   append(slices.Clone(job), "resources.jobs.*.name", "resources.jobs.*.max_concurrent_runs"),
		},
		{
			name:   "only the field added to an existing resource",
			before: `resources: {jobs: {a: {name: x, max_concurrent_runs: 1}}}`,
			after:  `resources: {jobs: {a: {name: x, max_concurrent_runs: 1, description: d}}}`,
			want:   append(slices.Clone(job), "resources.jobs.*.description"),
		},
		{
			name:   "a changed bool records its new value",
			before: `resources: {jobs: {a: {queue: {enabled: true}}}}`,
			after:  `resources: {jobs: {a: {queue: {enabled: false}}}}`,
			want:   append(slices.Clone(job), "resources.jobs.*.queue", "resources.jobs.*.queue.enabled", "resources.jobs.*.queue.enabled=false"),
		},
		{
			name:   "an entry added to a map",
			before: `resources: {jobs: {a: {tags: {team: x}}}}`,
			after:  `resources: {jobs: {a: {tags: {team: x, cost: y}}}}`,
			want:   append(slices.Clone(job), "resources.jobs.*.tags", "resources.jobs.*.tags.*"),
		},
		{
			name:   "list elements are paired by key",
			before: `resources: {jobs: {a: {tasks: [{task_key: t1}]}}}`,
			after:  `resources: {jobs: {a: {tasks: [{task_key: t1, description: d}, {task_key: t2}]}}}`,
			want: append(slices.Clone(job), "resources.jobs.*.tasks", "resources.jobs.*.tasks[*]",
				"resources.jobs.*.tasks[*].description", "resources.jobs.*.tasks[*].task_key"),
		},
		// The CLI filled in fields of t1 before Python ran; Python only added t0.
		{
			name:   "an element prepended to a keyed list",
			before: `resources: {jobs: {a: {tasks: [{task_key: t1, notebook_task: {notebook_path: p, source: WORKSPACE}}]}}}`,
			after:  `resources: {jobs: {a: {tasks: [{task_key: t0}, {task_key: t1, notebook_task: {notebook_path: p, source: WORKSPACE}}]}}}`,
			want:   append(slices.Clone(job), "resources.jobs.*.tasks", "resources.jobs.*.tasks[*]", "resources.jobs.*.tasks[*].task_key"),
		},
		{
			name:   "a keyed element that moved and changed",
			before: `resources: {jobs: {a: {tasks: [{task_key: t1}, {task_key: t2, max_retries: 1}]}}}`,
			after:  `resources: {jobs: {a: {tasks: [{task_key: t2, max_retries: 2}, {task_key: t1}]}}}`,
			want:   append(slices.Clone(job), "resources.jobs.*.tasks", "resources.jobs.*.tasks[*]", "resources.jobs.*.tasks[*].max_retries"),
		},
		{
			name:   "a list without keys that Python reorders",
			before: `resources: {jobs: {a: {permissions: [{level: CAN_VIEW, user_name: u}, {level: IS_OWNER, user_name: o}]}}}`,
			after:  `resources: {jobs: {a: {permissions: [{level: IS_OWNER, user_name: o}, {level: CAN_VIEW, user_name: u}]}}}`,
			want:   nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := NewBuilder(schema)
			b.MarkChanges(loadValue(t, tc.before), loadValue(t, tc.after))
			bits, err := b.Bits(struct{}{})
			require.NoError(t, err)
			want := slices.Clone(tc.want)
			slices.Sort(want)
			assert.Equal(t, want, setPaths(schema, bits))
		})
	}

	// A field written out as null is not set.
	before := loadValue(t, `resources: {jobs: {a: {name: x}}}`)
	after, err := dyn.SetByPath(before, dyn.MustPathFromString("resources.jobs.a.description"), dyn.NilValue)
	require.NoError(t, err)
	b := NewBuilder(schema)
	b.MarkChanges(before, after)
	bits, err := b.Bits(struct{}{})
	require.NoError(t, err)
	assert.Empty(t, setPaths(schema, bits))
}

func TestBuilderResolveVariables(t *testing.T) {
	schema := EmbeddedSchema()
	queue := []string{"resources", "resources.jobs", "resources.jobs.*", "resources.jobs.*.queue", "resources.jobs.*.queue.enabled"}
	for _, tc := range []struct {
		name      string
		config    string
		variables string
		want      []string
	}{
		{
			name:      "bool from a variable",
			config:    `resources: {jobs: {j: {queue: {enabled: "${var.flag}"}}}}`,
			variables: `variables: {flag: {value: false}}`,
			want:      append(slices.Clone(queue), "resources.jobs.*.queue.enabled=false"),
		},
		{
			name:      "bool from the long spelling of a variable reference",
			config:    `resources: {jobs: {j: {queue: {enabled: "${variables.flag.value}"}}}}`,
			variables: `variables: {flag: {value: false}}`,
			want:      append(slices.Clone(queue), "resources.jobs.*.queue.enabled=false"),
		},
		{
			name:      "bool from a variable's default",
			config:    `resources: {jobs: {j: {queue: {enabled: "${variables.flag.default}"}}}}`,
			variables: `variables: {flag: {default: true, value: false}}`,
			want:      append(slices.Clone(queue), "resources.jobs.*.queue.enabled=true"),
		},
		{
			name:      "bool from a string value, as --var passes it",
			config:    `resources: {jobs: {j: {queue: {enabled: "${var.flag}"}}}}`,
			variables: `variables: {flag: {value: "true"}}`,
			want:      append(slices.Clone(queue), "resources.jobs.*.queue.enabled=true"),
		},
		{
			name:      "bool from a field of a complex variable",
			config:    `resources: {jobs: {j: {queue: {enabled: "${var.settings.queue}"}}}}`,
			variables: `variables: {settings: {value: {queue: true}}}`,
			want:      append(slices.Clone(queue), "resources.jobs.*.queue.enabled=true"),
		},
		{
			name:      "bool from an element of a list variable",
			config:    `resources: {jobs: {j: {queue: {enabled: "${var.flags[1]}"}}}}`,
			variables: `variables: {flags: {value: [true, false]}}`,
			want:      append(slices.Clone(queue), "resources.jobs.*.queue.enabled=false"),
		},
		{
			name:      "undefined variable keeps presence only",
			config:    `resources: {jobs: {j: {queue: {enabled: "${var.missing}"}}}}`,
			variables: `variables: {flag: {value: true}}`,
			want:      queue,
		},
		{
			name:      "value that is not a bool keeps presence only",
			config:    `resources: {jobs: {j: {queue: {enabled: "${var.flag}"}}}}`,
			variables: `variables: {flag: {value: "maybe"}}`,
			want:      queue,
		},
		{
			name:      "other references are left alone",
			config:    `resources: {jobs: {j: {queue: {enabled: "${resources.jobs.other.queue.enabled}"}}}}`,
			variables: `resources: {jobs: {other: {queue: {enabled: true}}}}`,
			want:      queue,
		},
		{
			name:      "complex variable marks the fields inside it",
			config:    `resources: {jobs: {j: {job_clusters: "${var.clusters}"}}}`,
			variables: `variables: {clusters: {value: [{job_cluster_key: c, new_cluster: {num_workers: 1}}]}}`,
			want: []string{
				"resources", "resources.jobs", "resources.jobs.*", "resources.jobs.*.job_clusters", "resources.jobs.*.job_clusters[*]",
				"resources.jobs.*.job_clusters[*].job_cluster_key", "resources.jobs.*.job_clusters[*].new_cluster",
				"resources.jobs.*.job_clusters[*].new_cluster.num_workers",
			},
		},
		{
			name:      "quoted bool inside a complex variable",
			config:    `resources: {jobs: {j: {queue: "${var.q}"}}}`,
			variables: `variables: {q: {value: {enabled: "false"}}}`,
			want:      append(slices.Clone(queue), "resources.jobs.*.queue.enabled=false"),
		},
		{
			name:      "a variable key that spells a value entry",
			config:    `resources: {jobs: {j: {queue: "${var.q}"}}}`,
			variables: `variables: {q: {value: {"enabled=true": x, enabled: false}}}`,
			want:      append(slices.Clone(queue), "resources.jobs.*.queue.enabled=false"),
		},
		{
			name:      "interpolated string is a plain string",
			config:    `resources: {jobs: {j: {name: "job-${var.suffix}"}}}`,
			variables: `variables: {suffix: {value: {queue: {enabled: true}}}}`,
			want:      []string{"resources", "resources.jobs", "resources.jobs.*", "resources.jobs.*.name"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := NewBuilder(schema)
			b.Mark(loadValue(t, tc.config))
			b.ResolveVariables(loadValue(t, tc.variables))
			bits, err := b.Bits(struct{}{})
			require.NoError(t, err)
			want := slices.Clone(tc.want)
			slices.Sort(want)
			assert.Equal(t, want, setPaths(schema, bits))
		})
	}
}

func TestBitsRejectsUnsupportedTelemetryField(t *testing.T) {
	_, err := NewBuilder(EmbeddedSchema()).Bits(struct {
		Count int `metric:"count"`
	}{})
	assert.ErrorContains(t, err, "telemetry field Count has unsupported type int")
}

// A field without a metric tag is named by its Go name in snake case.
func TestTelemetryPathsWithoutMetricTag(t *testing.T) {
	paths, err := telemetryPaths(reflect.TypeFor[struct {
		Tagged         bool `metric:"legacy_key"`
		DMSCompatAuto  bool
		AppStarted     *bool
		SQLWarehouseOn BoolSetForTest
	}]())
	require.NoError(t, err)
	assert.Equal(t, []string{
		"telemetry.legacy_key",
		"telemetry.dms_compat_auto",
		"telemetry.app_started", "telemetry.app_started=true", "telemetry.app_started=false",
		"telemetry.sql_warehouse_on", "telemetry.sql_warehouse_on=true", "telemetry.sql_warehouse_on=false",
	}, paths)
}

func TestTelemetryPathsRejectsDuplicateNames(t *testing.T) {
	_, err := telemetryPaths(reflect.TypeFor[struct {
		SelectUsed bool
		Other      bool `metric:"select_used"`
	}]())
	assert.ErrorContains(t, err, "telemetry fields SelectUsed and Other are both named telemetry.select_used")
}

// BoolSetForTest stands in for bundle.BoolSet, which this package cannot import.
type BoolSetForTest struct{}

func (BoolSetForTest) Values() []bool { return nil }

func TestParseSchema(t *testing.T) {
	assert.Equal(t, []string{"variables", "bundle.name"}, parseSchema([]byte("variables\nbundle.name\n")))

	// Blank lines (leading, internal, trailing) are dropped so they can never
	// shift bit indices in the append-only schema.
	assert.Equal(t, []string{"variables", "bundle.name"}, parseSchema([]byte("\nvariables\n\nbundle.name\n\n")))
	assert.Empty(t, parseSchema([]byte("")))
	assert.Empty(t, parseSchema([]byte("\n\n")))
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	// Sizes around byte boundaries exercise the MSB-first packing and padding.
	for _, n := range []int{1, 7, 8, 9, 16, 17, MaxSize} {
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
			assert.Equal(t, bits, decoded)
		}
	}
	// The shape that lets the bitmap leave regulated regions takes 1 to MaxSize bits.
	for _, n := range []int{0, MaxSize + 1} {
		_, err := Encode(make([]bool, n), ContextFullBundle)
		assert.ErrorContains(t, err, "the encoding takes 1 to 32768", n)
	}
}

func TestEncodeBitOrder(t *testing.T) {
	encoded, err := Encode([]bool{true, false, false, false, false, false, false, true, true}, ContextFullBundle)
	require.NoError(t, err)

	body, ok := strings.CutPrefix(encoded, "bm1.")
	require.True(t, ok, encoded)
	compressed, err := base64.StdEncoding.DecodeString(body)
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
		{
			name:    "missing prefix",
			encoded: deflateBase64(t, append(header(magic, 1), 0x80)),
			err:     `missing "bm1." prefix`,
		},
		{
			name:    "bad base64",
			encoded: "bm1.not-valid-base64!!!",
			err:     "base64",
		},
		{
			name:    "not deflate",
			encoded: "bm1." + base64.StdEncoding.EncodeToString([]byte{0xff, 0xff}),
			err:     "inflate",
		},
		{
			name:    "short header",
			encoded: "bm1." + deflateBase64(t, []byte("DBTB")),
			err:     "payload too short",
		},
		{
			name:    "bad magic",
			encoded: "bm1." + deflateBase64(t, append(header("XXXX", 1), 0x80)),
			err:     "bad magic",
		},
		{
			name:    "missing bitmap bytes",
			encoded: "bm1." + deflateBase64(t, header(magic, 9)),
			err:     "payload size mismatch",
		},
		{
			name:    "extra bitmap bytes",
			encoded: "bm1." + deflateBase64(t, append(header(magic, 1), 0x80, 0x00)),
			err:     "payload size mismatch",
		},
		{
			name:    "size near 2^32",
			encoded: "bm1." + deflateBase64(t, header(magic, 0xFFFFFFFF)),
			err:     "payload size mismatch",
		},
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
	tm := struct{ SelectUsed bool }{SelectUsed: true}
	b.ReportAllocs()
	for b.Loop() {
		builder := NewBuilder(schema)
		builder.Mark(root.Value())
		_, err := builder.Bits(tm)
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
		{
			name: "nothing set",
			set:  nil,
			want: nil,
		},
		{
			name: "field",
			set:  []string{"bundle", "bundle.name"},
			want: []string{"P   bundle", "P   bundle.name"},
		},
		{
			name: "bool true",
			set:  []string{"flag", "flag=true"},
			want: []string{"PT  flag"},
		},
		{
			name: "bool false",
			set:  []string{"flag", "flag=false"},
			want: []string{"P F flag"},
		},
		{
			name: "both values under a wildcard",
			set:  []string{"jobs.*.enabled", "jobs.*.enabled=true", "jobs.*.enabled=false"},
			want: []string{"PTF jobs.*.enabled"},
		},
		{
			name: "unresolved reference",
			set:  []string{"ref"},
			want: []string{"P   ref"},
		},
		{
			name: "value without presence",
			set:  []string{"flag=true"},
			want: []string{" T  flag"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := append([]string{"size=12 context=0"}, tc.want...)
			assert.Equal(t, want, Text(schema, bitsFor(schema, tc.set), ContextFullBundle))
		})
	}

	// A bitmap from an older CLI has fewer bits; the rest of the schema is unset.
	assert.Equal(t, []string{"size=1 context=7", "P   bundle"}, Text(schema, []bool{true}, 7))

	// A bitmap from a newer CLI has more bits; a set one past the schema is a placeholder.
	newer := append(bitsFor(schema, []string{"bundle"}), false, true)
	assert.Equal(t, []string{"size=14 context=0", "P   bundle", "P   future #13"}, Text(schema, newer, ContextFullBundle))
}

func TestToJSON(t *testing.T) {
	schema := []string{
		"bundle", "bundle.name",
		"jobs", "jobs.*", "jobs.*.tasks", "jobs.*.tasks[*]",
		"jobs.*.tasks[*].enabled", "jobs.*.tasks[*].enabled=true", "jobs.*.tasks[*].enabled=false",
		"later",
	}
	job := func(key string, leaf any) map[string]any {
		return map[string]any{"jobs": map[string]any{"*": map[string]any{key: leaf}}}
	}
	enabled := job("tasks[*]", map[string]any{"enabled": true})
	for _, tc := range []struct {
		name                  string
		set                   []string
		presence, isTrue, isF map[string]any
	}{
		{
			name:     "nothing set",
			set:      nil,
			presence: map[string]any{},
			isTrue:   map[string]any{},
			isF:      map[string]any{},
		},
		{
			name:     "a parent with set children is a subtree",
			set:      []string{"bundle", "bundle.name"},
			presence: map[string]any{"bundle": map[string]any{"name": true}},
			isTrue:   map[string]any{},
			isF:      map[string]any{},
		},
		{
			name:     "a set field without set children is a leaf",
			set:      []string{"jobs", "jobs.*", "jobs.*.tasks"},
			presence: job("tasks", true),
			isTrue:   map[string]any{},
			isF:      map[string]any{},
		},
		{
			name:     "a list's elements replace its leaf",
			set:      []string{"jobs", "jobs.*", "jobs.*.tasks", "jobs.*.tasks[*]"},
			presence: job("tasks[*]", true),
			isTrue:   map[string]any{},
			isF:      map[string]any{},
		},
		{
			name:     "bool false",
			set:      []string{"jobs", "jobs.*", "jobs.*.tasks", "jobs.*.tasks[*]", "jobs.*.tasks[*].enabled", "jobs.*.tasks[*].enabled=false"},
			presence: enabled,
			isTrue:   map[string]any{},
			isF:      enabled,
		},
		{
			name:     "both values under a wildcard",
			set:      []string{"jobs", "jobs.*", "jobs.*.tasks", "jobs.*.tasks[*]", "jobs.*.tasks[*].enabled", "jobs.*.tasks[*].enabled=true", "jobs.*.tasks[*].enabled=false"},
			presence: enabled,
			isTrue:   enabled,
			isF:      enabled,
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

	// A bitmap from a newer CLI has more bits; a set one past the schema is a placeholder.
	got = ToJSON(schema, append(bitsFor(schema, []string{"bundle"}), false, true), ContextFullBundle)
	assert.Equal(t, JSON{Size: len(schema) + 2, Presence: map[string]any{"bundle": true, "future #11": true}, True: map[string]any{}, False: map[string]any{}}, got)

	// The list field's leaf is left out whichever of the two comes first.
	got = ToJSON([]string{"x[*]", "x"}, []bool{true, true}, ContextFullBundle)
	assert.Equal(t, map[string]any{"x[*]": true}, got.Presence)
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
	// "telemetry" namespaces the telemetry fields, and a dotted metric tag such
	// as "experimental.use_legacy_run_as" nests too; neither is a schema entry.
	assert.Equal(t, presence+len(implicitParents(schema)), countNodes(got.Presence))
	// A bool field has no children, so each value entry is a leaf.
	assert.Equal(t, trueValues, countLeaves(got.True))
	assert.Equal(t, falseValues, countLeaves(got.False))
}

// countNodes counts the schema entries a tree stands for: a "tasks[*]" node
// also stands for "tasks".
func countNodes(tree map[string]any) int {
	n := 0
	for k, v := range tree {
		n += 1 + strings.Count(k, listSuffix)
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

// The databricks-bundles decoder ships its own copy of the schema. A stale copy
// still decodes every bit it has, so only a byte comparison catches it.
func TestPythonSchemaMatchesEmbedded(t *testing.T) {
	python, err := os.ReadFile("../../python/databricks/bundles/_bitmap/schema.deflate")
	require.NoError(t, err)
	assert.True(t, bytes.Equal(schemaDeflate, python), "python/databricks/bundles/_bitmap/schema.deflate differs from bundle/bitmap/schema.deflate; run 'task generate-bitmap-schema'")
}

func TestNilBuilder(t *testing.T) {
	var b *Builder
	cfg := dyn.V(map[string]dyn.Value{"bundle": dyn.V(map[string]dyn.Value{"name": dyn.V("x")})})
	assert.NotPanics(t, func() {
		b.Mark(cfg)
		b.MarkChanges(dyn.InvalidValue, cfg)
		b.ResolveVariables(cfg)
	})
}

func TestCheckAppendOnly(t *testing.T) {
	assert.NoError(t, CheckAppendOnly([]string{"a", "b"}, []string{"a", "b"}))
	assert.NoError(t, CheckAppendOnly([]string{"a", "b"}, []string{"a", "b", "c"}))
	assert.NoError(t, CheckAppendOnly(nil, []string{"a"}))
	assert.ErrorContains(t, CheckAppendOnly([]string{"a", "b"}, []string{"b", "a"}), `bit 0 is "a" in the base schema but "b" now`)
	assert.ErrorContains(t, CheckAppendOnly([]string{"a", "b"}, []string{"a", "c", "b"}), `bit 1 is "b"`)
	assert.ErrorContains(t, CheckAppendOnly([]string{"a", "b"}, []string{"a"}), "fewer than the 2")
}

func TestDecompressSchemaRoundTrip(t *testing.T) {
	compressed, err := CompressSchema([]byte("a\nb\n"))
	require.NoError(t, err)
	schema, err := DecompressSchema(compressed)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b"}, schema)
}

// implicitParents returns the dotted prefixes of schema entries that are not
// entries themselves, which ToJSON still renders as nodes.
func implicitParents(schema []string) []string {
	entries := map[string]bool{}
	for _, p := range schema {
		entries[p] = true
	}
	var parents []string
	seen := map[string]bool{}
	for _, p := range schema {
		if strings.HasSuffix(p, trueSuffix) || strings.HasSuffix(p, falseSuffix) {
			continue
		}
		for i := range len(p) {
			if p[i] != '.' {
				continue
			}
			if prefix := p[:i]; !entries[prefix] && !seen[prefix] {
				seen[prefix] = true
				parents = append(parents, prefix)
			}
		}
	}
	return parents
}

// validShape is a copy of DATA_SHAPE_DABS_BUNDLE_BITMAP from universe
// compliance/shape: the format the logging pipeline lets out of regulated
// (FedRAMP) regions. A bitmap that fails it is redacted there.
func validShape(v string) bool {
	body, ok := strings.CutPrefix(v, "bm1.")
	if !ok || len(body) < 4 || len(body)%4 != 0 || len(body) > MaxEncodedBodyLen {
		return false
	}
	for i, c := range body {
		switch {
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '+', c == '/':
		case c == '=' && i >= len(body)-2 && (i == len(body)-1 || body[len(body)-1] == '='):
		default:
			return false
		}
	}
	return true
}

func TestEncodeFitsShape(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	sizes := []int{1, 2, 7, 8, 9, 100, len(EmbeddedSchema()), MaxSize}
	for range 50 {
		sizes = append(sizes, 1+r.IntN(MaxSize))
	}
	for _, size := range sizes {
		// Random bits compress worst; all set and none set are the extremes.
		for _, density := range []float64{0, 0.05, 0.5, 1} {
			bits := make([]bool, size)
			for i := range bits {
				bits[i] = r.Float64() < density
			}
			encoded, err := Encode(bits, ContextFullBundle)
			require.NoError(t, err)
			assert.True(t, validShape(encoded), "size %d density %v: %q fails the shape", size, density, encoded)
		}
	}
	for _, v := range []string{"bm1.orders_2024", "bm1.my.table", "BM1.AAAA", "bm2.AAAA", "bm1.AA=A", "bm1." + strings.Repeat("A", MaxEncodedBodyLen+4)} {
		assert.False(t, validShape(v), v)
	}
}
