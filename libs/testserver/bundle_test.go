package testserver

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stageOperation creates a deployment with one version that stages an operation for
// resourceKey, and returns the deployment and version ids.
func stageOperation(t *testing.T, s *FakeWorkspace, resourceKey string) (string, string) {
	t.Helper()

	parent := "/Users/" + TestUser.UserName
	resp := s.CreateDeployment(Request{Body: []byte(`{"initial_parent_path":"` + parent + `"}`)})
	require.Equal(t, 0, resp.StatusCode, resp.Body)
	var dep bundledeployments.Deployment
	remarshal(t, resp.Body, &dep)
	deploymentID := dep.Name[len("deployments/"):]

	body := `{"version_type":"VERSION_TYPE_DEPLOY","operations":[{"resource_key":"` + resourceKey + `","action_type":"OPERATION_ACTION_TYPE_UPDATE"}]}`
	resp = s.CreateVersion(Request{
		Body: []byte(body),
		URL:  &url.URL{RawQuery: "version_id=1"},
	}, deploymentID)
	require.Equal(t, 0, resp.StatusCode, resp.Body)

	return deploymentID, "1"
}

// updateOperation applies one update, and returns the sequence id for the next one.
func updateOperation(t *testing.T, s *FakeWorkspace, deploymentID, versionID, resourceKey, mask, body string) string {
	t.Helper()

	resp := s.UpdateOperation(Request{
		Body: []byte(body),
		URL:  &url.URL{RawQuery: "update_mask=" + url.QueryEscape(mask)},
	}, deploymentID, versionID, resourceKey)
	require.Equal(t, 0, resp.StatusCode, resp.Body)

	// The response types sequence_id as a string, which is why the SDK cannot read it.
	var raw map[string]any
	remarshal(t, resp.Body, &raw)
	return raw["sequence_id"].(string)
}

func listResources(t *testing.T, s *FakeWorkspace, deploymentID string) map[string]bundledeployments.Resource {
	t.Helper()

	resp := s.ListResources(deploymentID)
	require.Equal(t, 0, resp.StatusCode, resp.Body)

	var listed bundledeployments.ListResourcesResponse
	remarshal(t, resp.Body, &listed)

	out := map[string]bundledeployments.Resource{}
	for _, r := range listed.Resources {
		out[r.ResourceKey] = r
	}
	return out
}

func remarshal(t *testing.T, from, into any) {
	t.Helper()
	raw, err := json.Marshal(from)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, into))
}

// TestUpdateOperationProjectionFollowsTheMask pins the rule the CLI's update masks are
// built around: the deployment-level resource - what the next plan reads - moves only when
// the mask names state. An update that leaves state out reports an outcome and must not
// disturb what the deployment already holds.
func TestUpdateOperationProjectionFollowsTheMask(t *testing.T) {
	const key = "jobs.foo"
	const state = `{\"state\":{\"name\":\"foo\"}}`

	tests := []struct {
		name string
		// updates are applied in order, as (mask, body) pairs. The sequence id is filled in.
		updates [][2]string
		// wantResource is the state the deployment holds afterwards, empty for no resource.
		wantResource string
		wantID       string
	}{
		{
			name: "a write that names state records the resource",
			updates: [][2]string{
				{"state,error_message,resource_id,status", `{"state":"` + state + `","resource_id":"job-1","status":"OPERATION_STATUS_SUCCEEDED"}`},
			},
			wantResource: `{"state":{"name":"foo"}}`,
			wantID:       "job-1",
		},
		{
			name: "naming state with no value removes the resource",
			updates: [][2]string{
				{"state,error_message,resource_id,status", `{"state":"` + state + `","resource_id":"job-1","status":"OPERATION_STATUS_SUCCEEDED"}`},
				{"state,error_message,resource_id,status", `{"resource_id":"job-1","status":"OPERATION_STATUS_SUCCEEDED"}`},
			},
			wantResource: "",
		},
		{
			name: "a failure that leaves state out keeps the recorded resource",
			updates: [][2]string{
				{"state,error_message,resource_id,status", `{"state":"` + state + `","resource_id":"job-1","status":"OPERATION_STATUS_SUCCEEDED"}`},
				{"error_message,status", `{"error_message":"boom","status":"OPERATION_STATUS_FAILED"}`},
			},
			wantResource: `{"state":{"name":"foo"}}`,
			wantID:       "job-1",
		},
		{
			name: "a failure before any write records no resource",
			updates: [][2]string{
				{"error_message,status", `{"error_message":"boom","status":"OPERATION_STATUS_FAILED"}`},
			},
			wantResource: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewFakeWorkspace("http://localhost", "test-token")
			deploymentID, versionID := stageOperation(t, s, key)

			sequenceID := "0"
			for _, u := range tt.updates {
				mask, body := u[0], u[1]
				withSequence := body[:len(body)-1] + `,"sequence_id":"` + sequenceID + `"}`
				sequenceID = updateOperation(t, s, deploymentID, versionID, key, mask, withSequence)
			}

			resources := listResources(t, s, deploymentID)
			if tt.wantResource == "" {
				assert.NotContains(t, resources, key)
				return
			}
			require.Contains(t, resources, key)
			assert.JSONEq(t, tt.wantResource, resources[key].State)
			assert.Equal(t, tt.wantID, resources[key].ResourceId)
		})
	}
}
