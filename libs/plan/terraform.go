package plan

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

const planDirName = "plan"

// TODO: start using this. Also clean up the fields in the struct below.
type TerraformPlanOpts struct {
	// If true, the plan will be a IsDestroy plan.
	IsDestroy bool

	// Path where the terraform project we'll compute the plan for is rooted at.
	Root string

	// Executable executable to compute the plan with.
	Executable *tfexec.Terraform
}

// TODO: Remove any unnecessary fields in here.
type terraformPlan struct {
	ctx context.Context

	opts TerraformPlanOpts

	// Path to the computed plan.
	planPath string

	// In memory representation of the computed plan, obtained by running terraform show.
	tfPlan *tfjson.Plan

	// TODO: Can we invert this?
	notEmpty bool
}

// NewTerraformPlan creates a new Terraform plan for a terraform project rooted
// at the given path. If destroy is true, the plan will be a destroy plan.
func NewTerraformPlan(ctx context.Context, opts TerraformPlanOpts) (Plan, error) {
	plan := &terraformPlan{
		ctx:  ctx,
		opts: opts,
	}

	if plan.opts.Executable == nil {
		return nil, fmt.Errorf("terraform executable not provided")
	}
	if plan.opts.Root == "" {
		return nil, fmt.Errorf("root path to compute the terraform plan not provided")
	}

	// Initialize the terraform project
	err := opts.Executable.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		return nil, fmt.Errorf("terraform init: %w", err)
	}

	// Path where the computed plan will be persisted.
	planPath := filepath.Join(opts.Root, planDirName)

	log.Debugf(ctx, "Computing terraform plan")
	plan.notEmpty, err = opts.Executable.Plan(ctx, tfexec.Destroy(opts.IsDestroy), tfexec.Out(planPath))
	if err != nil {
		return nil, fmt.Errorf("terraform plan: %w", err)
	}

	log.Debugf(ctx, "Loading terraform plan by running terraform show")
	plan.tfPlan, err = opts.Executable.ShowPlanFile(ctx, planPath)
	if err != nil {
		return nil, fmt.Errorf("terraform show: %w", err)
	}

	// Persist the plan path
	plan.planPath = planPath

	return plan, nil
}

func (p *terraformPlan) Path() string {
	return p.planPath
}

// TODO: Add tests for this functionality.
func (p *terraformPlan) ActionsByTypes(query ...ActionType) []Action {
	actions := make([]Action, 0)
	for _, rc := range p.tfPlan.ResourceChanges {
		if rc.Type == "databricks_permissions" {
			// We don't need to print permissions changes.
			continue
		}

		if rc.Type == "databricks_grants" {
			// We don't need to print grants changes.
			continue
		}

		tfActions := rc.Change.Actions
		switch {
		case slices.Contains(query, ActionTypeDelete) && tfActions.Delete():
			actions = append(actions, Action{
				rtype: strings.TrimPrefix(rc.Type, "databricks_"),
				rname: rc.Name,
				atype: ActionTypeDelete,
			})
		case slices.Contains(query, ActionTypeRecreate) && tfActions.Replace():
			actions = append(actions, Action{
				rtype: strings.TrimPrefix(rc.Type, "databricks_"),
				rname: rc.Name,
				atype: ActionTypeRecreate,
			})
		default:
			// We don't need to track other action types yet.
		}
	}

	// Sort by action type first, then by resource type, and finally by resource name
	sort.Slice(actions, func(i, j int) bool {
		if actions[i].atype != actions[j].atype {
			return actions[i].atype < actions[j].atype
		}
		if actions[i].rtype != actions[j].rtype {
			return actions[i].rtype < actions[j].rtype
		}
		return actions[i].rname < actions[j].rname
	})
	return actions
}

func (p *terraformPlan) IsEmpty() bool {
	return !p.notEmpty
}

func (p *terraformPlan) Apply() error {
	err := p.opts.Executable.Apply(p.ctx, tfexec.DirOrPlan(p.planPath))
	if err != nil {
		return fmt.Errorf("terraform apply: %w", err)
	}
	return nil
}
