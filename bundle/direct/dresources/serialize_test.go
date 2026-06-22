package dresources

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/databricks/cli/libs/structs/structdiff"
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
