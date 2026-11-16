package serving_endpoints

import (
	"encoding/json"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDroppedFields(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{
			name: "nothing dropped",
			raw:  `{"predictions": [1, 2], "id": "x"}`,
		},
		{
			name: "chat fields the typed response doesn't have",
			raw:  `{"choices": [{"index": 0, "finish_reason": "stop", "message": {"role": "assistant", "content": "hi", "tool_calls": [{"id": "1"}]}}], "service_tier": "default"}`,
			want: []string{"choices[].finish_reason", "choices[].message.tool_calls", "service_tier"},
		},
		{
			name: "content parts become a string",
			raw:  `{"choices": [{"message": {"role": "assistant", "content": [{"type": "text", "text": "hi"}]}}]}`,
			want: []string{"choices[].message.content"},
		},
		{
			name: "null and zero values are not reported",
			raw:  `{"id": null, "created": 0, "predictions": [1], "custom_outputs": {}}`,
		},
		{
			name: "agent output",
			raw:  `{"output": [{"type": "message"}], "custom_outputs": {"k": 1}}`,
			want: []string{"custom_outputs", "output"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var typed serving.QueryEndpointResponse
			require.NoError(t, json.Unmarshal([]byte(tt.raw), &typed))
			assert.Equal(t, tt.want, droppedFields([]byte(tt.raw), &typed))
		})
	}
}

func TestDroppedFieldsInvalidJSON(t *testing.T) {
	assert.Nil(t, droppedFields([]byte(`{"predictions": [`), &serving.QueryEndpointResponse{}))
}
