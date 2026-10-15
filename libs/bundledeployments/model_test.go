package bundledeployments

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOperationUnmarshalSequenceId(t *testing.T) {
	for input, want := range map[string]int64{
		`{"sequence_id":"5","resource_key":"a"}`: 5,
		`{"sequence_id":7}`:                      7,
		`{}`:                                     0,
	} {
		var o Operation
		require.NoError(t, json.Unmarshal([]byte(input), &o))
		assert.Equal(t, want, o.SequenceId, input)
	}
}
