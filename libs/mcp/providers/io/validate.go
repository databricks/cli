package io

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/appdotbuild/go-mcp/pkg/config"
	"github.com/appdotbuild/go-mcp/pkg/sandbox"
	"github.com/appdotbuild/go-mcp/pkg/sandbox/dagger"
	"github.com/appdotbuild/go-mcp/pkg/sandbox/local"
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
		p.logger.Warn("failed to load project state", slog.String("error", err.Error()))
	}
	if state == nil {
		state = NewProjectState()
	}

	p.logger.Info("starting validation",
		slog.String("work_dir", workDir),
		slog.String("state", string(state.State)))

	var validation Validation
	if p.config != nil && p.config.Validation != nil {
		valConfig := p.config.Validation
		if valConfig.Command != "" {
			p.logger.Info("using custom validation command", slog.String("command", valConfig.Command))
			validation = NewValidationCmd(valConfig.Command, valConfig.DockerImage)
		}
	}

	if validation == nil {
		p.logger.Info("using default tRPC validation strategy")
		validation = NewValidationTRPC()
	}

	validationCfg := p.config.Validation
	if validationCfg == nil {
		validationCfg = &config.ValidationConfig{}
		validationCfg.SetDefaults()
	} else {
		validationCfg.SetDefaults()
	}

	var sb sandbox.Sandbox
	var sandboxType string
	if validationCfg.UseDagger {
		p.logger.Info("attempting to create Dagger sandbox")
		daggerSb, err := p.createDaggerSandbox(ctx, workDir, validationCfg)
		if err != nil {
			p.logger.Warn("failed to create Dagger sandbox, falling back to local",
				slog.String("error", err.Error()))
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
		p.logger.Info("using local sandbox")
		sb, err = p.createLocalSandbox(workDir)
		if err != nil {
			return nil, fmt.Errorf("failed to create local sandbox: %w", err)
		}
		sandboxType = "local"
	}

	// Log which sandbox is being used for transparency
	if sandboxType == "dagger" {
		p.logger.Info("✓ Using Dagger sandbox for validation (containerized, isolated environment)")
	} else {
		p.logger.Info("Using local sandbox for validation (host filesystem)")
	}

	defer func() {
		if closeErr := sb.Close(); closeErr != nil {
			p.logger.Warn("failed to close sandbox", slog.String("error", closeErr.Error()))
		}
	}()

	result, err := validation.Validate(ctx, sb, p.logger)
	if err != nil {
		return nil, fmt.Errorf("validation execution failed: %w", err)
	}

	if !result.Success {
		p.logger.Warn("validation failed", slog.String("message", result.Message))
		return result, nil
	}

	checksum, err := ComputeChecksum(workDir)
	if err != nil {
		p.logger.Warn("failed to compute checksum", slog.String("error", err.Error()))
		return &ValidateResult{
			Success: false,
			Message: fmt.Sprintf("Validation passed but failed to compute checksum: %v", err),
		}, nil
	}

	validatedState := state.Validate(checksum)
	if err := SaveState(workDir, validatedState); err != nil {
		p.logger.Warn("failed to save state", slog.String("error", err.Error()))
		return &ValidateResult{
			Success: false,
			Message: fmt.Sprintf("Validation passed but failed to save state: %v", err),
		}, nil
	}

	p.logger.Info("validation successful",
		slog.String("checksum", checksum),
		slog.String("state", string(validatedState.State)),
		slog.String("sandbox_type", sandboxType))

	result.SandboxType = sandboxType
	return result, nil
}

func (p *Provider) createDaggerSandbox(ctx context.Context, workDir string, cfg *config.ValidationConfig) (sandbox.Sandbox, error) {
	p.logger.Info("creating Dagger sandbox",
		slog.String("image", cfg.DockerImage),
		slog.Int("timeout", cfg.Timeout),
		slog.String("workDir", workDir))

	sb, err := dagger.NewDaggerSandbox(ctx, dagger.Config{
		Image:          cfg.DockerImage,
		ExecuteTimeout: cfg.Timeout,
		BaseDir:        "/workspace",
	})
	if err != nil {
		p.logger.Error("failed to create Dagger sandbox",
			slog.String("error", err.Error()),
			slog.String("image", cfg.DockerImage))
		return nil, err
	}

	p.logger.Debug("propagating environment variables")
	if err := p.propagateEnvironment(sb); err != nil {
		p.logger.Error("failed to propagate environment", slog.String("error", err.Error()))
		sb.Close()
		return nil, fmt.Errorf("failed to set environment: %w", err)
	}

	p.logger.Debug("syncing files from host to container", slog.String("workDir", workDir))
	if err := sb.RefreshFromHost(ctx, workDir, "/workspace"); err != nil {
		p.logger.Error("failed to sync files", slog.String("error", err.Error()))
		sb.Close()
		return nil, fmt.Errorf("failed to sync files: %w", err)
	}

	p.logger.Info("Dagger sandbox created successfully")
	return sb, nil
}

func (p *Provider) createLocalSandbox(workDir string) (sandbox.Sandbox, error) {
	p.logger.Info("creating local sandbox", slog.String("workDir", workDir))
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
			p.logger.Debug("propagated environment variable", slog.String("key", key))
		}
	}

	return nil
}
