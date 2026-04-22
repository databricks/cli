package tfdyn

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/dyn"
)

// convertExternalLocationResource transforms a ucm external_location entry
// into a dyn.Value shaped like a databricks_external_location Terraform
// block. credential_name may be either a literal (bring-your-own) or a
// ${resources.storage_credentials.<key>.name} interpolation — the render-
// level Interpolate pass rewrites the latter to the tf-native form.
func convertExternalLocationResource(_ context.Context, key string, vin dyn.Value) (dyn.Value, error) {
	pairs := []dyn.Pair{}
	appendString(&pairs, vin, "name", key)

	urlVal := vin.Get("url")
	if s, _ := urlVal.AsString(); s == "" {
		return dyn.InvalidValue, fmt.Errorf("external_location %q: url is required", key)
	}
	pairs = append(pairs, dyn.Pair{
		Key:   dyn.NewValue("url", urlVal.Locations()),
		Value: urlVal,
	})

	credVal := vin.Get("credential_name")
	if s, _ := credVal.AsString(); s == "" {
		return dyn.InvalidValue, fmt.Errorf("external_location %q: credential_name is required", key)
	}
	pairs = append(pairs, dyn.Pair{
		Key:   dyn.NewValue("credential_name", credVal.Locations()),
		Value: credVal,
	})

	appendStringIfSet(&pairs, vin, "comment")
	appendBoolIfSet(&pairs, vin, "read_only")
	appendBoolIfSet(&pairs, vin, "skip_validation")
	appendBoolIfSet(&pairs, vin, "fallback")

	return dyn.NewValue(dyn.NewMappingFromPairs(pairs), vin.Locations()), nil
}

type externalLocationConverter struct{}

func (externalLocationConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *Resources) error {
	v, err := convertExternalLocationResource(ctx, key, vin)
	if err != nil {
		return err
	}
	out.ExternalLocation[key] = v
	return nil
}

func init() {
	registerConverter("external_locations", externalLocationConverter{})
}
