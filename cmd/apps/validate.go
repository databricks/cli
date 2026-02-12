package apps

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/apps/validation"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/spf13/cobra"
)

func newValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "validate",
		Short:  "Validate a Databricks App project",
		Hidden: true,
		Long: `Validate a Databricks App project by running build, typecheck, and lint checks.

This command detects the project type and runs the appropriate validation:
- Node.js projects (package.json): runs npm install, build, typecheck, lint, and tests

After successful validation, a state file is saved that allows deployment
without re-running validation (unless code changes are detected).

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

	// Try to get bundle context for state storage
	var stateDir string
	b := root.TryConfigureBundle(cmd)
	if b != nil {
		stateDir = b.GetLocalStateDir(ctx, "apps")
	}

	opts := validation.ValidateOptions{}

	// Get validator for project type
	validator := validation.GetProjectValidator(projectPath)
	if validator != nil {
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
	} else {
		cmdio.LogString(ctx, "✅ No validator found for project type, skipping validation checks")
	}

	// Save state only if we have a bundle context
	if stateDir == "" {
		log.Debugf(ctx, "No bundle context, skipping state save")
		return nil
	}

	// Compute checksum and save state
	checksum, err := validation.ComputeChecksum(projectPath)
	if err != nil {
		return fmt.Errorf("failed to compute checksum: %w", err)
	}

	state := &validation.State{
		State:       validation.StateValidated,
		ValidatedAt: time.Now().UTC(),
		Checksum:    checksum,
	}

	if err := validation.SaveState(stateDir, state); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	cmdio.LogString(ctx, fmt.Sprintf("State saved (checksum: %s...)", checksum[:12]))
	return nil
}
