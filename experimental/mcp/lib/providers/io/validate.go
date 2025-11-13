package io

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	mcp "github.com/databricks/cli/experimental/mcp/lib"
	"github.com/databricks/cli/experimental/mcp/lib/sandbox"
	"github.com/databricks/cli/experimental/mcp/lib/sandbox/dagger"
	"github.com/databricks/cli/experimental/mcp/lib/sandbox/local"
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

	state, err := LoadState(workDir)
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
			validation = NewValidationCmd(valConfig.Command, valConfig.DockerImage)
		}
	}

	if validation == nil {
		log.Info(ctx, "using default tRPC validation strategy")
		validation = NewValidationTRPC()
	}

	validationCfg := p.config.Validation
	if validationCfg == nil {
		validationCfg = &mcp.ValidationConfig{}
		validationCfg.SetDefaults()
	} else {
		validationCfg.SetDefaults()
	}

	var sb sandbox.Sandbox
	var sandboxType string
	if validationCfg.UseDagger {
		log.Info(ctx, "attempting to create Dagger sandbox")
		daggerSb, err := p.createDaggerSandbox(ctx, workDir, validationCfg)
		if err != nil {
			log.Warnf(ctx, "failed to create Dagger sandbox, falling back to local: error=%s", err.Error())
			sb, err = p.createLocalSandbox(workDir)
			if err != nil {
				return nil, fmt.Errorf("failed to create local sandbox: %w", err)
			}
			sandboxType = "local"
		} else {
			sb = daggerSb
			sandboxType = "dagger"
		}
	} else {
		log.Info(ctx, "using local sandbox")
		sb, err = p.createLocalSandbox(workDir)
		if err != nil {
			return nil, fmt.Errorf("failed to create local sandbox: %w", err)
		}
		sandboxType = "local"
	}

	// Log which sandbox is being used for transparency
	if sandboxType == "dagger" {
		log.Info(ctx, "✓ Using Dagger sandbox for validation (containerized, isolated environment)")
	} else {
		log.Info(ctx, "Using local sandbox for validation (host filesystem)")
	}

	defer func() {
		if closeErr := sb.Close(); closeErr != nil {
			log.Warnf(ctx, "failed to close sandbox: error=%s", closeErr.Error())
		}
	}()

	result, err := validation.Validate(ctx, sb)
	if err != nil {
		return nil, fmt.Errorf("validation execution failed: %w", err)
	}

	if !result.Success {
		log.Warnf(ctx, "validation failed: message=%s", result.Message)
		return result, nil
	}

	checksum, err := ComputeChecksum(workDir)
	if err != nil {
		log.Warnf(ctx, "failed to compute checksum: error=%s", err.Error())
		return &ValidateResult{
			Success: false,
			Message: fmt.Sprintf("Validation passed but failed to compute checksum: %v", err),
		}, nil
	}

	validatedState := state.Validate(checksum)
	if err := SaveState(workDir, validatedState); err != nil {
		log.Warnf(ctx, "failed to save state: error=%s", err.Error())
		return &ValidateResult{
			Success: false,
			Message: fmt.Sprintf("Validation passed but failed to save state: %v", err),
		}, nil
	}

	log.Infof(ctx, "validation successful: checksum=%s, state=%s, sandbox_type=%s",
		checksum, string(validatedState.State), sandboxType)

	result.SandboxType = sandboxType
	return result, nil
}

func (p *Provider) createDaggerSandbox(ctx context.Context, workDir string, cfg *mcp.ValidationConfig) (sandbox.Sandbox, error) {
	log.Infof(ctx, "creating Dagger sandbox: image=%s, timeout=%d, workDir=%s",
		cfg.DockerImage, cfg.Timeout, workDir)

	sb, err := dagger.NewDaggerSandbox(ctx, dagger.Config{
		Image:          cfg.DockerImage,
		ExecuteTimeout: cfg.Timeout,
		BaseDir:        "/workspace",
	})
	if err != nil {
		log.Errorf(ctx, "failed to create Dagger sandbox: error=%s, image=%s",
			err.Error(), cfg.DockerImage)
		return nil, err
	}

	log.Debug(ctx, "propagating environment variables")
	if err := p.propagateEnvironment(sb); err != nil {
		log.Errorf(ctx, "failed to propagate environment: error=%s", err.Error())
		sb.Close()
		return nil, fmt.Errorf("failed to set environment: %w", err)
	}

	log.Debugf(ctx, "syncing files from host to container: workDir=%s", workDir)
	if err := sb.RefreshFromHost(ctx, workDir, "/workspace"); err != nil {
		log.Errorf(ctx, "failed to sync files: error=%s", err.Error())
		sb.Close()
		return nil, fmt.Errorf("failed to sync files: %w", err)
	}

	log.Info(ctx, "Dagger sandbox created successfully")
	return sb, nil
}

func (p *Provider) createLocalSandbox(workDir string) (sandbox.Sandbox, error) {
	log.Infof(p.ctx, "creating local sandbox: workDir=%s", workDir)
	return local.NewLocalSandbox(workDir)
}

func (p *Provider) propagateEnvironment(sb sandbox.Sandbox) error {
	daggerSb, ok := sb.(*dagger.DaggerSandbox)
	if !ok {
		return nil
	}

	envVars := []string{
		"DATABRICKS_HOST",
		"DATABRICKS_TOKEN",
		"DATABRICKS_WAREHOUSE_ID",
	}

	for _, key := range envVars {
		if value := os.Getenv(key); value != "" {
			daggerSb.WithEnv(key, value)
			log.Debugf(p.ctx, "propagated environment variable: key=%s", key)
		}
	}

	return nil
}
