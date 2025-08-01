package tfdyn

import (
	"context"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/database"
)

type databaseCatalogConverter struct{}

func (d databaseCatalogConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	// Normalize the output value to the target schema.
	vout, diags := convert.Normalize(database.DatabaseCatalog{}, vin)
	for _, diag := range diags {
		log.Debugf(ctx, "database Catalog normalization diagnostic: %s", diag.Summary)
	}
	out.DatabaseDatabaseCatalog[key] = vout.AsAny()

	return nil
}

func init() {
	registerConverter("database_catalogs", databaseCatalogConverter{})
}
