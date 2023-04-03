package terraform

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/libs/progress"
	"github.com/databricks/bricks/libs/terraform"
	"github.com/fatih/color"
	"github.com/hashicorp/terraform-exec/tfexec"
	"golang.org/x/term"
)

type plan struct {
	destroy bool
}

func (p *plan) Name() string {
	return "terraform.Plan"
}

func (p *plan) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	tf := b.Terraform
	if tf == nil {
		return nil, fmt.Errorf("terraform not initialized")
	}

	err := tf.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		return nil, fmt.Errorf("terraform init: %w", err)
	}

	// compute plan and get output in buf
	buf := &strings.Builder{}
	isDiff, err := tf.PlanJSON(ctx, buf, tfexec.Destroy(p.destroy))
	if err != nil {
		return nil, fmt.Errorf("terraform apply: %w", err)
	}
	if !isDiff {
		fmt.Fprintln(os.Stderr, "No resources to destroy!")
		return nil, nil
	}

	tfPlan := terraform.NewPlan()

	// store all events in struct
	for _, log := range strings.Split(buf.String(), "\n") {
		err := tfPlan.AddEvent(log)
		if err != nil {
			return nil, err
		}
	}

	// Log plan
	progressLogger, ok := progress.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("no progress logger found")
	}
	if tfPlan.ChangeSummary != nil && tfPlan.ChangeSummary.Summary != nil {
		fmt.Fprintf(os.Stderr, "Will destroy %d resources: \n\n", tfPlan.ChangeSummary.Summary.Remove)
	}
	for _, c := range tfPlan.PlannedChanges {
		progressLogger.Log(c.Change)
	}
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		// TODO: enforce forced flag here?
		b.ConfirmDestroy = true
		return nil, nil
	}

	red := color.New(color.FgRed).SprintFunc()
	fmt.Fprintf(os.Stderr, "\nThis will permanently %s resources! Proceed? [y/n]: ", red("destroy"))

	reader := bufio.NewReader(os.Stdin)
	ans, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if ans == "y\n" {
		fmt.Fprintln(os.Stderr, "Destroying resources!")
		b.ConfirmDestroy = true
	} else {
		fmt.Fprintln(os.Stderr, "Skipped!")
	}
	return nil, nil
}

// Apply returns a [bundle.Mutator] that runs the equivalent of `terraform apply`
// from the bundle's ephemeral working directory for Terraform.
func Plan(destroy bool) bundle.Mutator {
	return &plan{
		destroy: destroy,
	}
}
