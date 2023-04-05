package terraform

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/libs/cmdio"
	"github.com/fatih/color"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

// TODO: This is temporary. Come up with a robust way to log mutator progress and
// status events
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

func logDestroyPlan(l *cmdio.Logger, changes []*tfjson.ResourceChange) error {
	// TODO: remove once we have mutator logging in place
	fmt.Fprintln(os.Stderr, "The following resources will be removed: ")
	for _, c := range changes {
		if c.Change.Actions.Delete() {
			l.Log(&PlanResourceChange{
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
	// interface to io with the user
	logger, ok := cmdio.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("no logger found")
	}

	if b.Plan.IsEmpty {
		fmt.Fprintln(os.Stderr, "No resources to destroy!")
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
	err = logDestroyPlan(logger, plan.ResourceChanges)
	if err != nil {
		return nil, err
	}

	// Ask for confirmation, if needed
	if !b.Plan.ConfirmApply {
		red := color.New(color.FgRed).SprintFunc()
		b.Plan.ConfirmApply, err = logger.Ask(fmt.Sprintf("\nThis will permanently %s resources! Proceed? [y/n]: ", red("destroy")))
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

	fmt.Fprintln(os.Stderr, "Successfully destroyed resources!")
	return nil, nil
}

// Destroy returns a [bundle.Mutator] that runs the equivalent of `terraform destroy`
// from the bundle's ephemeral working directory for Terraform.
func Destroy() bundle.Mutator {
	return &destroy{}
}
