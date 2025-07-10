package tfdyn

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

func convertSqlWarehouseResource(ctx context.Context, vin dyn.Value) (dyn.Value, error) {
	vout, diags := convert.Normalize(sql.CreateWarehouseRequest{}, vin)
	for _, diag := range diags {
		log.Debugf(ctx, "sql warehouse normalization diagnostic: %s", diag.Summary)
	}

	return vout, nil
}

type sqlWarehouseConverter struct{}

func (sqlWarehouseConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	vout, err := convertSqlWarehouseResource(ctx, vin)
	if err != nil {
		return err
	}

	// Add the converted resource to the output.
	out.SqlEndpoint[key] = vout.AsAny()

	// Configure permissions for this resource.
	if permissions := convertPermissionsResource(ctx, vin); permissions != nil {
		permissions.SqlEndpointId = fmt.Sprintf("${databricks_sql_endpoint.%s.id}", key)
		out.Permissions["sql_endpoint_"+key] = permissions
	}

	return nil
}

func init() {
	registerConverter("sql_warehouses", sqlWarehouseConverter{})
}
