package apps

import (
	"context"
	"errors"
	"fmt"
	"os"

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
			skipTests      bool
		)

		deployCmd.Flags().BoolVar(&force, "force", false, "Force-override Git branch validation")
		deployCmd.Flags().BoolVar(&skipValidation, "skip-validation", false, "Skip project validation (build, typecheck, lint)")
		deployCmd.Flags().BoolVar(&skipTests, "skip-tests", true, "Skip running tests during validation")

		makeArgsOptionalWithBundle(deployCmd, "deploy [APP_NAME]")

		originalRunE := deployCmd.RunE
		deployCmd.RunE = func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				b := root.TryConfigureBundle(cmd)
				if b != nil {
					return runBundleDeploy(cmd, force, skipValidation, skipTests)
				}
			}

			err := originalRunE(cmd, args)
			return wrapError(cmd, deployReq.AppName, err)
		}

		deployCmd.Long = `Create an app deployment.

When run from a Databricks Apps project directory (containing databricks.yml)
without an APP_NAME argument, this command runs an enhanced deployment pipeline:
1. Validates the project (build, typecheck, lint for Node.js projects)
2. Deploys the project to the workspace
3. Runs the app

When an APP_NAME argument is provided (or when not in a project directory),
creates an app deployment using the API directly.

Arguments:
  APP_NAME: The name of the app. Required when not in a project directory.
            When provided in a project directory, uses API deploy instead of project deploy.

Examples:
  # Deploy from a project directory (enhanced flow with validation)
  databricks apps deploy

  # Deploy from a specific target
  databricks apps deploy --target prod

  # Deploy a specific app using the API (even from a project directory)
  databricks apps deploy my-app

  # Deploy from project with validation skip
  databricks apps deploy --skip-validation

  # Force deploy (override git branch validation)
  databricks apps deploy --force`
	}
}

// runBundleDeploy executes the enhanced deployment flow for project directories.
func runBundleDeploy(cmd *cobra.Command, force, skipValidation, skipTests bool) error {
	ctx := cmd.Context()

	// Get current working directory for validation
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Step 1: Validate project (unless skipped)
	if !skipValidation {
		validator := validation.GetProjectValidator(workDir)
		if validator != nil {
			opts := validation.ValidateOptions{
				SkipTests: skipTests,
			}
			result, err := validator.Validate(ctx, workDir, opts)
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
