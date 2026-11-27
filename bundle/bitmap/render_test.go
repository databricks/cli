package bitmap

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
