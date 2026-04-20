package tfdyn

import (
	"context"

	"github.com/databricks/cli/libs/dyn"
)

// convertCatalogResource transforms a ucm catalog entry into a dyn.Value
// shaped like the databricks_catalog Terraform resource.
//
// The mapping is intentionally narrow (M0 fields only): name, comment,
// storage_root, and tags-as-properties. force_destroy is always set so ucm
// can manage the catalog's lifecycle through terraform.
func convertCatalogResource(_ context.Context, key string, vin dyn.Value) (dyn.Value, error) {
	pairs := []dyn.Pair{}
	appendString(&pairs, vin, "name", key)
	appendStringIfSet(&pairs, vin, "comment")
	appendStringIfSet(&pairs, vin, "storage_root")

	if tags, ok := mapFromValue(vin.Get("tags")); ok {
		pairs = append(pairs, dyn.Pair{
			Key:   dyn.NewValue("properties", vin.Get("tags").Locations()),
			Value: tags,
		})
	}

	pairs = append(pairs, dyn.Pair{
		Key:   dyn.V("force_destroy"),
		Value: dyn.V(true),
	})

	return dyn.NewValue(dyn.NewMappingFromPairs(pairs), vin.Locations()), nil
}

type catalogConverter struct{}

func (catalogConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *Resources) error {
	v, err := convertCatalogResource(ctx, key, vin)
	if err != nil {
		return err
	}
	out.Catalog[key] = v
	return nil
}

func init() {
	registerConverter("catalogs", catalogConverter{})
}
