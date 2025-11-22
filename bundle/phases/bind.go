package phases

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
)

func Bind(ctx context.Context, b *bundle.Bundle, opts *terraform.BindOptions) {
	log.Info(ctx, "Phase: bind")

	engine, err := engine.FromEnv(ctx)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	bundle.ApplyContext(ctx, b, lock.Acquire())
	if logdiag.HasError(ctx) {
		return
	}

	defer func() {
		bundle.ApplyContext(ctx, b, lock.Release(lock.GoalBind))
	}()

	bundle.ApplySeqContext(ctx, b,
		terraform.Interpolate(),
		terraform.Write(),
		terraform.Import(opts),
	)
	if logdiag.HasError(ctx) {
		return
	}

	diags := statemgmt.PushResourcesState(ctx, b, engine)
	for _, d := range diags {
		logdiag.LogDiag(ctx, d)
	}
}

func Unbind(ctx context.Context, b *bundle.Bundle, bundleType, tfResourceType, resourceKey string) {
	log.Info(ctx, "Phase: unbind")

	engine, err := engine.FromEnv(ctx)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	bundle.ApplyContext(ctx, b, lock.Acquire())
	if logdiag.HasError(ctx) {
		return
	}

	defer func() {
		bundle.ApplyContext(ctx, b, lock.Release(lock.GoalUnbind))
	}()

	bundle.ApplySeqContext(ctx, b,
		terraform.Interpolate(),
		terraform.Write(),
		terraform.Unbind(bundleType, tfResourceType, resourceKey),
	)
	if logdiag.HasError(ctx) {
		return
	}

	diags := statemgmt.PushResourcesState(ctx, b, engine)
	for _, d := range diags {
		logdiag.LogDiag(ctx, d)
	}
}
