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
	"github.com/databricks/cli/experimental/apps-mcp/lib/validation"
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

	return cmd
}

func deployRun(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Load bundle to get work directory
	b, err := utils.ProcessBundle(cmd, utils.ProcessOptions{
		SkipInitialize: true,
	})
	if err != nil {
		return fmt.Errorf("failed to load bundle: %w", err)
	}

	workDir := b.BundleRootPath

	// Step 1: Validation
	log.Infof(ctx, "Running Node.js validation...")
	validator := &validation.ValidationNodeJs{}
	result, err := validator.Validate(ctx, workDir)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}
	if !result.Success {
		if result.Details != nil {
			return fmt.Errorf("validation failed (exit code %d): %s\nstderr: %s",
				result.Details.ExitCode, result.Message, result.Details.Stderr)
		}
		return fmt.Errorf("validation failed: %s", result.Message)
	}
	log.Infof(ctx, "Validation passed")

	// Step 2: Deploy
	log.Infof(ctx, "Deploying bundle...")
	b, err = utils.ProcessBundle(cmd, utils.ProcessOptions{
		AlwaysPull:   true,
		FastValidate: true,
		Build:        true,
		Deploy:       true,
	})
	if err != nil {
		return fmt.Errorf("deploy failed: %w", err)
	}
	log.Infof(ctx, "Deploy completed")

	// Step 3: Detect and run app
	appKey, err := detectApp(b)
	if err != nil {
		return err
	}

	log.Infof(ctx, "Running app: %s", appKey)
	err = runApp(ctx, cmd, b, appKey)
	if err != nil {
		return fmt.Errorf("failed to run app: %w", err)
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
func runApp(ctx context.Context, cmd *cobra.Command, b *bundle.Bundle, appKey string) error {
	// Lookup the app resource
	ref, err := resources.Lookup(b, appKey, run.IsRunnable)
	if err != nil {
		return fmt.Errorf("failed to lookup app: %w", err)
	}

	// Convert to runner
	runner, err := run.ToRunner(b, ref)
	if err != nil {
		return fmt.Errorf("failed to create runner: %w", err)
	}

	// Run the app
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
