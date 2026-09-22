package protos

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A failed connection is the zero value of every bool, so omitempty would drop
// the fields entirely and make the failure indistinguishable from an unreported
// one. This pins the wire payload for that case.
func TestSshTunnelEventEncodesFailureExplicitly(t *testing.T) {
	b, err := json.Marshal(SshTunnelEvent{})
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(b, &got))

	for _, field := range []string{
		"is_success",
		"is_reconnect",
		"auto_start_cluster",
		"has_base_environment",
		"has_usage_policy",
		"keep_detached_requested",
	} {
		assert.Equal(t, false, got[field], "%s must be sent as false, not omitted", field)
	}
}

// The teardown event's whole purpose is counting how often detached work is destroyed, so a
// "nothing was left behind" teardown has to arrive as false rather than as an absent field.
func TestSshTunnelTeardownEventEncodesFalseExplicitly(t *testing.T) {
	b, err := json.Marshal(SshTunnelTeardownEvent{})
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(b, &got))

	for _, field := range []string{
		"keep_detached_requested",
		"had_detached_descendants_at_teardown",
	} {
		assert.Equal(t, false, got[field], "%s must be sent as false, not omitted", field)
	}
}

// Guards fields added later: a bool that can legitimately be false must not
// carry omitempty, or its false case arrives as NULL and cannot be counted.
func TestSshTunnelEventBoolFieldsOmitOmitempty(t *testing.T) {
	for _, typ := range []reflect.Type{
		reflect.TypeFor[SshTunnelEvent](),
		reflect.TypeFor[SshTunnelTeardownEvent](),
	} {
		for field := range typ.Fields() {
			if field.Type.Kind() != reflect.Bool {
				continue
			}
			tag := field.Tag.Get("json")
			assert.NotContains(t, tag, "omitempty",
				"%s.%s has omitempty; a false value would be indistinguishable from not reported", typ.Name(), field.Name)
		}
	}
}
