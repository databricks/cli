package tfdyn

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertAlert(t *testing.T) {
	src := resources.Alert{
		AlertV2: sql.AlertV2{
			DisplayName:       "test_alert",
			QueryText:         "SELECT 1",
			WarehouseId:       "test_warehouse_id",
			CustomSummary:     "Test alert summary",
			CustomDescription: "Test alert description",
		},
		Permissions: []resources.AlertPermission{
			{
				Level:    "CAN_VIEW",
				UserName: "jane@doe.com",
			},
			{
				Level:                "CAN_MANAGE",
				ServicePrincipalName: "sp-test",
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = alertConverter{}.Convert(ctx, "test_alert", vin, out)
	require.NoError(t, err)

	// Assert equality on the alert
	alert := out.Alert["test_alert"]
	assert.Equal(t, map[string]any{
		"display_name":       "test_alert",
		"query_text":         "SELECT 1",
		"warehouse_id":       "test_warehouse_id",
		"custom_summary":     "Test alert summary",
		"custom_description": "Test alert description",
	}, alert)

	// Assert equality on the permissions
	assert.Equal(t, &schema.ResourcePermissions{
		AlertV2Id: "${databricks_alert_v2.test_alert.id}",
		AccessControl: []schema.ResourcePermissionsAccessControl{
			{
				UserName:        "jane@doe.com",
				PermissionLevel: "CAN_VIEW",
			},
			{
				ServicePrincipalName: "sp-test",
				PermissionLevel:      "CAN_MANAGE",
			},
		},
	}, out.Permissions["alert_test_alert"])
}
