package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

type loadMode int

const ErrorOnEmptyState loadMode = 0

type load struct {
	modes []loadMode
}

func (l *load) Name() string {
	return "terraform.Load"
}

// Partial representation of the latest Terraform state file format.
// We are only interested in resource types, names, modes, and ids.
type terraformState struct {
	Version   *int                     `json:"version"`
	Resources []terraformStateResource `json:"resources"`
}

type terraformStateResource struct {
	Type      string                           `json:"type"`
	Name      string                           `json:"name"`
	Mode      tfjson.ResourceMode              `json:"mode"`
	Instances []terraformStateResourceInstance `json:"instances"`
}

type terraformStateResourceInstance struct {
	Attributes terraformStateInstanceAttributes `json:"attributes"`
}

type terraformStateInstanceAttributes struct {
	ID string `json:"id"`
}

func (l *load) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	tf := b.Terraform
	if tf == nil {
		return diag.Errorf("terraform not initialized")
	}

	err := tf.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		return diag.Errorf("terraform init: %v", err)
	}

	rawState, err := b.Terraform.StatePull(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	var state terraformState
	if rawState != "" {
		err = json.Unmarshal([]byte(rawState), &state)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to parse deployment state: %w", err))
		}
	}

	err = l.validateState(&state)
	if err != nil {
		return diag.FromErr(err)
	}

	// Merge state into configuration.
	err = TerraformToBundle(&state, &b.Config)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func (l *load) validateState(state *terraformState) error {
	if state.Version != nil && *state.Version != 4 {
		return fmt.Errorf("unsupported deployment state version: %d. Try re-deploying the bundle", *state.Version)
	}

	if len(state.Resources) == 0 {
		if slices.Contains(l.modes, ErrorOnEmptyState) {
			return fmt.Errorf("no deployment state. Did you forget to run 'databricks bundle deploy'?")
		}
		return nil
	}

	return nil
}

func Load(modes ...loadMode) bundle.Mutator {
	return &load{modes: modes}
}
