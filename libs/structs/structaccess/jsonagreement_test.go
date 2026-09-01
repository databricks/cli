package structaccess_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/databricks/cli/libs/structs/internal/jsonshapes"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAgreesWithEncodingJSON checks structaccess against encoding/json over the shape
// corpus: a name the wire format carries must be readable, writable and valid, and a name
// it does not carry must be none of those.
//
// The write assertion goes through json.Marshal rather than the Go field, so it fails if
// Set stores into a field the wire format ignores -- which is the failure mode a shadowed
// or ambiguous embed produces, and the one a Go-field assertion cannot see.
func TestAgreesWithEncodingJSON(t *testing.T) {
	for _, shape := range jsonshapes.Shapes() {
		t.Run(shape.Name, func(t *testing.T) {
			for _, name := range shape.JSONFields {
				if slices.Contains(shape.KnownSetGap, name) {
					// Read side still has to agree; the write side is a recorded gap, asserted as
					// failing so that fixing it forces the entry out of the corpus.
					require.NoError(t, structaccess.ValidateByString(reflect.TypeOf(shape.Value), name))
					_, err := structaccess.GetByString(shape.Value, name)
					require.NoError(t, err)
					require.Error(t, structaccess.SetByString(freshLike(shape.Value), name, "written"),
						"KnownSetGap %q now works -- remove it from the corpus", name)
					continue
				}

				require.NoError(t, structaccess.ValidateByString(reflect.TypeOf(shape.Value), name),
					"%q is on the wire, so the type must validate it", name)

				_, err := structaccess.GetByString(shape.Value, name)
				require.NoError(t, err, "%q is on the wire, so Get must resolve it", name)

				fresh := freshLike(shape.Value)
				require.NoError(t, structaccess.SetByString(fresh, name, "written"),
					"%q is on the wire, so Set must reach it", name)

				leaves, err := jsonshapes.Leaves(fresh)
				require.NoError(t, err)
				assert.Equal(t, "written", leaves[name],
					"Set wrote somewhere encoding/json does not serialize: %v", leaves)
			}

			for _, name := range shape.Unreachable {
				assert.Error(t, structaccess.ValidateByString(reflect.TypeOf(shape.Value), name),
					"%q never reaches the wire, so the type must not validate it", name)

				_, err := structaccess.GetByString(shape.Value, name)
				assert.Error(t, err, "%q never reaches the wire, so Get must not resolve it", name)

				assert.Error(t, structaccess.SetByString(freshLike(shape.Value), name, "written"),
					"%q never reaches the wire, so Set must not claim to write it", name)
			}
		})
	}
}

// TestValidateIsAPropertyOfTheType checks that whether a path validates does not depend on
// the value: an embedded pointer left nil declares the same fields as one that is set.
func TestValidateIsAPropertyOfTheType(t *testing.T) {
	nilEmbed := reflect.TypeFor[*jsonshapes.NilPointerEmbed]() //exhaustruct:ignore
	setEmbed := reflect.TypeFor[*jsonshapes.SetPointerEmbed]() //exhaustruct:ignore

	for _, name := range []string{"value", "own"} {
		assert.NoError(t, structaccess.ValidateByString(nilEmbed, name))
		assert.NoError(t, structaccess.ValidateByString(setEmbed, name))
	}
}

// freshLike returns a new zero value of the same type as v, which is a pointer to a struct.
func freshLike(v any) any {
	return reflect.New(reflect.TypeOf(v).Elem()).Interface()
}
