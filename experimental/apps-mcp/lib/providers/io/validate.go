package io

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/log"
)

type ValidateArgs struct {
	WorkDir string `json:"work_dir"`
}

func (p *Provider) Validate(ctx context.Context, args *ValidateArgs) (*ValidateResult, error) {
	workDir, err := filepath.Abs(args.WorkDir)
	if err != nil {
		return nil, fmt.Errorf("invalid work directory: %w", err)
	}

	if !filepath.IsAbs(workDir) {
		return nil, errors.New("work_dir must be an absolute path")
	}

	if _, err := os.Stat(workDir); os.IsNotExist(err) {
		return nil, errors.New("work directory does not exist")
	}

	state, err := LoadState(ctx, workDir)
	if err != nil {
		log.Warnf(ctx, "failed to load project state: error=%v", err)
	}
	if state == nil {
		state = NewProjectState()
	}

	log.Infof(ctx, "starting validation: work_dir=%s, state=%s", workDir, string(state.State))

	var validation Validation
	if p.config != nil && p.config.Validation != nil {
		valConfig := p.config.Validation
		if valConfig.Command != "" {
			log.Infof(ctx, "using custom validation command: command=%s", valConfig.Command)
			validation = NewValidationCmd(valConfig.Command)
		}
	}

	if validation == nil {
		log.Info(ctx, "using default Node.js validation strategy")
		validation = NewValidationNodeJs()
	}

	result, err := validation.Validate(ctx, workDir)
	if err != nil {
		return nil, fmt.Errorf("validation execution failed: %w", err)
	}

	if !result.Success {
		log.Warnf(ctx, "validation failed: message=%s", result.Message)
		return result, nil
	}

	checksum, err := ComputeChecksum(ctx, workDir)
	if err != nil {
		log.Warnf(ctx, "failed to compute checksum: error=%s", err.Error())
		return &ValidateResult{
			Success: false,
			Message: fmt.Sprintf("Validation passed but failed to compute checksum: %v", err),
		}, nil
	}

	validatedState := state.Validate(checksum)
	if err := SaveState(ctx, workDir, validatedState); err != nil {
		log.Warnf(ctx, "failed to save state: error=%s", err.Error())
		return &ValidateResult{
			Success: false,
			Message: fmt.Sprintf("Validation passed but failed to save state: %v", err),
		}, nil
	}

	log.Infof(ctx, "validation successful: checksum=%s, state=%s",
		checksum, string(validatedState.State))

	return result, nil
}
