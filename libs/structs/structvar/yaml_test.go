package structvar_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

// ev is an expected value with the locations it is expected to have.
// val is a scalar, a []ev or a map[string]ev.
type ev struct {
	val  any
	locs []diag.Location
}

func nv(val any, locs []diag.Location) ev {
	return ev{val: val, locs: locs}
}

// plain converts the expected value to the plain Go value the decoder produces.
func (e ev) plain() any {
	switch v := e.val.(type) {
	case []ev:
		out := make([]any, len(v))
		for i, x := range v {
			out[i] = x.plain()
		}
		return out
	case map[string]ev:
		out := make(map[string]any, len(v))
		for k, x := range v {
			out[k] = x.plain()
		}
		return out
	default:
		return v
	}
}

// assertLocations checks the locations of every value in the expected tree.
func assertLocations(t *testing.T, l *structvar.Locations, e ev, where string) {
	assert.Equal(t, e.locs, l.Get(), "locations of %s", where)
	switch v := e.val.(type) {
	case []ev:
		for i, x := range v {
			assertLocations(t, l.Index(i), x, fmt.Sprintf("%s[%d]", where, i))
		}
	case map[string]ev:
		for k, x := range v {
			assertLocations(t, l.Key(k), x, where+"."+k)
		}
	}
}

func decodeFile(t *testing.T, file string) (any, *structvar.StructVar, error) {
	input, err := os.ReadFile(file)
	require.NoError(t, err)
	var v any
	sv, _, err := structvar.DecodeYAML(file, bytes.NewBuffer(input), &v)
	return v, sv, err
}

func loadExample(t *testing.T, file string) (any, *structvar.StructVar) {
	v, sv, err := decodeFile(t, file)
	require.NoError(t, err)
	return v, sv
}

// assertDecoded checks the decoded value and the locations of all its parts.
func assertDecoded(t *testing.T, expected ev, v any, sv *structvar.StructVar) {
	assert.Equal(t, expected.plain(), v)
	assertLocations(t, sv.Locations, expected, "root")
}

// equalToReference compares the decoded value to the one decoded by yaml.v3.
// Timestamps are kept as the string they were written as, so only their presence is compared.
func equalToReference(t *testing.T, ref, v any) {
	switch r := ref.(type) {
	case time.Time:
		assert.IsType(t, "", v)
	case map[string]any:
		got, ok := v.(map[string]any)
		require.True(t, ok, "expected a map, got %T", v)
		require.Len(t, got, len(r))
		for k, x := range r {
			equalToReference(t, x, got[k])
		}
	case []any:
		got, ok := v.([]any)
		require.True(t, ok, "expected a sequence, got %T", v)
		require.Len(t, got, len(r))
		for i, x := range r {
			equalToReference(t, x, got[i])
		}
	default:
		assert.EqualValues(t, ref, v)
	}
}

// loadYAML decodes the file and checks that the result matches yaml.v3.
func loadYAML(t *testing.T, path string) (any, *structvar.StructVar) {
	input, err := os.ReadFile(path)
	require.NoError(t, err)

	var ref any
	err = yaml.Unmarshal(input, &ref)
	require.NoError(t, err)

	v, sv := loadExample(t, path)
	require.NotNil(t, sv)

	equalToReference(t, ref, v)
	return v, sv
}

// at returns the value at the path of keys (strings) and indexes (ints).
func at(t *testing.T, v any, path ...any) any {
	for _, p := range path {
		switch p := p.(type) {
		case string:
			m, ok := v.(map[string]any)
			require.True(t, ok, "expected a map at %q, got %T", p, v)
			v = m[p]
		case int:
			s, ok := v.([]any)
			require.True(t, ok, "expected a sequence at %d, got %T", p, v)
			v = s[p]
		}
	}
	return v
}

// locsAt returns the locations of the value at the path of keys (strings) and indexes (ints).
func locsAt(sv *structvar.StructVar, path ...any) []diag.Location {
	l := sv.Locations
	for _, p := range path {
		switch p := p.(type) {
		case string:
			l = l.Key(p)
		case int:
			l = l.Index(p)
		}
	}
	return l.Get()
}

func TestYAMLEmpty(t *testing.T) {
	v, sv := loadYAML(t, "testdata/yaml/empty.yml")
	assert.Nil(t, v)
	assert.Empty(t, sv.Locations.Get())
}
