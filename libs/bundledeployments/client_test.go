package bundledeployments

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/databricks/cli/libs/apiclient/apiclienttest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateDeploymentSendsMaskAndBody(t *testing.T) {
	c := New(apiclienttest.New(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/2.0/bundle/deployments/d1", r.URL.Path)
		assert.Equal(t, "display_name,mode", r.URL.Query().Get("update_mask"))
		body, _ := io.ReadAll(r.Body)
		assert.JSONEq(t, `{"display_name":"x"}`, string(body))
		_, _ = w.Write([]byte(`{"name":"deployments/d1","display_name":"x"}`))
	}))
	d, err := c.UpdateDeployment(t.Context(), UpdateDeploymentRequest{
		Name:       "deployments/d1",
		Deployment: Deployment{DisplayName: "x"},
		UpdateMask: FieldMask{Paths: []string{"display_name", "mode"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "deployments/d1", d.Name)
}

func TestListResourcesAllPaginates(t *testing.T) {
	c := New(apiclienttest.New(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/2.0/bundle/deployments/d1/resources", r.URL.Path)
		resp := ListResourcesResponse{Resources: []Resource{{Name: "r1"}}, NextPageToken: "p2"}
		if r.URL.Query().Get("page_token") == "p2" {
			resp = ListResourcesResponse{Resources: []Resource{{Name: "r2"}}}
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	res, err := c.ListResourcesAll(t.Context(), ListResourcesRequest{Parent: "deployments/d1"})
	require.NoError(t, err)
	require.Len(t, res, 2)
	assert.Equal(t, "r2", res[1].Name)
}
