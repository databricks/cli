package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

func init() {
	registerTestConfig("alerts", &resources.Alert{
		DisplayName: "my-alert",
		QueryText:   "SELECT 1",
		WarehouseId: "test-warehouse-id",
		Schedule: sql.CronSchedule{
			QuartzCronSchedule: "0 0 12 * * ?",
			TimezoneId:         "UTC",
		},
		Evaluation: sql.AlertV2Evaluation{
			ComparisonOperator: sql.ComparisonOperatorGreaterThan,
			Source: sql.AlertV2OperandColumn{
				Name: "column1",
			},
			Threshold: &sql.AlertV2Operand{
				Column: &sql.AlertV2OperandColumn{
					Name: "column2",
				},
			},
		},
	})

	registerTestDeps("alerts.permissions", func(ctx context.Context, client *databricks.WorkspaceClient) (any, error) {
		resp, err := client.AlertsV2.CreateAlert(ctx, sql.CreateAlertV2Request{
			Alert: sql.AlertV2{
				DisplayName: "alert-permissions",
				QueryText:   "SELECT 1",
				WarehouseId: "test-warehouse-id",
				Schedule: sql.CronSchedule{
					QuartzCronSchedule: "0 0 12 * * ?",
					TimezoneId:         "UTC",
				},
				Evaluation: sql.AlertV2Evaluation{
					ComparisonOperator: sql.ComparisonOperatorGreaterThan,
					Source: sql.AlertV2OperandColumn{
						Name: "column1",
					},
				},
			},
		})
		if err != nil {
			return nil, err
		}

		return &PermissionsState{
			ObjectID: "/alertsv2/" + resp.Id,
			EmbeddedSlice: []StatePermission{{
				Level:    "CAN_MANAGE",
				UserName: "user@example.com",
			}},
		}, nil
	})

	registerKnownMissingInStateType("alerts",
		"file_path",
	)

	registerCreateRecoveryOptOut("alerts", "the backend assigns the alert id, display names are not unique, and the create request takes no idempotency token")
}
