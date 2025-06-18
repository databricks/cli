package phases

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
)

func Bind(ctx context.Context, b *bundle.Bundle, opts *terraform.BindOptions) (diags diag.Diagnostics) {
	log.Info(ctx, "Phase: bind")

	diags = bundle.Apply(ctx, b, lock.Acquire())
	if diags.HasError() {
		return diags
	}

	defer func() {
		diags = diags.Extend(bundle.Apply(ctx, b, lock.Release(lock.GoalBind)))
	}()

	diags = diags.Extend(bundle.ApplySeq(ctx, b,
		statemgmt.StatePull(),
		terraform.Interpolate(),
		terraform.Write(),
		terraform.Import(opts),
		statemgmt.StatePush(),
	))

	return diags
}

func Unbind(ctx context.Context, b *bundle.Bundle, resourceType, resourceKey string) (diags diag.Diagnostics) {
	log.Info(ctx, "Phase: unbind")

	diags = bundle.Apply(ctx, b, lock.Acquire())
	if diags.HasError() {
		return diags
	}

	defer func() {
		diags = diags.Extend(bundle.Apply(ctx, b, lock.Release(lock.GoalUnbind)))
	}()

	diags = diags.Extend(bundle.ApplySeq(ctx, b,
		statemgmt.StatePull(),
		terraform.Interpolate(),
		terraform.Write(),
		terraform.Unbind(resourceType, resourceKey),
		statemgmt.StatePush(),
	))

	return diags
}
