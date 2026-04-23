package phases

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/databricks/cli/ucm/deploy"
	"github.com/databricks/cli/ucm/deploy/direct"
)

// Deploy runs the initialize → build → terraform-init → terraform-apply →
// state-push sequence for the terraform engine, or the direct-apply path for
// the direct engine. Errors are reported via logdiag; terraform-engine
// state.Push is only called when the apply succeeds, so a mid-apply failure
// leaves the remote state on the previous Seq and the local cache updated
// but un-acknowledged.
//
// The terraform apply acquires its own deploy lock for the write window; the
// preceding Pull (in Initialize) and the subsequent Push each acquire and
// release the lock independently. Between those two lock windows the lock is
// released — intentional, because holding a remote lock across a long
// terraform apply would create availability problems for other targets.
//
// The direct engine is lock-free at this layer: state is a per-root local
// file and the SDK calls it issues serialize naturally through the UC API.
// Cross-process contention on the same target is a known gap — follow-up.
func Deploy(ctx context.Context, u *ucm.Ucm, opts Options) {
	log.Info(ctx, "Phase: deploy")

	setting := Initialize(ctx, u, opts)
	if logdiag.HasError(ctx) {
		return
	}

	if setting.Type.IsDirect() {
		deployDirect(ctx, u, opts)
		return
	}
	deployTerraform(ctx, u, opts)
}

func deployTerraform(ctx context.Context, u *ucm.Ucm, opts Options) {
	tf := Build(ctx, u, opts)
	if tf == nil || logdiag.HasError(ctx) {
		return
	}

	if err := tf.Init(ctx, u); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("terraform init: %w", err))
		return
	}

	if err := tf.Apply(ctx, u, opts.ForceLock); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("terraform apply: %w", err))
		return
	}

	pushBackend := opts.Backend
	pushBackend.ForceLock = opts.ForceLock
	if err := deploy.Push(ctx, u, pushBackend); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("push remote state: %w", err))
		return
	}
}

func deployDirect(ctx context.Context, u *ucm.Ucm, opts Options) {
	ucm.ApplyContext(ctx, u, mutator.ResolveResourceReferences())
	if logdiag.HasError(ctx) {
		return
	}

	factory := opts.directClientFactoryOrDefault()
	client, err := factory(ctx, u)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("resolve direct client: %w", err))
		return
	}

	statePath := direct.StatePath(u)
	state, err := direct.LoadState(statePath)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("load direct state: %w", err))
		return
	}

	plan := direct.CalculatePlan(u, state)
	applyErr := direct.Apply(ctx, u, client, plan, state)
	// Always persist state — Apply mutates it as it goes, so partial progress
	// from a mid-apply error must survive the process exit.
	if saveErr := direct.SaveState(statePath, state); saveErr != nil {
		if applyErr == nil {
			logdiag.LogError(ctx, fmt.Errorf("save direct state: %w", saveErr))
			return
		}
		log.Warnf(ctx, "save direct state after apply error: %v", saveErr)
	}
	if applyErr != nil {
		logdiag.LogError(ctx, fmt.Errorf("direct apply: %w", applyErr))
	}
}
