package phases

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy"
	"github.com/databricks/cli/ucm/deploy/direct"
)

// Destroy runs the initialize → terraform-init → terraform-destroy →
// state-push sequence for the terraform engine, or the direct engine's
// equivalent: delete every recorded resource and persist the emptied state.
//
// For the terraform engine, the Build phase is skipped — destroy operates
// on the already-rendered terraform config cached from the last apply
// (tf.Init re-renders from current ucm.yml which is still necessary for the
// resource graph). The final Push uploads the post-destroy terraform.tfstate
// so peers observe the emptied state.
//
// For the direct engine, destroy walks the recorded state in reverse UC
// dependency order (grants → schemas → catalogs) and issues per-resource
// delete calls. The state file is rewritten on every successful delete so a
// mid-destroy error leaves the file consistent with whatever actually
// survived the run.
func Destroy(ctx context.Context, u *ucm.Ucm, opts Options) {
	log.Info(ctx, "Phase: destroy")

	setting := Initialize(ctx, u, opts)
	if logdiag.HasError(ctx) {
		return
	}

	if setting.Type.IsDirect() {
		destroyDirect(ctx, u, opts)
		return
	}
	destroyTerraform(ctx, u, opts)
}

func destroyTerraform(ctx context.Context, u *ucm.Ucm, opts Options) {
	factory := opts.terraformFactoryOrDefault()
	tf, err := factory(ctx, u)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("build terraform wrapper: %w", err))
		return
	}

	if err := tf.Init(ctx, u); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("terraform init: %w", err))
		return
	}

	if err := tf.Destroy(ctx, u); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("terraform destroy: %w", err))
		return
	}

	if err := deploy.Push(ctx, u, opts.Backend); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("push remote state: %w", err))
		return
	}
}

func destroyDirect(ctx context.Context, u *ucm.Ucm, opts Options) {
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

	_, destroyErr := direct.Destroy(ctx, u, client, state)
	if saveErr := direct.SaveState(statePath, state); saveErr != nil {
		if destroyErr == nil {
			logdiag.LogError(ctx, fmt.Errorf("save direct state: %w", saveErr))
			return
		}
		log.Warnf(ctx, "save direct state after destroy error: %v", saveErr)
	}
	if destroyErr != nil {
		logdiag.LogError(ctx, fmt.Errorf("direct destroy: %w", destroyErr))
	}
}
