package dresources

import (
	"encoding/json"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMlflowModelRemoteJSONRoundTrip ensures the wrapper-only ModelId field survives
// state serialization. The embedded ml.ModelDatabricks defines its own Marshal/Unmarshal
// which would otherwise be promoted to the wrapper and drop ModelId.
func TestMlflowModelRemoteJSONRoundTrip(t *testing.T) {
	original := &MlflowModelRemote{
		ModelDatabricks: ml.ModelDatabricks{
			Name:        "my_model",
			Description: "my_description",
			Id:          "internal_id",
		},
		ModelId: "model_id_123",
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, "model_id_123", got["model_id"], "model_id must be present in serialized JSON")

	var loaded MlflowModelRemote
	require.NoError(t, json.Unmarshal(data, &loaded))
	assert.Equal(t, "model_id_123", loaded.ModelId, "ModelId must be preserved across marshal/unmarshal")
	assert.Equal(t, "my_model", loaded.Name)
	assert.Equal(t, "my_description", loaded.Description)
}
