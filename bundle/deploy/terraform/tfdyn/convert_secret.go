package tfdyn

import (
	"context"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
)

type secretConverter struct{}

func (secretConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	// Normalize the output value to the target schema.
	vout, diags := convert.Normalize(schema.ResourceSecret{}, vin)
	for _, diag := range diags {
		log.Debugf(ctx, "secret normalization diagnostic: %s", diag.Summary)
	}

	vout, err := convertLifecycle(ctx, vout, vin.Get("lifecycle"))
	if err != nil {
		return err
	}

	// Add the converted resource to the output.
	out.Secret[key] = vout.AsAny()

	return nil
}

func init() {
	registerConverter("secrets", secretConverter{})
}
