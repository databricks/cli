package phases

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy"
)

// Deploy runs the initialize → build → terraform-init → terraform-apply →
// state-push sequence. Errors are reported via logdiag; state.Push is only
// called when the apply succeeds, so a mid-apply failure leaves the remote
// state on the previous Seq and the local cache updated but un-acknowledged.
//
// The terraform apply acquires its own deploy lock for the write window; the
// preceding Pull (in Initialize) and the subsequent Push each acquire and
// release the lock independently. Between those two lock windows the lock is
// released — intentional, because holding a remote lock across a long
// terraform apply would create availability problems for other targets.
func Deploy(ctx context.Context, u *ucm.Ucm, opts Options) {
	log.Info(ctx, "Phase: deploy")

	setting := Initialize(ctx, u, opts)
	if logdiag.HasError(ctx) || setting.Type.IsDirect() {
		return
	}

	tf := Build(ctx, u, opts)
	if tf == nil || logdiag.HasError(ctx) {
		return
	}

	if err := tf.Init(ctx, u); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("terraform init: %w", err))
		return
	}

	if err := tf.Apply(ctx, u); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("terraform apply: %w", err))
		return
	}

	if err := deploy.Push(ctx, u, opts.Backend); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("push remote state: %w", err))
		return
	}
}
