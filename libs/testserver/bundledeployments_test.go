package testserver

import (
	"fmt"
	"strings"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testVersion = "deployments/1/versions/1"

// workspaceWithStagedCreates holds deployment 1 at version 1 with a staged create per key.
func workspaceWithStagedCreates(keys ...string) *FakeWorkspace {
	workspace := NewFakeWorkspace("http://test.test", "token")
	d := &DmsDeployment{
		Resources:  map[string]bundledeployments.Resource{},
		Operations: map[string]*bundledeployments.Operation{},
	}
	for _, key := range keys {
		name := testVersion + "/operations/" + key
		d.Operations[name] = &bundledeployments.Operation{
			Name:        name,
			ResourceKey: key,
			ActionType:  bundledeployments.OperationActionTypeOperationActionTypeCreate,
			Status:      operationStatusPending,
		}
	}
	workspace.DmsDeployments["1"] = d
	return workspace
}

// createdEntry records a successful create of key at sequence.
func createdEntry(key string, sequence int) string {
	return fmt.Sprintf(`{"operation": {"name": %q, "sequence_id": %d, "state": "{}", "resource_id": "id-%s", "error_message": "", "status": "OPERATION_STATUS_SUCCEEDED"}, "update_mask": "state,error_message,resource_id,status"}`,
		testVersion+"/operations/"+key, sequence, key)
}

func batchUpdate(workspace *FakeWorkspace, entries ...string) Response {
	return workspace.BatchUpdateOperations(Request{
		Body: []byte(`{"requests": [` + strings.Join(entries, ",") + `]}`),
	}, "1", "1")
}

func TestBatchUpdateOperationsAppliesEveryUpdateInOrder(t *testing.T) {
	workspace := workspaceWithStagedCreates("jobs.a", "jobs.b")

	response := batchUpdate(workspace, createdEntry("jobs.b", 0), createdEntry("jobs.a", 0))

	require.Equal(t, 0, response.StatusCode)
	operations := response.Body.(map[string]any)["operations"].([]map[string]any)
	require.Len(t, operations, 2)
	assert.Equal(t, testVersion+"/operations/jobs.b", operations[0]["name"])
	assert.Equal(t, testVersion+"/operations/jobs.a", operations[1]["name"])
	assert.Equal(t, "1", operations[0]["sequence_id"])
	d := workspace.DmsDeployments["1"]
	assert.Equal(t, "id-jobs.a", d.Resources["jobs.a"].ResourceId)
	assert.Equal(t, "id-jobs.b", d.Resources["jobs.b"].ResourceId)
}

func TestBatchUpdateOperationsRejectsTheWholeBatch(t *testing.T) {
	workspace := workspaceWithStagedCreates("jobs.a", "jobs.b")

	// jobs.b is still at sequence 0, so the second entry is stale.
	response := batchUpdate(workspace, createdEntry("jobs.a", 0), createdEntry("jobs.b", 1))

	assert.Equal(t, 409, response.StatusCode)
	d := workspace.DmsDeployments["1"]
	assert.Equal(t, int64(0), d.Operations[testVersion+"/operations/jobs.a"].SequenceId)
	assert.Empty(t, d.Resources)
}

func TestBatchUpdateOperationsRejectsMalformedBatches(t *testing.T) {
	tooMany := make([]string, maxBatchOperationUpdates+1)
	for i := range tooMany {
		tooMany[i] = createdEntry(fmt.Sprintf("jobs.j%d", i), 0)
	}
	otherVersion := strings.Replace(createdEntry("jobs.a", 0), testVersion, "deployments/1/versions/2", 1)

	for name, entries := range map[string][]string{
		"empty":         nil,
		"too many":      tooMany,
		"repeated":      {createdEntry("jobs.a", 0), createdEntry("jobs.a", 0)},
		"other version": {otherVersion},
	} {
		t.Run(name, func(t *testing.T) {
			workspace := workspaceWithStagedCreates("jobs.a")

			response := batchUpdate(workspace, entries...)

			assert.Equal(t, 400, response.StatusCode)
			assert.Empty(t, workspace.DmsDeployments["1"].Resources)
		})
	}
}
