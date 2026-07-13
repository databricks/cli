package tfdyn

import (
	"context"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
)

type postgresCatalogConverter struct{}

func (c postgresCatalogConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	// The bundle config has flattened CatalogCatalogSpec fields at the top level.
	// Terraform expects them nested in a "spec" block.
	specFields := specFieldNames(schema.ResourcePostgresCatalogSpec{})
	topLevelFields := []string{"catalog_id"}

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

	vout, diags := convert.Normalize(schema.ResourcePostgresCatalog{}, vout)
	for _, diag := range diags {
		log.Debugf(ctx, "postgres catalog normalization diagnostic: %s", diag.Summary)
	}

	vout, err := convertLifecycle(ctx, vout, vin.Get("lifecycle"))
	if err != nil {
		return err
	}

	out.PostgresCatalog[key] = vout.AsAny()

	return nil
}

func init() {
	registerConverter("postgres_catalogs", postgresCatalogConverter{})
}
