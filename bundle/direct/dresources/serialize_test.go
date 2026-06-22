package dresources

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/cli/libs/structs/structtag"
	"github.com/stretchr/testify/require"
)

// assertJSONRoundTrip marshals v to JSON, unmarshals it back into a fresh value
// of the same type, and asserts the two are equal field-by-field.
//
// This guards against wrapper structs (StateType, RemoteType) that embed an SDK
// type with its own MarshalJSON but forget to define their own: the embedded
// marshaler takes over and silently drops the wrapper's extra fields. The state
// file round-trips through exactly this path (dstate.SaveState json.Marshal ->
// apply.parseState json.Unmarshal), so a missing wrapper marshaler corrupts state.
func assertJSONRoundTrip(t *testing.T, v any, label string) {
	t.Helper()

	data, err := json.Marshal(v)
	require.NoError(t, err, "%s: Marshal failed", label)

	roundTripped := reflect.New(reflect.TypeOf(v)).Interface()
	err = json.Unmarshal(data, roundTripped)
	require.NoError(t, err, "%s: Unmarshal failed", label)
	back := reflect.ValueOf(roundTripped).Elem().Interface()

	// Diff the Go values rather than the JSON: a wrapper that drops fields keeps
	// them populated in v but loses them in back, so structdiff flags it even
	// though both marshal to the same (already-truncated) JSON. structdiff skips
	// ForceSendFields and json:"-" fields, which are intentionally not serialized.
	// Free-form any fields must be populated with []any/map[string]any (as JSON
	// decoding yields) so they round-trip to the same concrete type.
	changes, err := structdiff.GetStructDiff(v, back, nil)
	require.NoError(t, err)
	require.Empty(t, changes, "%s lost fields in JSON round-trip\nbefore: %s\nafter:  %s", label, jsonDump(v), jsonDump(back))
}

// TestStateTypeRoundTrip verifies that every resource's StateType survives a
// json.Marshal -> json.Unmarshal cycle without losing fields. The state file is
// persisted and reloaded through exactly this path.
func TestStateTypeRoundTrip(t *testing.T) {
	_, client := setupTestServerClient(t)

	for resourceType, resource := range SupportedResources {
		adapter, err := NewAdapter(resource, resourceType, client)
		require.NoError(t, err)

		t.Run(resourceType, func(t *testing.T) {
			inputConfig, ok := testConfig[resourceType]
			if !ok {
				// No populated fixture: fall back to a zero value. This still
				// exercises the marshalers, just without field-preservation signal.
				inputConfig = reflect.New(adapter.InputConfigType().Elem()).Interface()
			}

			newState, err := adapter.PrepareState(inputConfig)
			require.NoError(t, err, "PrepareState failed")

			assertJSONRoundTrip(t, newState, "StateType "+resourceType)
		})
	}
}

// TestRoundtripRemoteType verifies that every resource's RemoteType survives a
// json.Marshal -> json.Unmarshal cycle without losing fields. RemoteType is
// emitted in the plan's "remote_state" field, so a wrapper that embeds an SDK
// type with its own MarshalJSON must define its own or its extra fields vanish.
//
// Unlike the StateType check, we don't have a fixture per RemoteType, so we fill
// every field with a non-zero value via reflection. That way a dropped field is
// always non-zero before the round-trip and zero after, regardless of which
// fields a realistic value would populate.
func TestRoundtripRemoteType(t *testing.T) {
	for resourceType, resource := range SupportedResources {
		adapter, err := NewAdapter(resource, resourceType, nil)
		require.NoError(t, err)

		t.Run(resourceType, func(t *testing.T) {
			remote := reflect.New(adapter.RemoteType().Elem())
			fillNonZero(remote.Elem(), 0)
			assertJSONRoundTrip(t, remote.Interface(), "RemoteType "+resourceType)
		})
	}
}

// fillNonZero recursively populates v with non-zero values so that every
// serializable field is observable in a round-trip. It skips ForceSendFields
// (json:"-") and bounds recursion depth to avoid runaway on self-referential
// SDK types; bounding is safe because the fields at risk (a wrapper's own
// fields alongside an embedded SDK type) sit at the top level.
func fillNonZero(v reflect.Value, depth int) {
	if depth > 6 {
		return
	}
	switch v.Kind() {
	case reflect.Bool:
		v.SetBool(true)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v.SetInt(1)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v.SetUint(1)
	case reflect.Float32, reflect.Float64:
		v.SetFloat(1)
	case reflect.String:
		v.SetString("x")
	case reflect.Pointer:
		v.Set(reflect.New(v.Type().Elem()))
		fillNonZero(v.Elem(), depth+1)
	case reflect.Slice:
		elem := reflect.New(v.Type().Elem()).Elem()
		fillNonZero(elem, depth+1)
		v.Set(reflect.Append(v, elem))
	case reflect.Map:
		v.Set(reflect.MakeMap(v.Type()))
		val := reflect.New(v.Type().Elem()).Elem()
		fillNonZero(val, depth+1)
		v.SetMapIndex(reflect.ValueOf("k").Convert(v.Type().Key()), val)
	case reflect.Interface:
		// Free-form any fields decode to map[string]any from JSON.
		v.Set(reflect.ValueOf(map[string]any{"k": "v"}))
	case reflect.Struct:
		t := v.Type()
		for i := range t.NumField() {
			sf := t.Field(i)
			if !sf.IsExported() || sf.Name == "ForceSendFields" {
				continue
			}
			if structtag.JSONTag(sf.Tag.Get("json")).Name() == "-" {
				continue
			}
			fillNonZero(v.Field(i), depth+1)
		}
	default:
		// Kinds that don't appear in SDK state/remote types (chan, func,
		// complex, array, etc.) are left at their zero value.
	}
}
