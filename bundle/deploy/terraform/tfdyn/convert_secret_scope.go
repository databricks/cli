package tfdyn

import (
	"context"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

type secretScopeConverter struct{}

func (s secretScopeConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	// Normalize the output value to the target schema.
	vout, diags := convert.Normalize(workspace.SecretScope{}, vin)
	for _, diag := range diags {
		log.Debugf(ctx, "secret scope normalization diagnostic: %s", diag.Summary)
	}

	// Add the converted resource to the output.
	out.SecretScope[key] = vout.AsAny()

	return nil
}

func init() {
	registerConverter("secret_scopes", secretScopeConverter{})
}
