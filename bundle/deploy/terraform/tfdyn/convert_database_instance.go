package tfdyn

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/database"
)

type databaseInstanceConverter struct{}

func (d databaseInstanceConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	// Normalize the output value to the target schema.
	vout, diags := convert.Normalize(database.DatabaseInstance{}, vin)
	for _, diag := range diags {
		log.Debugf(ctx, "database instance normalization diagnostic: %s", diag.Summary)
	}
	out.DatabaseInstance[key] = vout.AsAny()

	// Configure permissions for this resource.
	if permissions := convertPermissionsResource(ctx, vin); permissions != nil {
		permissions.DatabaseInstanceName = fmt.Sprintf("${databricks_database_instance.%s.name}", key)
		out.Permissions["database_instance_"+key] = permissions
	}

	return nil
}

func init() {
	registerConverter("database_instances", databaseInstanceConverter{})
}
