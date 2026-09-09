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
	// field it does carry as written - so the body has to hold every masked field, empty values
	// included, which ForceSendFields is what secures. Both bodies the CLI builds are asserted on
	// the wire by acceptance/bundle/dms; this pins the rule they both rely on.
	update := OperationUpdate{
		Fields:     DescribesResource,
		State:      json.RawMessage(`{"state":{"name":"foo"}}`),
		ResourceID: "job-1",
		Status:     bundledeployments.OperationStatusOperationStatusSucceeded,
	}

	// A successful write reports no error, and the empty value is what clears an earlier one.
	operation, err := newOperationUpdate(update, "3")
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"sequence_id": 3,
		"state": "{\"state\":{\"name\":\"foo\"}}",
		"resource_id": "job-1",
		"error_message": "",
		"status": "OPERATION_STATUS_SUCCEEDED"
	}`, marshalBody(t, operation))

	// A failure keeps the recorded state, so state is absent rather than empty: naming it
	// would clear what the resource last recorded.
	operation, err = newOperationUpdate(NewFailureUpdate("job-1", nil, errors.New("boom")), "3")
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"sequence_id": 3,
		"error_message": "boom",
		"status": "OPERATION_STATUS_FAILED"
	}`, marshalBody(t, operation))

	// The deployment's own fields follow the same rule. Clearing deployment_mode - a target that
	// stops setting mode - sends it empty, which the SDK struct's omitempty would have dropped.
	deployment := newDeploymentUpdate(Metadata{DisplayName: "b", TargetName: "t"}, "target_name,deployment_mode")
	assert.JSONEq(t, `{
		"target_name": "t",
		"deployment_mode": ""
	}`, marshalBody(t, deployment))
}

// marshalBody renders the request body the SDK sends for v, so a test sees the wire the
// ForceSendFields produce.
func marshalBody(t *testing.T, v any) string {
	t.Helper()
	body, err := json.Marshal(v)
	require.NoError(t, err)
	return string(body)
}
