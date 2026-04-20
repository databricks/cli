package tfdyn

import (
	"context"

	"github.com/databricks/cli/libs/dyn"
)

// convertSchemaResource shapes a ucm schema entry like a databricks_schema
// Terraform resource. When the parent catalog is a ucm-managed resource
// (i.e. a key present on out.Catalog) the emitted block carries a
// depends_on entry pointing at the catalog so terraform orders the graph
// correctly; free-form catalog names bypass the depends_on because no
// ucm-managed dependency exists.
func convertSchemaResource(_ context.Context, key string, vin dyn.Value, out *Resources) (dyn.Value, error) {
	pairs := []dyn.Pair{}
	appendString(&pairs, vin, "name", key)

	catalogField := vin.Get("catalog")
	catalogName, _ := catalogField.AsString()
	pairs = append(pairs, dyn.Pair{
		Key:   dyn.NewValue("catalog_name", catalogField.Locations()),
		Value: dyn.NewValue(catalogName, catalogField.Locations()),
	})

	appendStringIfSet(&pairs, vin, "comment")

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

	if _, managed := out.Catalog[catalogName]; managed {
		pairs = append(pairs, dyn.Pair{
			Key: dyn.V("depends_on"),
			Value: dyn.V([]dyn.Value{
				dyn.V("databricks_catalog." + catalogName),
			}),
		})
	}

	return dyn.NewValue(dyn.NewMappingFromPairs(pairs), vin.Locations()), nil
}

type schemaConverter struct{}

func (schemaConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *Resources) error {
	v, err := convertSchemaResource(ctx, key, vin, out)
	if err != nil {
		return err
	}
	out.Schema[key] = v
	return nil
}

func init() {
	registerConverter("schemas", schemaConverter{})
}
