package bitmap

import (
	"slices"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestNilBuilder(t *testing.T) {
	var b *Builder
	cfg := dyn.V(map[string]dyn.Value{"bundle": dyn.V(map[string]dyn.Value{"name": dyn.V("x")})})
	assert.NotPanics(t, func() {
		b.Mark(cfg)
		b.MarkChanges(dyn.InvalidValue, cfg)
		b.ResolveVariables(cfg)
	})
}
