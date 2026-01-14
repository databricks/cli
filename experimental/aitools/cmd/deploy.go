package mcp

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/resources"
	"github.com/databricks/cli/bundle/run"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/experimental/aitools/lib/state"
	"github.com/databricks/cli/experimental/aitools/lib/validation"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/spf13/cobra"
)

func newDeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Validate, deploy bundle, and run app",
		Long: `Validate, deploy bundle, and run app.

This command runs a complete deployment pipeline:
1. Validates the Node.js project (npm install, build, typecheck, test)
2. Deploys the bundle to the workspace
3. Runs the app defined in the bundle

The command will stop immediately if any step fails.`,
		Args: root.NoArgs,
		RunE: deployRun,
	}

	// Add --var flag required by ProcessBundle
	cmd.Flags().StringSlice("var", []string{}, `set values for variables defined in bundle config. Example: --var="foo=bar"`)

	return cmd
}

func deployRun(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	workDir := "."

	// Load and verify state
	currentState, err := state.LoadState(workDir)
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}
	if currentState == nil {
		return errors.New("cannot deploy: project not validated (run validate first)")
	}

	// Verify checksum before deploy
	if currentState.GetChecksum() != "" {
		match, err := state.VerifyChecksum(workDir, currentState.GetChecksum())
		if err != nil {
			return fmt.Errorf("failed to verify checksum: %w", err)
		}
		if !match {
			return errors.New("cannot deploy: code changed since validation (run validate again)")
		}
	}

	// Check state transition is valid
	newState, err := currentState.Deploy()
	if err != nil {
		return err
	}

	log.Infof(ctx, "Running Node.js validation...")
	validator := &validation.ValidationNodeJs{}
	result, err := validator.Validate(ctx, workDir)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	cmdio.LogString(ctx, result.String())

	if !result.Success {
		return errors.New("validation failed")
	}
	log.Infof(ctx, "Validation passed")

	log.Infof(ctx, "Deploying bundle...")
	b, err := utils.ProcessBundle(cmd, utils.ProcessOptions{
		AlwaysPull:   true,
		FastValidate: true,
		Build:        true,
		Deploy:       true,
	})
	if err != nil {
		return fmt.Errorf("deploy failed: %w", err)
	}
	log.Infof(ctx, "Deploy completed")

	appKey, err := detectApp(b)
	if err != nil {
		return err
	}

	log.Infof(ctx, "Running app: %s", appKey)
	err = runApp(ctx, b, appKey)
	if err != nil {
		return fmt.Errorf("failed to run app: %w", err)
	}

	// Save deployed state
	if err := state.SaveState(workDir, newState); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

// detectApp finds the single app in the bundle configuration.
func detectApp(b *bundle.Bundle) (string, error) {
	apps := b.Config.Resources.Apps

	if len(apps) == 0 {
		return "", errors.New("no apps found in bundle configuration")
	}

	if len(apps) > 1 {
		return "", errors.New("multiple apps found in bundle, cannot auto-detect")
	}

	// Get the single app key
	for key := range apps {
		return key, nil
	}

	return "", errors.New("unexpected error detecting app")
}

// runApp runs the specified app using the runner interface.
func runApp(ctx context.Context, b *bundle.Bundle, appKey string) error {
	ref, err := resources.Lookup(b, appKey, run.IsRunnable)
	if err != nil {
		return fmt.Errorf("failed to lookup app: %w", err)
	}

	runner, err := run.ToRunner(b, ref)
	if err != nil {
		return fmt.Errorf("failed to create runner: %w", err)
	}

	output, err := runner.Run(ctx, &run.Options{})
	if err != nil {
		return fmt.Errorf("failed to run app: %w", err)
	}

	// Log output
	if output != nil {
		resultString, err := output.String()
		if err != nil {
			return err
		}
		log.Infof(ctx, "App output: %s", resultString)
	}

	return nil
}
