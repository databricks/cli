package dresources

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/structs/structtag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestForceSendZeroValues verifies that every StateType/RemoteType honors the SDK
// ForceSendFields convention: a zero-valued scalar field whose name is in
// ForceSendFields must still be serialized.
//
// The round-trip tests in serialize_test.go fill every field non-zero and leave
// ForceSendFields empty, so they cover the "populated field silently dropped"
// failure mode but never the opposite one: a force-sent zero value that is omitted
// because the type marshals via plain encoding/json (which knows nothing about
// ForceSendFields) rather than the SDK marshaler. That path is the entire reason
// ForceSendFields and the wrapper marshalers exist, so it is the one worth pinning.
func TestForceSendZeroValues(t *testing.T) {
	for _, dim := range []struct {
		label  string
		typeOf func(*Adapter) reflect.Type
	}{
		{"StateType", (*Adapter).StateType},
		{"RemoteType", (*Adapter).RemoteType},
	} {
		for resourceType, resource := range SupportedResources {
			adapter, err := NewAdapter(resource, resourceType, nil)
			require.NoError(t, err)
			t.Run(dim.label+"/"+resourceType, func(t *testing.T) {
				assertForceSendHonored(t, dim.typeOf(adapter))
			})
		}
	}
}

// assertForceSendHonored builds a zero value of typ (a pointer to struct), tells the
// struct that owns ForceSendFields to force-send each of its own scalar omitempty
// fields, marshals, and asserts each such field survives into the JSON. ForceSendFields
// governs the fields declared alongside it (embedded structs carry their own), so the
// forced fields and the ForceSendFields must come from the same struct.
func assertForceSendHonored(t *testing.T, typ reflect.Type) {
	t.Helper()

	sf, ok := typ.Elem().FieldByName("ForceSendFields")
	if !ok || sf.Type != reflect.TypeFor[[]string]() {
		t.Skip("no ForceSendFields")
	}

	ptr := reflect.New(typ.Elem())
	owner := fieldByIndexAlloc(ptr.Elem(), sf.Index[:len(sf.Index)-1])

	var goNames, jsonNames []string
	for field := range owner.Type().Fields() {
		if !field.IsExported() || field.Anonymous || !isBasicKind(field.Type.Kind()) {
			continue
		}
		tag := structtag.JSONTag(field.Tag.Get("json"))
		if name := tag.Name(); name != "" && name != "-" && strings.Contains(field.Tag.Get("json"), "omitempty") {
			goNames = append(goNames, field.Name)
			jsonNames = append(jsonNames, name)
		}
	}
	if len(goNames) == 0 {
		t.Skip("no force-sendable scalar fields")
	}

	owner.Field(sf.Index[len(sf.Index)-1]).Set(reflect.ValueOf(goNames))

	data, err := json.Marshal(ptr.Interface())
	require.NoError(t, err)

	keys := jsonKeys(data)
	for _, name := range jsonNames {
		assert.Contains(t, keys, name,
			"field %q is force-sent (zero value) but was dropped from the JSON: the type marshals via plain encoding/json instead of a ForceSendFields-aware marshaler", name)
	}
}

func isBasicKind(k reflect.Kind) bool {
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

// fieldByIndexAlloc walks index from v, allocating a nil pointer before descending
// through it, so a field reached through an unset pointer is still addressable.
func fieldByIndexAlloc(v reflect.Value, index []int) reflect.Value {
	for _, i := range index {
		if v.Kind() == reflect.Pointer {
			if v.IsNil() {
				v.Set(reflect.New(v.Type().Elem()))
			}
			v = v.Elem()
		}
		v = v.Field(i)
	}
	return v
}

// jsonKeys returns every object key present anywhere in the JSON document, so a
// forced field is found regardless of the nesting level it serializes at.
func jsonKeys(data []byte) map[string]bool {
	var root any
	if err := json.Unmarshal(data, &root); err != nil {
		return nil
	}
	keys := map[string]bool{}
	var walk func(any)
	walk = func(x any) {
		switch t := x.(type) {
		case map[string]any:
			for k, v := range t {
				keys[k] = true
				walk(v)
			}
		case []any:
			for _, e := range t {
				walk(e)
			}
		}
	}
	walk(root)
	return keys
}
