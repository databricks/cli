package mcp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/experimental/aitools/lib/state"
	"github.com/databricks/cli/experimental/aitools/lib/validation"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

// getValidator returns the appropriate validator based on the validator type.
func getValidator(validatorType, customCommand string) (validation.Validation, error) {
	switch validatorType {
	case "nodejs":
		return &validation.ValidationNodeJs{}, nil
	case "custom":
		if customCommand == "" {
			return nil, errors.New("--custom-command is required when --validator=custom")
		}
		return &validation.ValidationCmd{Command: customCommand}, nil
	default:
		return nil, fmt.Errorf("unknown validator: %s (available: nodejs, custom)", validatorType)
	}
}

func newValidateCmd() *cobra.Command {
	var (
		validatorType string
		customCommand string
	)

	cmd := &cobra.Command{
		Use:   "validate <work-dir>",
		Short: "Validate a Databricks app project",
		Long: `Validate a Databricks app project by running build, type checks, tests, etc.

Supports multiple validation strategies for different project types.

Exit codes:
  0 - Validation succeeded
  1 - Validation failed
  2 - Invalid flags or configuration`,
		Example: `  databricks experimental aitools tools validate ./my-project
  databricks experimental aitools tools validate ./my-project --validator=custom --custom-command="./validate.sh"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			workDir := args[0]

			// Validate directory exists
			absPath, err := filepath.Abs(workDir)
			if err != nil {
				return fmt.Errorf("invalid work directory path: %w", err)
			}

			if _, err := os.Stat(absPath); os.IsNotExist(err) {
				return fmt.Errorf("work directory does not exist: %s", absPath)
			}

			// Get validator
			validator, err := getValidator(validatorType, customCommand)
			if err != nil {
				return err
			}

			// Run validation
			result, err := validator.Validate(ctx, absPath)
			if err != nil {
				return fmt.Errorf("validation error: %w", err)
			}

			// Output result
			cmdio.LogString(ctx, result.String())

			// Return appropriate exit code
			if !result.Success {
				return errors.New("validation failed")
			}

			// Compute checksum and transition to validated state
			checksum, err := state.ComputeChecksum(absPath)
			if err != nil {
				return fmt.Errorf("failed to compute checksum: %w", err)
			}

			// Load current state or create new scaffolded state
			currentState, err := state.LoadState(absPath)
			if err != nil {
				return fmt.Errorf("failed to load state: %w", err)
			}
			if currentState == nil {
				currentState = state.NewScaffolded()
			}

			// Transition to validated
			newState := currentState.Validate(checksum)
			if err := state.SaveState(absPath, newState); err != nil {
				return fmt.Errorf("failed to save state: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&validatorType, "validator", "nodejs",
		"Validator to use: nodejs or custom")
	cmd.Flags().StringVar(&customCommand, "custom-command", "",
		"Custom command to run (for validator=custom)")

	return cmd
}
