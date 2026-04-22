package tfdyn

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/dyn"
)

// convertVolumeResource transforms a ucm volume entry into a dyn.Value
// shaped like a databricks_volume Terraform block. Validates the
// MANAGED-vs-EXTERNAL invariant: EXTERNAL volumes require
// storage_location, MANAGED ones must not carry one.
func convertVolumeResource(_ context.Context, key string, vin dyn.Value) (dyn.Value, error) {
	pairs := []dyn.Pair{}
	appendString(&pairs, vin, "name", key)

	catVal := vin.Get("catalog_name")
	if s, _ := catVal.AsString(); s == "" {
		return dyn.InvalidValue, fmt.Errorf("volume %q: catalog_name is required", key)
	}
	pairs = append(pairs, dyn.Pair{
		Key:   dyn.NewValue("catalog_name", catVal.Locations()),
		Value: catVal,
	})

	schemaVal := vin.Get("schema_name")
	if s, _ := schemaVal.AsString(); s == "" {
		return dyn.InvalidValue, fmt.Errorf("volume %q: schema_name is required", key)
	}
	pairs = append(pairs, dyn.Pair{
		Key:   dyn.NewValue("schema_name", schemaVal.Locations()),
		Value: schemaVal,
	})

	typeVal := vin.Get("volume_type")
	vType, _ := typeVal.AsString()
	vType = strings.ToUpper(vType)
	if vType != "MANAGED" && vType != "EXTERNAL" {
		return dyn.InvalidValue, fmt.Errorf("volume %q: volume_type must be MANAGED or EXTERNAL, got %q", key, vType)
	}
	pairs = append(pairs, dyn.Pair{
		Key:   dyn.NewValue("volume_type", typeVal.Locations()),
		Value: dyn.NewValue(vType, typeVal.Locations()),
	})

	locVal := vin.Get("storage_location")
	locStr, _ := locVal.AsString()
	switch vType {
	case "EXTERNAL":
		if locStr == "" {
			return dyn.InvalidValue, fmt.Errorf("volume %q: storage_location is required for EXTERNAL volumes", key)
		}
		pairs = append(pairs, dyn.Pair{
			Key:   dyn.NewValue("storage_location", locVal.Locations()),
			Value: locVal,
		})
	case "MANAGED":
		if locStr != "" {
			return dyn.InvalidValue, fmt.Errorf("volume %q: storage_location must not be set for MANAGED volumes", key)
		}
	}

	appendStringIfSet(&pairs, vin, "comment")

	return dyn.NewValue(dyn.NewMappingFromPairs(pairs), vin.Locations()), nil
}

type volumeConverter struct{}

func (volumeConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *Resources) error {
	v, err := convertVolumeResource(ctx, key, vin)
	if err != nil {
		return err
	}
	out.Volume[key] = v
	return nil
}

func init() {
	registerConverter("volumes", volumeConverter{})
}
