package resources

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPostgresBranchConfigMarshalHonorsForceSendFields pins the reason
// PostgresBranchConfig.MarshalJSON is declared on a value receiver.
//
// NoExpiry is omitempty, so its zero value is emitted only because the embedded
// BranchSpec.ForceSendFields names it, and the SDK marshaler is what honors
// ForceSendFields. With a pointer-receiver MarshalJSON, only *T satisfies
// json.Marshaler, so marshalling a value falls back to the promoted
// BranchSpec.MarshalJSON, which drops the config's own fields.
func TestPostgresBranchConfigMarshalHonorsForceSendFields(t *testing.T) {
	c := PostgresBranchConfig{
		BranchId: "b1", Parent: "projects/p1",
		ForceSendFields: []string{"NoExpiry"},
	}

	b, err := json.Marshal(c)
	require.NoError(t, err)
	assert.JSONEq(t, `{"branch_id":"b1","parent":"projects/p1","no_expiry":false}`, string(b))
}

// An explicit false on the config's own *bool fields survives a JSON round trip
// (as the direct engine's state does). These fields are declared outside
// BranchSpec, so they must not end up in BranchSpec.ForceSendFields, which the
// SDK marshaler would reject.
func TestPostgresBranchConfigExplicitFalseRoundTrip(t *testing.T) {
	in := `{"branch_id":"b1","parent":"projects/p1","replace_existing":false,"purge_on_delete":false}`

	var c PostgresBranchConfig
	require.NoError(t, json.Unmarshal([]byte(in), &c))
	assert.Equal(t, new(false), c.ReplaceExisting)
	assert.Equal(t, new(false), c.PurgeOnDelete)

	out, err := json.Marshal(c)
	require.NoError(t, err)
	assert.JSONEq(t, in, string(out))
}

// TestPostgresBranchMarshalValueAndPointerAgree guards the invariant directly:
// a value-receiver MarshalJSON makes json.Marshal(x) and json.Marshal(&x)
// produce the same bytes. A pointer-only marshaler would send them down two
// different code paths.
func TestPostgresBranchMarshalValueAndPointerAgree(t *testing.T) {
	b := PostgresBranch{
		ID:       "the-id",
		BranchId: "b1",
		Parent:   "projects/p1",
	}

	byValue, err := json.Marshal(b)
	require.NoError(t, err)
	byPointer, err := json.Marshal(&b)
	require.NoError(t, err)
	assert.Equal(t, string(byPointer), string(byValue))
	assert.JSONEq(t, `{"id":"the-id","lifecycle":{},"branch_id":"b1","parent":"projects/p1"}`, string(byValue))
}
