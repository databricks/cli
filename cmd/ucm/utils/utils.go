// Package utils contains helpers shared by `databricks ucm` subcommand
// implementations — primarily ProcessUcm, which mirrors the role of
// cmd/bundle/utils.ProcessBundle for the ucm verbs, and ResolveEngineSetting,
// which picks the effective deployment engine.
package utils

import (
	"context"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/engine"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/databricks/cli/ucm/phases"
	"github.com/databricks/cli/ucm/statemgmt"
	"github.com/spf13/cobra"
)

// PreMutateHook is a test-only seam. When non-nil, it runs after the Ucm is
// loaded and the target/variables are resolved but before the
// workspace-context mutators (PopulateCurrentUser, DefineDefaultWorkspaceRoot,
// ExpandWorkspaceRoot). Tests install it to pre-seed a fake CurrentUser so
// the network-backed PopulateCurrentUser mutator short-circuits and the
// downstream RootPath defaults can resolve offline.
//
// Left at nil in production; MUST NOT be set outside of tests.
var PreMutateHook func(context.Context, *ucm.Ucm)

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

	// InitIDs hydrates resource.ID from the local terraform.tfstate and runs
	// the InitializeURLs mutator. Set this for read-only consumers (e.g.
	// `ucm summary`) that need URLs to reflect deployed-vs-not-deployed
	// state. Mirrors cmd/bundle/utils.ProcessOptions.InitIDs.
	InitIDs bool
}

// ProcessUcm loads the ucm.yml rooted at the working directory (or
// DATABRICKS_UCM_ROOT), selects the target indicated by --target (or the
// default target), applies any --var overrides, resolves variables, and — if
// opts.Validate — runs the validation phase.
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

	// Apply --var before SetVariables so CLI values win over defaults/env.
	configureVariables(ctx, u, varsFromCmd(cmd))
	if logdiag.HasError(ctx) {
		return u
	}

	phases.Variables(ctx, u)
	if logdiag.HasError(ctx) {
		return u
	}

	if PreMutateHook != nil {
		PreMutateHook(ctx, u)
		if logdiag.HasError(ctx) {
			return u
		}
	}

	// Workspace context: parallel to bundle/phases.Initialize's Populate/Default/Expand
	// triple. Runs after variables (so name/target are final) and before
	// validate (so the header rendered alongside diagnostics includes User/Path).
	ucm.ApplySeqContext(ctx, u,
		mutator.PopulateCurrentUser(),
		mutator.DefineDefaultWorkspaceRoot(),
		mutator.ExpandWorkspaceRoot(),
	)
	if logdiag.HasError(ctx) {
		return u
	}

	// Hydrate resource.ID from the local tfstate and populate URL fields.
	// Missing tfstate is treated as "first run" and leaves IDs unset, so
	// InitializeURLs will leave URL empty and summary can render
	// "(not deployed)". Mirrors cmd/bundle/utils.ProcessBundle's InitIDs path.
	if opts.InitIDs {
		ucm.ApplyFuncContext(ctx, u, func(ctx context.Context, u *ucm.Ucm) {
			for _, d := range statemgmt.Load(ctx, u) {
				logdiag.LogDiag(ctx, d)
			}
		})
		if logdiag.HasError(ctx) {
			return u
		}
		ucm.ApplyContext(ctx, u, mutator.InitializeURLs())
		if logdiag.HasError(ctx) {
			return u
		}
	}

	if opts.Validate {
		phases.Validate(ctx, u)
	}
	return u
}

// varsFromCmd reads the `--var` StringSlice flag if it is wired on cmd.
// Returns nil if the flag is absent (e.g. tests that build cmds without it).
func varsFromCmd(cmd *cobra.Command) []string {
	f := cmd.Flag("var")
	if f == nil {
		return nil
	}
	vals, err := cmd.Flags().GetStringSlice("var")
	if err != nil {
		return nil
	}
	return vals
}

// configureVariables assigns .Value on each named variable via
// Config.InitializeVariables. Errors are surfaced as diagnostics.
func configureVariables(ctx context.Context, u *ucm.Ucm, vars []string) {
	if len(vars) == 0 {
		return
	}
	ucm.ApplyFuncContext(ctx, u, func(ctx context.Context, u *ucm.Ucm) {
		if err := u.Config.InitializeVariables(vars); err != nil {
			logdiag.LogDiag(ctx, diag.FromErr(err)[0])
		}
	})
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
