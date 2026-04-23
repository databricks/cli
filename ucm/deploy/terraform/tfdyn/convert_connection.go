package tfdyn

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/dyn"
)

// convertConnectionResource transforms a ucm connection entry into a
// dyn.Value shaped like a databricks_connection Terraform block.
// options is required and must contain at least one key; the TF provider
// handles per-connection-type validation of which keys are expected.
func convertConnectionResource(_ context.Context, key string, vin dyn.Value) (dyn.Value, error) {
	pairs := []dyn.Pair{}
	appendString(&pairs, vin, "name", key)

	typeVal := vin.Get("connection_type")
	if s, _ := typeVal.AsString(); s == "" {
		return dyn.InvalidValue, fmt.Errorf("connection %q: connection_type is required", key)
	}
	pairs = append(pairs, dyn.Pair{
		Key:   dyn.NewValue("connection_type", typeVal.Locations()),
		Value: typeVal,
	})

	optsVal := vin.Get("options")
	optsMap, ok := optsVal.AsMap()
	if !ok || optsMap.Len() == 0 {
		return dyn.InvalidValue, fmt.Errorf("connection %q: options is required and must be non-empty", key)
	}
	pairs = append(pairs, dyn.Pair{
		Key:   dyn.NewValue("options", optsVal.Locations()),
		Value: optsVal,
	})

	appendStringIfSet(&pairs, vin, "comment")
	if props, ok := mapFromValue(vin.Get("properties")); ok {
		pairs = append(pairs, dyn.Pair{
			Key:   dyn.NewValue("properties", vin.Get("properties").Locations()),
			Value: props,
		})
	}
	appendBoolIfSet(&pairs, vin, "read_only")

	return dyn.NewValue(dyn.NewMappingFromPairs(pairs), vin.Locations()), nil
}

type connectionConverter struct{}

func (connectionConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *Resources) error {
	v, err := convertConnectionResource(ctx, key, vin)
	if err != nil {
		return err
	}
	out.Connection[key] = v
	return nil
}

func init() {
	registerConverter("connections", connectionConverter{})
}
