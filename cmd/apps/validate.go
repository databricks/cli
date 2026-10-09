package apps

import (
	"errors"
	"fmt"
	"os"

	"github.com/databricks/cli/libs/apps/validation"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate a Databricks App project",
		Long: `Validate a Databricks App project by running build, typecheck, and lint checks.

This command detects the project type and runs the appropriate validation:
- Node.js projects (package.json): installs dependencies and runs typegen,
  lint:ast-grep, typecheck, build, and test scripts when present

An npm or pnpm packageManager declaration in package.json takes precedence over
lockfiles in the project directory. Conflicting lockfiles produce a warning.
Without a supported declaration, the package manager is detected from lockfiles:
- package-lock.json or npm-shrinkwrap.json: npm
- pnpm-lock.yaml: pnpm

Only npm and pnpm are supported. Unsupported packageManager declarations and
Yarn/Bun lockfiles produce a warning and are ignored. Without a supported
declaration, projects with both npm and pnpm lockfiles warn and use npm;
projects without a supported lockfile also use npm.
The selected package manager must be on PATH in the project directory.
Dependencies are installed when node_modules is absent.

Scripts use the package manager's normal workspace behavior. For pnpm workspaces,
define root scripts that run the checks for the packages you want to validate.

For AppKit projects (appkit.plugins.json), it first checks that resources accessed on
behalf of the user are not app-only and that their scopes are in user_api_scopes.

Examples:
  # Validate the current directory
  databricks apps validate

  # Validate a specific directory
  databricks apps validate --path ./my-app

  # Run a quick validation without tests
  databricks apps validate --skip-tests`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(cmd)
		},
	}

	cmd.Flags().String("path", "", "Path to the project directory (defaults to current directory)")
	cmd.Flags().Bool("skip-tests", false, "Skip running tests for faster validation")

	return cmd
}

func runValidate(cmd *cobra.Command) error {
	ctx := cmd.Context()

	// Get project path
	projectPath, _ := cmd.Flags().GetString("path")
	if projectPath == "" {
		var err error
		projectPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	// Get validation options
	skipTests, _ := cmd.Flags().GetBool("skip-tests")
	opts := validation.ValidateOptions{
		SkipTests: skipTests,
	}

	if err := validation.ValidateAuthModes(projectPath); err != nil {
		return err
	}

	// Get validator for project type
	validator := validation.GetProjectValidator(projectPath)
	if validator == nil {
		return errors.New("no supported project type detected (looking for package.json)")
	}

	// Run validation
	result, err := validator.Validate(ctx, projectPath, opts)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	if !result.Success {
		if result.Details != nil {
			cmdio.LogString(ctx, result.Details.Error())
		}
		return errors.New("validation failed")
	}

	cmdio.LogString(ctx, "✅ "+result.Message)
	return nil
}
