package utils

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/spf13/cobra"
)

func configureVariables(cmd *cobra.Command, b *bundle.Bundle, variables []string) {
	bundle.ApplyFuncContext(cmd.Context(), b, func(ctx context.Context, b *bundle.Bundle) {
		err := b.Config.InitializeVariables(variables)
		if err != nil {
			logdiag.LogError(ctx, err)
		}
	})
}

func ConfigureBundleWithVariables(cmd *cobra.Command) *bundle.Bundle {
	// Load bundle config and apply target
	b := root.MustConfigureBundle(cmd)
	ctx := cmd.Context()
	if logdiag.HasError(ctx) {
		return b
	}

	variables, err := cmd.Flags().GetStringSlice("var")
	if err != nil {
		logdiag.LogDiag(ctx, diag.FromErr(err)[0])
		return b
	}

	// Initialize variables by assigning them values passed as command line flags
	configureVariables(cmd, b, variables)

	engine, err := DeploymentEngine(ctx)
	if err != nil {
		logdiag.LogError(ctx, err)
		return b
	}

	// We use "direct-exp" while direct backend is not suitable for end users.
	// Once we consider it usable we'll change the value to "direct".
	// This is to prevent accidentally running direct backend with older CLI versions where it was still considered experimental.
	b.DirectDeployment = engine == "direct-exp"

	// Set the engine in the user agent
	ctx = useragent.InContext(ctx, "engine", engine)
	cmd.SetContext(ctx)
	return b
}

func DeploymentEngine(ctx context.Context) (string, error) {
	engine := env.Get(ctx, "DATABRICKS_BUNDLE_ENGINE")

	// By default, use Terraform
	if engine == "" {
		return "terraform", nil
	}

	if engine != "terraform" && engine != "direct-exp" {
		return "", fmt.Errorf("unexpected setting for DATABRICKS_BUNDLE_ENGINE=%#v (expected 'terraform' or 'direct-exp' or absent/empty which means 'terraform')", engine)
	}

	return engine, nil
}

func GetPlan(ctx context.Context, b *bundle.Bundle) (*deployplan.Plan, error) {
	phases.Initialize(ctx, b)
	if logdiag.HasError(ctx) {
		return nil, root.ErrAlreadyPrinted
	}

	bundle.ApplyContext(ctx, b, validate.FastValidate())
	if logdiag.HasError(ctx) {
		return nil, root.ErrAlreadyPrinted
	}

	phases.Build(ctx, b)
	if logdiag.HasError(ctx) {
		return nil, root.ErrAlreadyPrinted
	}

	plan := phases.Plan(ctx, b)
	if logdiag.HasError(ctx) {
		return nil, root.ErrAlreadyPrinted
	}

	// Direct engine includes noop actions, TF does not. This adds no-op actions for consistency:
	if !b.DirectDeployment {
		for _, group := range b.Config.Resources.AllResources() {
			for rKey := range group.Resources {
				resourceKey := "resources." + group.Description.PluralName + "." + rKey
				if _, ok := plan.Plan[resourceKey]; !ok {
					plan.Plan[resourceKey] = &deployplan.PlanEntry{
						Action: deployplan.ActionTypeSkip.String(),
					}
				}
			}
		}
	}

	return plan, nil
}
