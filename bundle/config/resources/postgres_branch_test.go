package resources

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
