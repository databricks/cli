package apps

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/resources"
	"github.com/databricks/cli/bundle/run"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/apps/validation"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/spf13/cobra"
)

// ErrorWrapper is a function type for wrapping deployment errors.
type ErrorWrapper func(cmd *cobra.Command, appName string, err error) error

// hasBundleConfig checks if the current directory contains a bundle configuration file.
func hasBundleConfig() bool {
	wd, err := os.Getwd()
	if err != nil {
		return false
	}
	_, err = config.FileNames.FindInPath(wd)
	return err == nil
}

// BundleDeployOverrideWithWrapper creates a deploy override function that uses
// the provided error wrapper for API fallback errors.
func BundleDeployOverrideWithWrapper(wrapError ErrorWrapper) func(*cobra.Command, *apps.CreateAppDeploymentRequest) {
	return func(deployCmd *cobra.Command, deployReq *apps.CreateAppDeploymentRequest) {
		var (
			force          bool
			skipValidation bool
		)

		deployCmd.Flags().BoolVar(&force, "force", false, "Force-override Git branch validation")
		deployCmd.Flags().BoolVar(&skipValidation, "skip-validation", false, "Skip project validation entirely")

		makeArgsOptionalWithBundle(deployCmd, "deploy [APP_NAME]")

		originalRunE := deployCmd.RunE
		deployCmd.RunE = func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				b := root.TryConfigureBundle(cmd)
				if b != nil {
					return runBundleDeploy(cmd, force, skipValidation)
				}
			}

			err := originalRunE(cmd, args)
			return wrapError(cmd, deployReq.AppName, err)
		}

		deployCmd.Long = `Create an app deployment.

When run from a Databricks Apps project directory (containing databricks.yml)
without an APP_NAME argument, this command runs an enhanced deployment pipeline:
1. Validates the project if needed (code changed or never validated)
2. Deploys the project to the workspace
3. Runs the app

Validation is automatically run when:
- No previous validation state exists
- Code has changed since last validation (checksum mismatch)

When an APP_NAME argument is provided (or when not in a project directory),
creates an app deployment using the API directly.

Arguments:
  APP_NAME: The name of the app. Required when not in a project directory.
            When provided in a project directory, uses API deploy instead of project deploy.

Examples:
  # Deploy from a project directory (auto-validates if needed)
  databricks apps deploy

  # Deploy from a specific target
  databricks apps deploy --target prod

  # Deploy a specific app using the API (even from a project directory)
  databricks apps deploy my-app

  # Skip validation entirely
  databricks apps deploy --skip-validation

  # Force deploy (override git branch validation)
  databricks apps deploy --force`
	}
}

// runBundleDeploy executes the enhanced deployment flow for project directories.
func runBundleDeploy(cmd *cobra.Command, force, skipValidation bool) error {
	ctx := cmd.Context()

	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Step 1: Validate if needed (unless --skip-validation)
	if !skipValidation {
		if err := validateIfNeeded(cmd, workDir); err != nil {
			return err
		}
	} else {
		log.Debugf(ctx, "Skipping validation (--skip-validation)")
	}

	// Step 2: Deploy project
	cmdio.LogString(ctx, "Deploying project...")
	b, err := utils.ProcessBundle(cmd, utils.ProcessOptions{
		InitFunc: func(b *bundle.Bundle) {
			b.Config.Bundle.Force = force
		},
		// Context is already initialized by the workspace command's PreRunE
		SkipInitContext: true,
		AlwaysPull:      true,
		FastValidate:    true,
		Build:           true,
		Deploy:          true,
	})
	if err != nil {
		return fmt.Errorf("deploy failed: %w", err)
	}
	log.Infof(ctx, "Deploy completed")

	// Step 3: Detect and run app
	appKey, err := detectBundleApp(b)
	if err != nil {
		return err
	}

	log.Infof(ctx, "Running app: %s", appKey)
	if err := runBundleApp(ctx, b, appKey); err != nil {
		cmdio.LogString(ctx, "✔ Deployment succeeded, but failed to start app")
		return fmt.Errorf("failed to run app: %w. Run `databricks apps logs` to view logs", err)
	}

	// Step 4: Update state to deployed
	if err := updateStateToDeployed(workDir); err != nil {
		log.Warnf(ctx, "Failed to update state file: %v", err)
	}

	cmdio.LogString(ctx, "✔ Deployment complete!")
	return nil
}

// detectBundleApp finds the single app in the project configuration.
func detectBundleApp(b *bundle.Bundle) (string, error) {
	bundleApps := b.Config.Resources.Apps

	if len(bundleApps) == 0 {
		return "", errors.New("no apps found in project configuration")
	}

	if len(bundleApps) > 1 {
		return "", errors.New("multiple apps found in project, cannot auto-detect")
	}

	for key := range bundleApps {
		return key, nil
	}

	return "", errors.New("unexpected error detecting app")
}

// runBundleApp runs the specified app using the runner interface.
func runBundleApp(ctx context.Context, b *bundle.Bundle, appKey string) error {
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

	if output != nil {
		resultString, err := output.String()
		if err != nil {
			return err
		}
		log.Infof(ctx, "App output: %s", resultString)
	}

	return nil
}

// validateIfNeeded checks validation state and runs validation if needed.
func validateIfNeeded(cmd *cobra.Command, workDir string) error {
	ctx := cmd.Context()

	state, err := validation.LoadState(workDir)
	if err != nil {
		return fmt.Errorf("failed to load validation state: %w", err)
	}

	currentChecksum, err := validation.ComputeChecksum(workDir)
	if err != nil {
		return fmt.Errorf("failed to compute checksum: %w", err)
	}

	// Check if validation is needed
	needsValidation := state == nil || currentChecksum != state.Checksum
	if !needsValidation {
		log.Debugf(ctx, "Validation state up-to-date (checksum: %s...)", state.Checksum[:12])
		return nil
	}

	if state == nil {
		log.Infof(ctx, "No previous validation state, running validation...")
	} else {
		log.Infof(ctx, "Code changed since last validation, re-validating...")
	}

	// Run validation
	opts := validation.ValidateOptions{}
	validator := validation.GetProjectValidator(workDir)
	if validator != nil {
		result, err := validator.Validate(ctx, workDir, opts)
		if err != nil {
			return fmt.Errorf("validation error: %w", err)
		}

		if !result.Success {
			if result.Details != nil {
				cmdio.LogString(ctx, result.Details.Error())
			}
			return errors.New("validation failed - fix errors before deploying")
		}
		cmdio.LogString(ctx, "✅ "+result.Message)
	} else {
		log.Debugf(ctx, "No validator found for project type, skipping validation checks")
	}

	// Save state
	newState := &validation.State{
		State:       validation.StateValidated,
		ValidatedAt: time.Now().UTC(),
		Checksum:    currentChecksum,
	}
	if err := validation.SaveState(workDir, newState); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

// updateStateToDeployed updates the state file to mark the project as deployed.
func updateStateToDeployed(workDir string) error {
	state, err := validation.LoadState(workDir)
	if err != nil {
		return err
	}

	if state == nil {
		// No state file, nothing to update
		return nil
	}

	state.State = validation.StateDeployed
	return validation.SaveState(workDir, state)
}
