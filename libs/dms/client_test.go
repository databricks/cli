package dms

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientNamesEveryResourceTheSameWay(t *testing.T) {
	// One format each, so a call only ever passes ids.
	assert.Equal(t, "deployments/dep-1", DeploymentName("dep-1"))
	assert.Equal(t, "deployments/dep-1/versions/2", versionName("dep-1", 2))
}

func TestDeploymentIDFromName(t *testing.T) {
	id, err := deploymentIDFromName("deployments/abc-123")
	require.NoError(t, err)
	assert.Equal(t, "abc-123", id)

	_, err = deploymentIDFromName("abc-123")
	assert.Error(t, err)

	_, err = deploymentIDFromName("deployments/")
	assert.Error(t, err)
}

func TestUpdateRequestSendsExactlyTheMaskedFields(t *testing.T) {
	// The service rejects an update whose mask names a field the body leaves out, and treats a
	// field it does carry as written - so the body has to hold every masked field and nothing
	// else, empty values included. Both masks the CLI builds are asserted on the wire by
	// acceptance/bundle/dms; this pins the rule they both rely on.
	update := OperationUpdate{
		Fields:     DescribesResource,
		State:      json.RawMessage(`{"state":{"name":"foo"}}`),
		ResourceID: "job-1",
		Status:     bundledeployments.OperationStatusOperationStatusSucceeded,
	}

	// A successful write reports no error, and the empty value is what clears an earlier one.
	assert.Equal(t, map[string]any{
		"state":         `{"state":{"name":"foo"}}`,
		"resource_id":   "job-1",
		"error_message": "",
		"status":        bundledeployments.OperationStatusOperationStatusSucceeded,
		"sequence_id":   "3",
	}, newUpdateRequest(update, "3"))

	// A failure keeps the recorded state, so state is absent rather than empty: naming it
	// would clear what the resource last recorded.
	assert.Equal(t, map[string]any{
		"error_message": "boom",
		"status":        bundledeployments.OperationStatusOperationStatusFailed,
		"sequence_id":   "3",
	}, newUpdateRequest(NewFailureUpdate("job-1", nil, errors.New("boom")), "3"))

	// The deployment's own fields follow the same rule. Clearing deployment_mode - a target that
	// stops setting mode - sends it empty, which the SDK struct's omitempty would have dropped.
	deployment := Metadata{DisplayName: "b", TargetName: "t"}.deployment()
	assert.Equal(t, map[string]any{
		"target_name":     "t",
		"deployment_mode": bundledeployments.DeploymentMode(""),
	}, newDeploymentUpdate(deployment, "target_name,deployment_mode"))
}
