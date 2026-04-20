package tfdyn

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/dyn"
)

// securableField maps a ucm securable type token to the field name on the
// databricks_grants Terraform resource. Only catalog and schema are in M0;
// volumes/external_locations/storage_credentials land in M2.
var securableField = map[string]string{
	"catalog": "catalog",
	"schema":  "schema",
}

// convertGrantResource emits a databricks_grants block for a single ucm
// grant entry. When the securable name matches a ucm-managed catalog or
// schema the grant references it through a TF interpolation
// (`${databricks_catalog.<key>.id}`) so terraform plans the dependency
// edge; otherwise the literal name is emitted.
func convertGrantResource(_ context.Context, vin dyn.Value, out *Resources) (dyn.Value, error) {
	secVal := vin.Get("securable")
	kindVal := secVal.Get("type")
	nameVal := secVal.Get("name")

	kind, _ := kindVal.AsString()
	name, _ := nameVal.AsString()

	field, ok := securableField[kind]
	if !ok {
		return dyn.InvalidValue, fmt.Errorf("unsupported securable type %q (M0 supports catalog, schema)", kind)
	}

	reference := name
	dependsOn := []dyn.Value{}
	switch field {
	case "catalog":
		if _, managed := out.Catalog[name]; managed {
			reference = fmt.Sprintf("${databricks_catalog.%s.name}", name)
			dependsOn = append(dependsOn, dyn.V("databricks_catalog."+name))
		}
	case "schema":
		if _, managed := out.Schema[name]; managed {
			reference = fmt.Sprintf("${databricks_schema.%s.id}", name)
			dependsOn = append(dependsOn, dyn.V("databricks_schema."+name))
		}
	}

	principalVal := vin.Get("principal")
	privilegesVal := vin.Get("privileges")

	grantEntry := dyn.NewValue(dyn.NewMappingFromPairs([]dyn.Pair{
		{
			Key:   dyn.NewValue("principal", principalVal.Locations()),
			Value: principalVal,
		},
		{
			Key:   dyn.NewValue("privileges", privilegesVal.Locations()),
			Value: privilegesVal,
		},
	}), vin.Locations())

	pairs := []dyn.Pair{
		{
			Key:   dyn.NewValue(field, nameVal.Locations()),
			Value: dyn.NewValue(reference, nameVal.Locations()),
		},
		{
			Key:   dyn.V("grant"),
			Value: dyn.V([]dyn.Value{grantEntry}),
		},
	}

	if len(dependsOn) > 0 {
		pairs = append(pairs, dyn.Pair{
			Key:   dyn.V("depends_on"),
			Value: dyn.V(dependsOn),
		})
	}

	return dyn.NewValue(dyn.NewMappingFromPairs(pairs), vin.Locations()), nil
}

type grantConverter struct{}

func (grantConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *Resources) error {
	v, err := convertGrantResource(ctx, vin, out)
	if err != nil {
		return fmt.Errorf("grant %q: %w", key, err)
	}
	out.Grants[key] = v
	return nil
}

func init() {
	registerConverter("grants", grantConverter{})
}
