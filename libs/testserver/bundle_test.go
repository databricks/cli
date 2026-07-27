package testserver

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDeploymentBodyKeepsTypedFieldsAndLastSuccessfulVersionID guards against
// serializing the deployment through a struct that embeds
// bundledeployments.Deployment: Deployment has its own MarshalJSON, which is
// promoted to the embedding struct and silently drops last_successful_version_id.
// The CLI read path treats a missing value as "DMS does not own the state", so
// losing the field here makes the whole overlay path untestable.
func TestDeploymentBodyKeepsTypedFieldsAndLastSuccessfulVersionID(t *testing.T) {
	d := &dmsDeployment{lastSuccessfulVersionID: "2"}
	d.deployment.Name = "deployments/abc"
	d.deployment.LastVersionId = "3"
	d.deployment.TargetName = "default"

	body, err := deploymentBody(d)
	require.NoError(t, err)

	assert.Equal(t, "deployments/abc", body["name"])
	assert.Equal(t, "3", body["last_version_id"])
	assert.Equal(t, "default", body["target_name"])
	assert.Equal(t, "2", body["last_successful_version_id"])

	// The response must round-trip as JSON the same way, since that is what the
	// client actually reads.
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	assert.JSONEq(t,
		`{"name":"deployments/abc","last_version_id":"3","target_name":"default","last_successful_version_id":"2"}`,
		string(raw))
}

// TestDeploymentBodyOmitsUnsetLastSuccessfulVersionID checks that a deployment
// with no successful version does not advertise one: the read path must keep
// using the local state file in that case.
func TestDeploymentBodyOmitsUnsetLastSuccessfulVersionID(t *testing.T) {
	d := &dmsDeployment{}
	d.deployment.Name = "deployments/abc"

	body, err := deploymentBody(d)
	require.NoError(t, err)

	assert.NotContains(t, body, "last_successful_version_id")
}
