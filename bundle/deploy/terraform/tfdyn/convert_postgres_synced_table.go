package tfdyn

import (
	"context"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
)

type postgresSyncedTableConverter struct{}

func (c postgresSyncedTableConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	// The bundle config has flattened SyncedTableSyncedTableSpec fields at the top level.
	// Terraform expects them nested in a "spec" block.
	specFields := specFieldNames(schema.ResourcePostgresSyncedTableSpec{})
	topLevelFields := []string{"synced_table_id"}

	specMap := make(map[string]dyn.Value)
	for _, field := range specFields {
		if v := vin.Get(field); v.Kind() != dyn.KindInvalid {
			specMap[field] = v
		}
	}

	outMap := make(map[string]dyn.Value)
	for _, field := range topLevelFields {
		if v := vin.Get(field); v.Kind() != dyn.KindInvalid {
			outMap[field] = v
		}
	}
	if len(specMap) > 0 {
		outMap["spec"] = dyn.V(specMap)
	}

	vout := dyn.V(outMap)

	vout, diags := convert.Normalize(schema.ResourcePostgresSyncedTable{}, vout)
	for _, diag := range diags {
		log.Debugf(ctx, "postgres synced table normalization diagnostic: %s", diag.Summary)
	}

	vout, err := convertLifecycle(ctx, vout, vin.Get("lifecycle"))
	if err != nil {
		return err
	}

	out.PostgresSyncedTable[key] = vout.AsAny()

	return nil
}

func init() {
	registerConverter("postgres_synced_tables", postgresSyncedTableConverter{})
}
