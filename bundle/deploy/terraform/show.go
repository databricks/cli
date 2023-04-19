package terraform

import (
	"context"
	"fmt"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/libs/cmdio"
	"github.com/hashicorp/terraform-exec/tfexec"
)

type showPlan struct{}

func (m *showPlan) Name() string {
	return "terraform.ShowPlan"
}

func (m *showPlan) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	tf := b.Terraform
	if tf == nil {
		return nil, fmt.Errorf("terraform not initialized")
	}

	err := tf.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		return nil, fmt.Errorf("terraform init: %w", err)
	}

	// read plan file
	plan, err := tf.ShowPlanFile(ctx, b.Plan.Path)
	if err != nil {
		return nil, err
	}

	// compute bundle specific change events
	changeEvents := make([]*ResourceChangeEvent, 0)
	for _, change := range plan.ResourceChanges {
		if change.Change == nil {
			continue
		}
		if change.Change.Actions.Replace() {
			b.Plan.IsReplacingResource = true
		}
		if change.Change.Actions.Delete() {
			b.Plan.IsDeletingResource = true
		}
		event := toResourceChangeEvent(change)
		if event == nil {
			continue
		}
		changeEvents = append(changeEvents, event)
	}

	// return without logging anything if no relevant change events in computed plan
	if len(changeEvents) == 0 {
		return nil, nil
	}

	// log resource changes
	cmdio.LogString(ctx, "The following resource changes will be applied:")
	for _, event := range changeEvents {
		cmdio.Log(ctx, event)
	}
	cmdio.LogNewline(ctx)
	return nil, nil
}

func ShowPlan() bundle.Mutator {
	return &showPlan{}
}
