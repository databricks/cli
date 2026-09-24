package dresources

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/aisearch"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVectorSearchEndpointUsagePolicyUpdateWire verifies the AI Search path, body, and exact mask.
func TestVectorSearchEndpointUsagePolicyUpdateWire(t *testing.T) {
	for _, tc := range []struct {
		name string
		old  string
		new  string
	}{
		{name: "set", old: "old-policy", new: "new-policy"},
		{name: "clear", old: "old-policy", new: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := testserver.New(t)
			server.Handle("PATCH", "/api/2.0/ai-search/{name...}", func(req testserver.Request) any {
				assert.Equal(t, "/api/2.0/ai-search/workspaces/123456/endpoints/my-endpoint", req.URL.Path)
				assert.Equal(t, "usage_policy_id", req.URL.Query().Get("update_mask"))

				var body map[string]any
				require.NoError(t, json.Unmarshal(req.Body, &body))
				assert.Equal(t, "workspaces/123456/endpoints/my-endpoint", body["name"])
				assert.Equal(t, tc.new, body["usage_policy_id"])
				return aisearch.Endpoint{Name: "workspaces/123456/endpoints/my-endpoint", UsagePolicyId: tc.new}
			})

			client, err := databricks.NewWorkspaceClient(&databricks.Config{
				Host:        server.URL,
				Token:       "testtoken",
				WorkspaceID: "123456",
			})
			require.NoError(t, err)

			_, err = (&ResourceVectorSearchEndpoint{}).New(client).DoUpdate(
				t.Context(),
				"my-endpoint",
				&vectorsearch.CreateEndpoint{UsagePolicyId: tc.new},
				&deployplan.PlanEntry{Changes: deployplan.Changes{
					"usage_policy_id": {Action: deployplan.Update, Old: tc.old, New: tc.new},
				}},
			)
			require.NoError(t, err)
		})
	}
}
