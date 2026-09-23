package dresources

import (
	"encoding/json"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeLibraries(t *testing.T) {
	six := compute.Library{Pypi: &compute.PythonPyPiLibrary{Package: "six"}}
	urllib3 := compute.Library{Pypi: &compute.PythonPyPiLibrary{Package: "urllib3"}}

	// generic re-encodes v as map[string]any / []any, the shape a plan read back from disk arrives in.
	generic := func(v any) any {
		b, err := json.Marshal(v)
		require.NoError(t, err)
		var out any
		require.NoError(t, json.Unmarshal(b, &out))
		return out
	}

	tests := []struct {
		name    string
		value   any
		want    []compute.Library
		wantErr bool
	}{
		{name: "single typed", value: six, want: []compute.Library{six}},
		{name: "slice typed", value: []compute.Library{six, urllib3}, want: []compute.Library{six, urllib3}},
		{name: "single generic JSON", value: generic(six), want: []compute.Library{six}},
		{name: "slice generic JSON", value: generic([]compute.Library{six, urllib3}), want: []compute.Library{six, urllib3}},
		{name: "not a library", value: "not a library", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeLibraries(tt.value)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			// Compare JSON: unmarshalling populates the SDK's ForceSendFields bookkeeping, which the
			// literals in want do not set, but the library content is what matters.
			wantJSON, err := json.Marshal(tt.want)
			require.NoError(t, err)
			gotJSON, err := json.Marshal(got)
			require.NoError(t, err)
			assert.JSONEq(t, string(wantJSON), string(gotJSON))
		})
	}
}
