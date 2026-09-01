package structdiff_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/databricks/cli/libs/structs/internal/jsonshapes"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDiffReportsWhatJSONCarries checks that a change to a field the wire format carries
// shows up in the diff, at the path encoding/json puts it at. A field structdiff cannot see
// is a field the direct engine never sends an update for.
func TestDiffReportsWhatJSONCarries(t *testing.T) {
	for _, shape := range jsonshapes.Shapes() {
		t.Run(shape.Name, func(t *testing.T) {
			for _, name := range shape.JSONFields {
				before := shape.Value
				after := cloneWith(t, shape, name, "changed")
				if after == nil {
					continue // recorded write gap; the read side is covered in structaccess
				}

				changes, err := structdiff.GetStructDiff(before, after, nil)
				require.NoError(t, err)

				var paths []string
				for _, change := range changes {
					paths = append(paths, change.Path.String())
				}
				assert.Contains(t, paths, name,
					"%q changed but structdiff did not report it", name)
			}
		})
	}
}

// TestDiffNeverReportsAnUnreachableField checks the other direction: a name encoding/json
// refuses to serialize must never appear in a diff, or the engine would try to send a field
// that cannot exist on the wire.
func TestDiffNeverReportsAnUnreachableField(t *testing.T) {
	for _, shape := range jsonshapes.Shapes() {
		if len(shape.Unreachable) == 0 {
			continue
		}
		t.Run(shape.Name, func(t *testing.T) {
			zero := freshLike(shape.Value)
			changes, err := structdiff.GetStructDiff(zero, shape.Value, nil)
			require.NoError(t, err)

			var reported []string
			for _, change := range changes {
				if slices.Contains(shape.Unreachable, change.Path.String()) {
					reported = append(reported, change.Path.String())
				}
			}
			if shape.Gap("structdiff") {
				assert.NotEmpty(t, reported,
					"structdiff no longer reports an unreachable field -- remove the recorded gap")
				t.Logf("recorded gap: structdiff reports %v, which encoding/json does not serialize", reported)
				return
			}
			assert.Empty(t, reported,
				"structdiff reported %v, which encoding/json does not serialize", reported)
		})
	}
}

// cloneWith returns a copy of the shape's value with one field set, or nil when the shape
// records that structaccess cannot write that field yet.
func cloneWith(t *testing.T, shape jsonshapes.Shape, name, value string) any {
	t.Helper()

	clone := freshLike(shape.Value)
	if err := structaccess.SetByString(clone, name, value); err != nil {
		for _, gap := range shape.KnownSetGap {
			if gap == name {
				return nil
			}
		}
		t.Fatalf("cannot set %q: %v", name, err)
	}
	return clone
}

// freshLike returns a new zero value of the same type as v, a pointer to a struct.
func freshLike(v any) any {
	return reflect.New(reflect.TypeOf(v).Elem()).Interface()
}
