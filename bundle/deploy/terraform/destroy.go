package terraform

import (
	"context"
	"fmt"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/libs/cmdio"
	"github.com/fatih/color"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

func (w *destroy) logDestroyPlan(ctx context.Context, changes []*tfjson.ResourceChange) error {
	cmdio.LogMutatorEvent(ctx, w.Name(), cmdio.MutatorRunning, "The following resources will be removed: ")
	for _, c := range changes {
		if c.Change.Actions.Delete() {
			cmdio.Log(ctx, &ResourceChange{
				ResourceType: c.Type,
				Action:       "delete",
				ResourceName: c.Name,
			})
		}
	}
	return nil
}

type destroy struct{}

func (w *destroy) Name() string {
	return "terraform.Destroy"
}

func (w *destroy) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	// return early if plan is empty
	if b.Plan.IsEmpty {
		cmdio.LogMutatorEvent(ctx, w.Name(), cmdio.MutatorCompleted, "No resources to destroy!\n")
		return nil, nil
	}

	tf := b.Terraform
	if tf == nil {
		return nil, fmt.Errorf("terraform not initialized")
	}

	// read plan file
	plan, err := tf.ShowPlanFile(ctx, b.Plan.Path)
	if err != nil {
		return nil, err
	}

	// print the resources that will be destroyed
	err = w.logDestroyPlan(ctx, plan.ResourceChanges)
	if err != nil {
		return nil, err
	}

	// Ask for confirmation, if needed
	if !b.Plan.ConfirmApply {
		red := color.New(color.FgRed).SprintFunc()
		b.Plan.ConfirmApply, err = cmdio.Ask(ctx, fmt.Sprintf("\nThis will permanently %s resources! Proceed? [y/n]: ", red("destroy")))
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

	// Apply terraform according to the computed destroy plan
	err = tf.Apply(ctx, tfexec.DirOrPlan(b.Plan.Path))
	if err != nil {
		return nil, fmt.Errorf("terraform destroy: %w", err)
	}

	cmdio.LogMutatorEvent(ctx, w.Name(), cmdio.MutatorCompleted, "Successfully destroyed resources!\n")
	return nil, nil
}

// Destroy returns a [bundle.Mutator] that runs the conceptual equivalent of
// `terraform destroy ./plan` from the bundle's ephemeral working directory for Terraform.
func Destroy() bundle.Mutator {
	return &destroy{}
}
