package terraform

import (
	"context"
	"fmt"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/libs/cmdio"
	"github.com/fatih/color"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type ApplyGoal string

var (
	ApplyDeploy  = ApplyGoal("deploy")
	ApplyDestroy = ApplyGoal("destroy")
)

type apply struct {
	goal ApplyGoal
}

func (w *apply) Name() string {
	return "terraform.Apply"
}

func (w *apply) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	// return early if plan is empty
	if b.Plan.IsEmpty {
		if w.goal == ApplyDeploy {
			cmdio.LogString(ctx, "No resource changes to apply. Skipping deploy!")
		}
		if w.goal == ApplyDestroy {
			cmdio.LogString(ctx, "No resources to destroy in plan. Skipping destroy!")
		}
		return nil, nil
	}

	tf := b.Terraform
	if tf == nil {
		return nil, fmt.Errorf("terraform not initialized")
	}

	err := tf.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		return nil, fmt.Errorf("terraform init: %w", err)
	}

	// Ask for confirmation, if needed
	if !b.Plan.ConfirmApply {
		red := color.New(color.FgRed).SprintFunc()
		yellow := color.New(color.FgYellow).SprintFunc()
		if b.Plan.IsReplacingResource {
			cmdio.LogString(ctx, fmt.Sprintf("One or more resources will be %s. Any previous metadata associated might be lost.", yellow("replaced")))
		}
		if b.Plan.IsDeletingResource {
			cmdio.LogString(ctx, fmt.Sprintf("One or more resources will be permanently %s", red("destroyed")))
		}
		b.Plan.ConfirmApply, err = cmdio.Ask(ctx, "Proceed with apply? [y/n]: ")
		if err != nil {
			return nil, err
		}
	}

	// return if confirmation was not provided
	if !b.Plan.ConfirmApply {
		return nil, nil
	}

	if b.Plan.Path == "" {
		return nil, fmt.Errorf("no plan found")
	}

	if w.goal == ApplyDeploy {
		cmdio.LogString(ctx, "\nStarting resource deployment")
	}
	if w.goal == ApplyDestroy {
		cmdio.LogString(ctx, "\nStarting resource destruction")
	}

	// Apply terraform according to the computed plan
	err = tf.Apply(ctx, tfexec.DirOrPlan(b.Plan.Path))
	if err != nil {
		return nil, fmt.Errorf("terraform apply: %w", err)
	}

	if w.goal == ApplyDeploy {
		cmdio.LogString(ctx, "Successfully deployed resources!")
	}
	if w.goal == ApplyDestroy {
		cmdio.LogString(ctx, "Successfully destroyed resources!")
	}
	return nil, nil
}

// Apply returns a [bundle.Mutator] that runs the equivalent of `terraform apply`
// from the bundle's ephemeral working directory for Terraform.
func Apply(goal ApplyGoal) bundle.Mutator {
	return &apply{
		goal: goal,
	}
}
