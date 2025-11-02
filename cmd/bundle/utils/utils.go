package utils

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deployplan"
	bundleenv "github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/logdiag"
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

// getTargetFromCmd returns the target name from command flags or environment.
func getTargetFromCmd(cmd *cobra.Command) string {
	// Check command line flag first
	if flag := cmd.Flag("target"); flag != nil {
		if value := flag.Value.String(); value != "" {
			return value
		}
	}

	// Check deprecated environment flag
	if flag := cmd.Flag("environment"); flag != nil {
		if value := flag.Value.String(); value != "" {
			return value
		}
	}

	// Fall back to environment variable
	target, _ := bundleenv.Target(cmd.Context())
	return target
}

// ReloadBundle reloads the bundle configuration without modifying the command context.
// This is useful when you need to refresh the bundle configuration after changes
// without side effects like setting values on the context.
func ReloadBundle(cmd *cobra.Command) *bundle.Bundle {
	ctx := cmd.Context()

	// Load the bundle configuration fresh from the filesystem
	b := bundle.MustLoad(ctx)
	if b == nil || logdiag.HasError(ctx) {
		return b
	}

	// Load the target configuration
	if target := getTargetFromCmd(cmd); target == "" {
		phases.LoadDefaultTarget(ctx, b)
	} else {
		phases.LoadNamedTarget(ctx, b, target)
	}

	if logdiag.HasError(ctx) {
		return b
	}

	// Configure the workspace profile if provided
	configureProfile(cmd, b)

	// Configure variables if provided
	variables, err := cmd.Flags().GetStringSlice("var")
	if err != nil {
		logdiag.LogDiag(ctx, diag.FromErr(err)[0])
		return b
	}
	configureVariables(cmd, b, variables)
	return b
}

// configureProfile applies the profile flag to the bundle.
func configureProfile(cmd *cobra.Command, b *bundle.Bundle) {
	profile := getProfileFromCmd(cmd)
	if profile == "" {
		return
	}

	bundle.ApplyFuncContext(cmd.Context(), b, func(ctx context.Context, b *bundle.Bundle) {
		b.Config.Workspace.Profile = profile
	})
}

// getProfileFromCmd returns the profile from command flags or environment.
func getProfileFromCmd(cmd *cobra.Command) string {
	// Check command line flag first
	if flag := cmd.Flag("profile"); flag != nil {
		if value := flag.Value.String(); value != "" {
			return value
		}
	}

	// Fall back to environment variable
	return env.Get(cmd.Context(), "DATABRICKS_CONFIG_PROFILE")
}

func GetPlan(ctx context.Context, b *bundle.Bundle) (*deployplan.Plan, error) {
	plan := phases.Plan(ctx, b)
	if logdiag.HasError(ctx) {
		return nil, root.ErrAlreadyPrinted
	}

	// Direct engine includes noop actions, TF does not. This adds no-op actions for consistency:
	if !*b.DirectDeployment {
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
