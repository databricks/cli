package app

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
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/experimental/dev/lib/validation"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/spf13/cobra"
)

func newDeployCmd() *cobra.Command {
	var (
		force          bool
		skipValidation bool
	)

	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Validate, deploy the AppKit application and run it",
		Long: `Validate, deploy the AppKit application and run it.

This command runs a deployment pipeline:
1. Validates the project (build, typecheck, tests for Node.js projects)
2. Deploys the bundle to the workspace
3. Runs the app

Examples:
  # Deploy to default target
  databricks experimental dev app deploy

  # Deploy to a specific target
  databricks experimental dev app deploy --target prod

  # Skip validation (if already validated)
  databricks experimental dev app deploy --skip-validation

  # Force deploy (override git branch validation)
  databricks experimental dev app deploy --force

  # Set bundle variables
  databricks experimental dev app deploy --var="warehouse_id=abc123"`,
		Args: root.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeploy(cmd, force, skipValidation)
		},
	}

	cmd.Flags().StringP("target", "t", "", "Deployment target (e.g., dev, prod)")
	cmd.Flags().BoolVar(&force, "force", false, "Force-override Git branch validation")
	cmd.Flags().BoolVar(&skipValidation, "skip-validation", false, "Skip project validation (build, typecheck, lint)")
	cmd.Flags().StringSlice("var", []string{}, `Set values for variables defined in bundle config. Example: --var="key=value"`)

	return cmd
}

func runDeploy(cmd *cobra.Command, force, skipValidation bool) error {
	ctx := cmd.Context()

	// Check for bundle configuration
	if _, err := os.Stat("databricks.yml"); os.IsNotExist(err) {
		return errors.New("no databricks.yml found; run this command from a bundle directory")
	}

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
	if err := runApp(ctx, b, appKey); err != nil {
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

// detectApp finds the single app in the bundle configuration.
func detectApp(b *bundle.Bundle) (string, error) {
	apps := b.Config.Resources.Apps

	if len(apps) == 0 {
		return "", errors.New("no apps found in bundle configuration")
	}

	if len(apps) > 1 {
		return "", errors.New("multiple apps found in bundle, cannot auto-detect")
	}

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

	if output != nil {
		resultString, err := output.String()
		if err != nil {
			return err
		}
		log.Infof(ctx, "App output: %s", resultString)
	}

	return nil
}
