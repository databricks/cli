// Package utils contains helpers shared by `databricks ucm` subcommand
// implementations — primarily ProcessUcm, which mirrors the role of
// cmd/bundle/utils.ProcessBundle for the ucm verbs, and ResolveEngineSetting,
// which picks the effective deployment engine.
package utils

import (
	"context"

	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/engine"
	"github.com/databricks/cli/ucm/phases"
	"github.com/spf13/cobra"
)

const (
	sourceConfig  = "config"
	sourceEnv     = "env"
	sourceDefault = "default"
)

// ProcessOptions controls optional behavior in ProcessUcm. M0 only exposes
// the `Validate` switch; more knobs will join as additional verbs land.
type ProcessOptions struct {
	// Validate runs phases.Validate after loading the target. Set this when
	// implementing the `validate` or `policy-check` verbs.
	Validate bool
}

// ProcessUcm loads the ucm.yml rooted at the working directory (or
// DATABRICKS_UCM_ROOT), selects the target indicated by --target (or the
// default target), and — if opts.Validate — runs the validation phase.
//
// Errors are reported via logdiag. The caller should check
// logdiag.HasError(cmd.Context()) and render diagnostics before returning.
func ProcessUcm(cmd *cobra.Command, opts ProcessOptions) *ucm.Ucm {
	ctx := cmd.Context()
	if !logdiag.IsSetup(ctx) {
		ctx = logdiag.InitContext(ctx)
		cmd.SetContext(ctx)
	}

	u := ucm.MustLoad(ctx)
	if u == nil || logdiag.HasError(ctx) {
		return u
	}

	if target := getTargetFromCmd(cmd); target == "" {
		phases.LoadDefaultTarget(ctx, u)
	} else {
		phases.LoadNamedTarget(ctx, u, target)
	}
	if logdiag.HasError(ctx) {
		return u
	}

	if opts.Validate {
		phases.Validate(ctx, u)
	}
	return u
}

// ResolveEngineSetting determines the effective engine for a ucm project.
//
// Priority is: ucm.engine in config > DATABRICKS_UCM_ENGINE env var > Default.
// The returned EngineSetting always has a concrete Type (never EngineNotSet):
// callers get a ready-to-dispatch value without having to handle the unset case.
func ResolveEngineSetting(ctx context.Context, u *config.Ucm) (engine.EngineSetting, error) {
	var configEngine engine.EngineType
	if u != nil {
		configEngine = u.Engine
	}

	if configEngine != engine.EngineNotSet {
		return engine.EngineSetting{
			Type:       configEngine,
			Source:     sourceConfig,
			ConfigType: configEngine,
		}, nil
	}

	envEngine, err := engine.FromEnv(ctx)
	if err != nil {
		return engine.EngineSetting{}, err
	}
	if envEngine != engine.EngineNotSet {
		return engine.EngineSetting{
			Type:   envEngine,
			Source: sourceEnv,
		}, nil
	}

	return engine.EngineSetting{
		Type:   engine.Default,
		Source: sourceDefault,
	}, nil
}

func getTargetFromCmd(cmd *cobra.Command) string {
	if flag := cmd.Flag("target"); flag != nil {
		return flag.Value.String()
	}
	return ""
}
