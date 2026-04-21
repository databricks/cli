package phases

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
)

// Build renders the ucm configuration tree into the working directory's
// main.tf.json via the terraform wrapper's Render step. The underlying
// conversion is done by ucm/deploy/terraform/tfdyn.Convert and driven from
// (*terraform.Terraform).Render; Build is the phase-level veneer so cmd layer
// progress messages have a single pivot.
//
// Build returns the wrapper so callers can thread it into Plan/Deploy/Destroy
// without recreating it (important: terraform.New resolves the binary on
// disk, which is expensive). A nil return value means logdiag.HasError is
// set and the caller should bail.
func Build(ctx context.Context, u *ucm.Ucm, opts Options) TerraformWrapper {
	log.Info(ctx, "Phase: build")

	factory := opts.terraformFactoryOrDefault()
	tf, err := factory(ctx, u)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("build terraform wrapper: %w", err))
		return nil
	}
	if err := tf.Render(ctx, u); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("render terraform config: %w", err))
		return nil
	}
	return tf
}
