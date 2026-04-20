package terraform

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/ucm"
	"github.com/hashicorp/terraform-exec/tfexec"
)

// Init ensures main.tf.json is current (calls Render) and then runs
// `terraform init` in the working directory.
//
// Init is idempotent: calling it twice on the same *Terraform will re-render
// main.tf.json (cheap) but skip the underlying terraform init the second
// time. Callers that want to force a re-init (e.g. after a provider version
// bump) should construct a fresh *Terraform.
func (t *Terraform) Init(ctx context.Context, u *ucm.Ucm) error {
	if t == nil {
		return fmt.Errorf("terraform: nil wrapper")
	}

	if err := t.Render(ctx, u); err != nil {
		return err
	}

	if err := t.ensureRunner(ctx); err != nil {
		return err
	}

	if t.initialized {
		log.Debugf(ctx, "terraform init: already initialized, skipping")
		return nil
	}

	if err := t.runner.Init(ctx, tfexec.Upgrade(true)); err != nil {
		return fmt.Errorf("terraform init: %w", err)
	}
	t.initialized = true
	log.Infof(ctx, "terraform init completed")
	return nil
}
