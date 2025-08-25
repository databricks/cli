package tfdyn

import (
	"context"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/database"
)

type syncedDatabaseTableConverter struct{}

func (s syncedDatabaseTableConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	// Normalize the output value to the target schema.
	vout, diags := convert.Normalize(database.SyncedDatabaseTable{}, vin)
	for _, diag := range diags {
		log.Debugf(ctx, "synced database table normalization diagnostic: %s", diag.Summary)
	}
	out.DatabaseSyncedDatabaseTable[key] = vout.AsAny()

	return nil
}

func init() {
	registerConverter("synced_database_tables", syncedDatabaseTableConverter{})
}
