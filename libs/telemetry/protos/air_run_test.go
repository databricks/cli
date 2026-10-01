package protos

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAirRunEventOptionalMeasurements(t *testing.T) {
	for _, size := range []*int64{nil, new(int64(0)), new(int64(123))} {
		raw, err := json.Marshal(AirRunEvent{
			CodeSourceSizeBytes:           size,
			CodeSourcePackagingDurationMs: size,
			CodeSourceUploadDurationMs:    size,
		})
		require.NoError(t, err)
		var fields map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(raw, &fields))
		for _, name := range []string{"code_source_size_bytes", "code_source_packaging_duration_ms", "code_source_upload_duration_ms"} {
			if size == nil {
				assert.NotContains(t, fields, name)
			} else {
				var got int64
				require.NoError(t, json.Unmarshal(fields[name], &got))
				assert.Equal(t, *size, got)
			}
		}
		assert.Equal(t, "0", string(fields["num_gpus"]))
		assert.Equal(t, "0", string(fields["max_retries"]))
		assert.Equal(t, "false", string(fields["submitted_successfully"]))
	}
}
