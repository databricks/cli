package phases

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy/terraform"
)

// Plan runs the initialize → build → terraform-init → terraform-plan sequence
// and returns the resulting *terraform.PlanResult. Errors are reported via
// logdiag; on error Plan returns nil and the caller should check
// logdiag.HasError before rendering any output.
//
// Plan does NOT call state.Push — a plan never advances remote state. The
// deploy-side lock is held only for the state.Pull in Initialize and
// released; planning itself runs lock-free because it never writes.
func Plan(ctx context.Context, u *ucm.Ucm, opts Options) *terraform.PlanResult {
	log.Info(ctx, "Phase: plan")

	setting := Initialize(ctx, u, opts)
	if logdiag.HasError(ctx) || setting.Type.IsDirect() {
		return nil
	}

	tf := Build(ctx, u, opts)
	if tf == nil || logdiag.HasError(ctx) {
		return nil
	}

	if err := tf.Init(ctx, u); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("terraform init: %w", err))
		return nil
	}

	result, err := tf.Plan(ctx, u)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("terraform plan: %w", err))
		return nil
	}
	return result
}
