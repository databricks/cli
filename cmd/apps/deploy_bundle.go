package apps

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/resources"
	"github.com/databricks/cli/bundle/run"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/libs/apps/validation"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/spf13/cobra"
)

// ErrorWrapper is a function type for wrapping deployment errors.
type ErrorWrapper func(cmd *cobra.Command, appName string, err error) error

// isBundleDirectory checks if the current directory contains a databricks.yml file.
func isBundleDirectory() bool {
	_, err := os.Stat("databricks.yml")
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
		deployCmd.Flags().BoolVar(&skipValidation, "skip-validation", false, "Skip project validation (build, typecheck, lint)")

		// Update the command usage to reflect that APP_NAME is optional when in bundle mode
		deployCmd.Use = "deploy [APP_NAME]"

		// Override Args to allow 0 or 1 arguments (bundle mode vs API mode)
		deployCmd.Args = func(cmd *cobra.Command, args []string) error {
			// In bundle mode, no arguments needed
			if isBundleDirectory() {
				if len(args) > 0 {
					return errors.New("APP_NAME argument is not allowed when deploying from a bundle directory")
				}
				return nil
			}
			// In API mode, exactly 1 argument required
			if len(args) != 1 {
				return fmt.Errorf("accepts 1 arg(s), received %d", len(args))
			}
			return nil
		}

		originalRunE := deployCmd.RunE
		deployCmd.RunE = func(cmd *cobra.Command, args []string) error {
			// If we're in a bundle directory, use the enhanced deploy flow
			if isBundleDirectory() {
				return runBundleDeploy(cmd, force, skipValidation)
			}

			// Otherwise, fall back to the original API deploy command
			err := originalRunE(cmd, args)
			return wrapError(cmd, deployReq.AppName, err)
		}

		// Update the help text to explain the dual behavior
		deployCmd.Long = `Create an app deployment.

When run from a directory containing a databricks.yml bundle configuration,
this command runs an enhanced deployment pipeline:
1. Validates the project (build, typecheck, lint for Node.js projects)
2. Deploys the bundle to the workspace
3. Runs the app

When run from a non-bundle directory, creates an app deployment using the API.

Arguments:
  APP_NAME: The name of the app (required only when not in a bundle directory).

Examples:
  # Deploy from a bundle directory (no app name required)
  databricks apps deploy

  # Deploy a specific app using the API
  databricks apps deploy my-app

  # Deploy from bundle with validation skip
  databricks apps deploy --skip-validation

  # Force deploy (override git branch validation)
  databricks apps deploy --force`
	}
}

// runBundleDeploy executes the enhanced deployment flow for bundle directories.
func runBundleDeploy(cmd *cobra.Command, force, skipValidation bool) error {
	ctx := cmd.Context()

	// Get current working directory for validation
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Step 1: Validate project (unless skipped)
	if !skipValidation {
		validator := getProjectValidator(workDir)
		if validator != nil {
			result, err := validator.Validate(ctx, workDir)
			if err != nil {
				return fmt.Errorf("validation error: %w", err)
			}

			if !result.Success {
				// Show error details
				if result.Details != nil {
					cmdio.LogString(ctx, result.Details.Error())
				}
				return errors.New("validation failed - fix errors before deploying")
			}
			cmdio.LogString(ctx, "✅ "+result.Message)
		} else {
			log.Debugf(ctx, "No validator found for project type, skipping validation")
		}
	}

	// Step 2: Deploy bundle
	cmdio.LogString(ctx, "Deploying bundle...")
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
		appName := b.Config.Resources.Apps[appKey].Name
		return fmt.Errorf("failed to run app: %w. Run `databricks apps logs %s` to view logs", err, appName)
	}

	cmdio.LogString(ctx, "✔ Deployment complete!")
	return nil
}

// getProjectValidator returns the appropriate validator based on project type.
// Returns nil if no validator is applicable.
func getProjectValidator(workDir string) validation.Validation {
	// Check for Node.js project (package.json exists)
	packageJSON := filepath.Join(workDir, "package.json")
	if _, err := os.Stat(packageJSON); err == nil {
		return &validation.ValidationNodeJs{}
	}
	return nil
}

// detectBundleApp finds the single app in the bundle configuration.
func detectBundleApp(b *bundle.Bundle) (string, error) {
	bundleApps := b.Config.Resources.Apps

	if len(bundleApps) == 0 {
		return "", errors.New("no apps found in bundle configuration")
	}

	if len(bundleApps) > 1 {
		return "", errors.New("multiple apps found in bundle, cannot auto-detect")
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
