package structtag_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/databricks/cli/libs/structs/structtag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJSONTagNameMatchesEncodingJSON checks structtag's reading of a json tag against what
// encoding/json does with the same tag. Every other package asks structtag for a field's
// name, so a tag it reads differently renames or hides the field for all of them at once.
//
// The oracle is a real marshal: the emitted object key is, by definition, the name
// encoding/json chose.
func TestJSONTagNameMatchesEncodingJSON(t *testing.T) {
	tests := []struct {
		tag string
		// key is the object key encoding/json emits, or "" when it omits the field.
		key string
	}{
		{tag: `json:"name"`, key: "name"},
		{tag: `json:"name,omitempty"`, key: "name"},
		{tag: `json:"name,string"`, key: "name"},
		{tag: `json:"-"`, key: ""},
		// A lone dash means "skip"; a dash with a comma means a field literally named "-".
		{tag: `json:"-,"`, key: "-"},
		{tag: `json:",omitempty"`, key: "Field"},
		{tag: `json:""`, key: "Field"},
	}

	for _, tc := range tests {
		t.Run(tc.tag, func(t *testing.T) {
			typ := reflect.StructOf([]reflect.StructField{{
				Name: "Field",
				Type: reflect.TypeOf(""),
				Tag:  reflect.StructTag(tc.tag),
			}})
			value := reflect.New(typ)
			value.Elem().Field(0).SetString("v")

			blob, err := json.Marshal(value.Interface())
			require.NoError(t, err)
			var emitted map[string]any
			require.NoError(t, json.Unmarshal(blob, &emitted))

			if tc.key == "" {
				require.Empty(t, emitted, "expected the field to be skipped, got %s", blob)
			} else {
				require.Contains(t, emitted, tc.key, "encoding/json emitted %s", blob)
			}

			// What structtag reports has to lead every package to the same conclusion: the
			// emitted key, or "-" for a field encoding/json skips.
			name := structtag.JSONTag(typ.Field(0).Tag.Get("json")).Name()
			switch {
			case tc.key == "":
				assert.Equal(t, "-", name, "a skipped field must read as %q", "-")
			case name == "":
				// An empty tag name means "fall back to the Go field name", which is what
				// encoding/json did.
				assert.Equal(t, "Field", tc.key)
			default:
				assert.Equal(t, tc.key, name)
			}
		})
	}
}

// TestOmitEmptyMatchesEncodingJSON checks the other half of the tag: whether a zero value is
// dropped. structaccess decides ForceSendFields from this, so reading it wrongly means a
// field is sent when it should be absent, or absent when it should be sent.
func TestOmitEmptyMatchesEncodingJSON(t *testing.T) {
	for _, tc := range []struct {
		tag       string
		omitEmpty bool
	}{
		{tag: `json:"name"`, omitEmpty: false},
		{tag: `json:"name,omitempty"`, omitEmpty: true},
		{tag: `json:",omitempty"`, omitEmpty: true},
		{tag: `json:"name,string,omitempty"`, omitEmpty: true},
	} {
		t.Run(tc.tag, func(t *testing.T) {
			typ := reflect.StructOf([]reflect.StructField{{
				Name: "Field",
				Type: reflect.TypeOf(""),
				Tag:  reflect.StructTag(tc.tag),
			}})

			blob, err := json.Marshal(reflect.New(typ).Interface())
			require.NoError(t, err)

			dropped := string(blob) == "{}"
			assert.Equal(t, tc.omitEmpty, dropped,
				"encoding/json emitted %s for a zero value", blob)
			assert.Equal(t, tc.omitEmpty, structtag.JSONTag(typ.Field(0).Tag.Get("json")).OmitEmpty())
		})
	}
}
