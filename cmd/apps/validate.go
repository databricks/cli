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
		Use:    "validate",
		Short:  "Validate a Databricks App project",
		Hidden: true,
		Long: `Validate a Databricks App project by running build, typecheck, and lint checks.

This command detects the project type and runs the appropriate validation:
- Node.js projects (package.json): runs npm install, build, typecheck, and lint

Examples:
  # Validate the current directory
  databricks apps validate

  # Validate a specific directory
  databricks apps validate --path ./my-app`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(cmd)
		},
	}

	cmd.Flags().String("path", "", "Path to the project directory (defaults to current directory)")

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

	// Get validator for project type
	validator := validation.GetProjectValidator(projectPath)
	if validator == nil {
		return errors.New("no supported project type detected (looking for package.json)")
	}

	// Run validation
	result, err := validator.Validate(ctx, projectPath)
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
