package aircmd

import (
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGrantJobPermissions(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)

	var requestBody string
	server.Handle("PATCH", "/api/2.0/permissions/jobs/123", func(req testserver.Request) any {
		requestBody = string(req.Body)
		return map[string]any{}
	})

	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	err = grantJobPermissions(t.Context(), w, "123", []permission{
		{UserName: new("alice@example.com"), Level: "CAN_MANAGE"},
		{GroupName: new("data-team"), Level: "CAN_VIEW"},
		{ServicePrincipalName: new("training-sp"), Level: "CAN_MANAGE_RUN"},
	})
	require.NoError(t, err)

	assert.JSONEq(t, `{
		"access_control_list": [
			{"user_name": "alice@example.com", "permission_level": "CAN_MANAGE"},
			{"group_name": "data-team", "permission_level": "CAN_VIEW"},
			{"service_principal_name": "training-sp", "permission_level": "CAN_MANAGE_RUN"}
		]
	}`, requestBody)
}
