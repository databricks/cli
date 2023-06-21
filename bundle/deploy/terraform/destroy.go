package terraform

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/fatih/color"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

type PlanResourceChange struct {
	ResourceType string `json:"resource_type"`
	Action       string `json:"action"`
	ResourceName string `json:"resource_name"`
}

func (c *PlanResourceChange) String() string {
	result := strings.Builder{}
	switch c.Action {
	case "delete":
		result.WriteString("  delete ")
	default:
		result.WriteString(c.Action + " ")
	}
	switch c.ResourceType {
	case "databricks_job":
		result.WriteString("job ")
	case "databricks_pipeline":
		result.WriteString("pipeline ")
	default:
		result.WriteString(c.ResourceType + " ")
	}
	result.WriteString(c.ResourceName)
	return result.String()
}

func (c *PlanResourceChange) IsInplaceSupported() bool {
	return false
}

func logDestroyPlan(ctx context.Context, changes []*tfjson.ResourceChange) error {
	cmdio.LogString(ctx, "The following resources will be removed:")
	for _, c := range changes {
		if c.Change.Actions.Delete() {
			cmdio.Log(ctx, &PlanResourceChange{
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

func (w *destroy) Apply(ctx context.Context, b *bundle.Bundle) error {
	// return early if plan is empty
	if b.Plan.IsEmpty {
		cmdio.LogString(ctx, "No resources to destroy in plan. Skipping destroy!")
		return nil
	}

	tf := b.Terraform
	if tf == nil {
		return fmt.Errorf("terraform not initialized")
	}

	// read plan file
	plan, err := tf.ShowPlanFile(ctx, b.Plan.Path)
	if err != nil {
		return err
	}

	// print the resources that will be destroyed
	err = logDestroyPlan(ctx, plan.ResourceChanges)
	if err != nil {
		return err
	}

	// Ask for confirmation, if needed
	if !b.Plan.ConfirmApply {
		red := color.New(color.FgRed).SprintFunc()
		b.Plan.ConfirmApply, err = cmdio.Ask(ctx, fmt.Sprintf("\nThis will permanently %s resources! Proceed? [y/n]: ", red("destroy")))
		if err != nil {
			return err
		}
	}

	// return if confirmation was not provided
	if !b.Plan.ConfirmApply {
		return nil
	}

	if b.Plan.Path == "" {
		return fmt.Errorf("no plan found")
	}

	cmdio.LogString(ctx, "Starting to destroy resources")

	// Apply terraform according to the computed destroy plan
	err = tf.Apply(ctx, tfexec.DirOrPlan(b.Plan.Path))
	if err != nil {
		return fmt.Errorf("terraform destroy: %w", err)
	}

	cmdio.LogString(ctx, "Successfully destroyed resources!")
	return nil
}

// Destroy returns a [bundle.Mutator] that runs the conceptual equivalent of
// `terraform destroy ./plan` from the bundle's ephemeral working directory for Terraform.
func Destroy() bundle.Mutator {
	return &destroy{}
}
