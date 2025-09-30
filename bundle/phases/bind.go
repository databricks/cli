package phases

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
)

func Bind(ctx context.Context, b *bundle.Bundle, opts *terraform.BindOptions) {
	log.Info(ctx, "Phase: bind")

	bundle.ApplyContext(ctx, b, lock.Acquire())
	if logdiag.HasError(ctx) {
		return
	}

	defer func() {
		bundle.ApplyContext(ctx, b, lock.Release(lock.GoalBind))
	}()

	bundle.ApplySeqContext(ctx, b,
		statemgmt.StatePull(),
		terraform.Interpolate(),
		terraform.Write(),
		terraform.Import(opts),
		statemgmt.StatePush(),
	)
}

func Unbind(ctx context.Context, b *bundle.Bundle, resourceType, resourceKey string) {
	log.Info(ctx, "Phase: unbind")

	bundle.ApplyContext(ctx, b, lock.Acquire())
	if logdiag.HasError(ctx) {
		return
	}

	defer func() {
		bundle.ApplyContext(ctx, b, lock.Release(lock.GoalUnbind))
	}()

	bundle.ApplySeqContext(ctx, b,
		statemgmt.StatePull(),
		terraform.Interpolate(),
		terraform.Write(),
		terraform.Unbind(resourceType, resourceKey),
		statemgmt.StatePush(),
	)
}
