package tfdyn

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/apps"
)

func convertAppResource(ctx context.Context, vin dyn.Value) (dyn.Value, error) {
	// Check if the description is not set and if it's not, set it to an empty string.
	// This is done to avoid TF drift because Apps API return empty string for description when if it's not set.
	if _, err := dyn.Get(vin, "description"); err != nil {
		vin, err = dyn.Set(vin, "description", dyn.V(""))
		if err != nil {
			return vin, err
		}
	}

	// Normalize the output value to the target schema.
	vout, diags := convert.Normalize(apps.App{}, vin)
	for _, diag := range diags {
		log.Debugf(ctx, "app normalization diagnostic: %s", diag.Summary)
	}

	return vout, nil
}

type appConverter struct{}

func (appConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	vout, err := convertAppResource(ctx, vin)
	if err != nil {
		return err
	}

	// We always set no_compute to true as it allows DABs not to wait for app compute to be started when app is created.
	vout, err = dyn.Set(vout, "no_compute", dyn.V(true))
	if err != nil {
		return err
	}

	// Add the converted resource to the output.
	out.App[key] = vout.AsAny()

	// Configure permissions for this resource.
	if permissions := convertPermissionsResource(ctx, vin); permissions != nil {
		permissions.AppName = fmt.Sprintf("${databricks_app.%s.name}", key)
		out.Permissions["app_"+key] = permissions
	}

	return nil
}

func init() {
	registerConverter("apps", appConverter{})
}
