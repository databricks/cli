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
// PurgeOnDelete is omitempty, so its zero value is emitted only because
// ForceSendFields names it, and the SDK marshaler is what honors ForceSendFields.
// With a pointer-receiver MarshalJSON, only *T satisfies json.Marshaler, so
// marshalling a value falls back to plain encoding/json -- which ignores
// ForceSendFields (tagged json:"-") and would drop the field.
func TestPostgresBranchConfigMarshalHonorsForceSendFields(t *testing.T) {
	c := PostgresBranchConfig{BranchId: "b1", Parent: "projects/p1"}
	c.ForceSendFields = []string{"PurgeOnDelete"}

	b, err := json.Marshal(c)
	require.NoError(t, err)
	assert.JSONEq(t, `{"branch_id":"b1","parent":"projects/p1","purge_on_delete":false}`, string(b))
}

// TestPostgresBranchMarshalValueAndPointerAgree guards the invariant directly:
// a value-receiver MarshalJSON makes json.Marshal(x) and json.Marshal(&x)
// produce the same bytes. A pointer-only marshaler would send them down two
// different code paths.
func TestPostgresBranchMarshalValueAndPointerAgree(t *testing.T) {
	b := PostgresBranch{}
	b.ID = "the-id"
	b.BranchId = "b1"
	b.Parent = "projects/p1"

	byValue, err := json.Marshal(b)
	require.NoError(t, err)
	byPointer, err := json.Marshal(&b)
	require.NoError(t, err)
	assert.Equal(t, string(byPointer), string(byValue))
	assert.JSONEq(t, `{"id":"the-id","lifecycle":{},"branch_id":"b1","parent":"projects/p1"}`, string(byValue))
}
