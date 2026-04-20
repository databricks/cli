package phases

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy"
)

// Destroy runs the initialize → terraform-init → terraform-destroy →
// state-push sequence. The Build phase is skipped — destroy operates on the
// already-rendered terraform config cached from the last apply (tf.Init
// re-renders from current ucm.yml which is still necessary for the resource
// graph). The final Push uploads the post-destroy terraform.tfstate so peers
// observe the emptied state.
func Destroy(ctx context.Context, u *ucm.Ucm, opts Options) {
	log.Info(ctx, "Phase: destroy")

	setting := Initialize(ctx, u, opts)
	if logdiag.HasError(ctx) || setting.Type.IsDirect() {
		return
	}

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
