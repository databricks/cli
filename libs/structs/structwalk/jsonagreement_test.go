package structwalk_test

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/databricks/cli/libs/structs/internal/jsonshapes"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structwalk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWalkVisitsExactlyWhatJSONEmits checks the value walk against encoding/json over the
// shape corpus. A path structwalk does not visit is a path structdiff cannot report drift
// on, and a path it visits but the wire format drops is a change that can never be sent.
func TestWalkVisitsExactlyWhatJSONEmits(t *testing.T) {
	for _, shape := range jsonshapes.Shapes() {
		t.Run(shape.Name, func(t *testing.T) {
			var visited []string
			require.NoError(t, structwalk.Walk(shape.Value, func(path *structpath.PathNode, _ any, _ *reflect.StructField) {
				visited = append(visited, path.String())
			}))
			slices.Sort(visited)

			blob, err := json.Marshal(shape.Value)
			require.NoError(t, err)
			emitted, err := jsonshapes.Leaves(shape.Value)
			require.NoError(t, err)
			var want []string
			for name := range emitted {
				want = append(want, name)
			}
			slices.Sort(want)

			if shape.WalkGap != nil {
				// A recorded gap holds the exact current output, so a different wrong answer
				// fails here too rather than passing as "still broken".
				gap := slices.Clone(shape.WalkGap)
				slices.Sort(gap)
				assert.Equal(t, gap, visited,
					"structwalk.Walk changed here; encoding/json emits %v -- update or remove the recorded gap", want)
				return
			}

			assert.Equal(t, want, visited, "encoding/json emitted %s", blob)

			for _, name := range shape.Unreachable {
				assert.NotContains(t, visited, name,
					"%q never reaches the wire, so the walk must not offer it", name)
			}
		})
	}
}

// isScalarKind mirrors what the value walk treats as a leaf, so the two walks are compared
// on the same footing.
func isScalarKind(k reflect.Kind) bool {
	switch k {
	case reflect.Bool, reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// TestWalkTypeCoversTheType checks the type walk, which differs from the value walk only
// where a value cannot reach what its type declares -- an embedded nil pointer contributes
// its fields to the type and nothing to the wire.
func TestWalkTypeCoversTheType(t *testing.T) {
	for _, shape := range jsonshapes.Shapes() {
		t.Run(shape.Name, func(t *testing.T) {
			var visited []string
			require.NoError(t, structwalk.WalkType(reflect.TypeOf(shape.Value), func(path *structpath.PatternNode, typ reflect.Type, _ *reflect.StructField) bool {
				if isScalarKind(typ.Kind()) {
					visited = append(visited, path.String())
				}
				return true
			}))
			slices.Sort(visited)

			want := append([]string(nil), shape.Fields()...)
			slices.Sort(want)
			if shape.WalkTypeGap != nil {
				gap := slices.Clone(shape.WalkTypeGap)
				slices.Sort(gap)
				assert.Equal(t, gap, visited,
					"structwalk.WalkType changed here; the type declares %v -- update or remove the recorded gap", want)
				return
			}
			assert.Equal(t, want, visited)

			for _, name := range shape.Unreachable {
				assert.NotContains(t, visited, name,
					"%q never reaches the wire, so the type walk must not offer it", name)
			}
		})
	}
}
