package utils

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/engine"
	"github.com/databricks/cli/ucm/config/mutator"
	"github.com/databricks/cli/ucm/config/validate"
	"github.com/databricks/cli/ucm/phases"
	"github.com/databricks/cli/ucm/statemgmt"
	"github.com/spf13/cobra"
)

// ProcessOptions controls optional behavior in ProcessUcm. Mirrors
// cmd/bundle/utils.ProcessOptions so verbs share the same opt-in switchboard.
type ProcessOptions struct {
	// If true, do not call logdiag.InitContext(); will panic if logdiag context is not initialized
	SkipInitContext bool

	// Function to call after ucm is loaded but before phases.Initialize() is called
	InitFunc func(u *ucm.Ucm)

	// If true, phases.Initialize() is not called
	SkipInitialize bool

	// If true, call PopulateLocations()
	IncludeLocations bool

	// Function to call after phases.Initialize()
	PostInitFunc func(context context.Context, u *ucm.Ucm) error

	// If true, call statemgmt.Load() to read the state and populate
	// resource IDs.
	ReadState bool

	// AlwaysPull is parameter parity with bundle.ProcessOptions; UCM's state
	// pull is always remote-fresh today, so this flag is currently a no-op.
	// Implies ReadState
	AlwaysPull bool

	// If true, calls statemgmt.Load() to read the state and update resources with IDs; also calls InitializeURLs()
	// Implies ReadState
	InitIDs bool

	// if true, pass ErrorOnEmptyState to statemgmt.Load (no-op in ucm today;
	// kept for bundle parity)
	// Implies ReadState
	ErrorOnEmptyState bool

	// If true, configure outputHandler for phases.Deploy
	Verbose bool

	// If true, call corresponding phase:
	FastValidate    bool
	Validate        bool
	Build           bool
	PreDeployChecks bool
	Deploy          bool

	// Path to pre-computed plan JSON file (direct engine only).
	// When set, skips Build and PreDeployChecks phases, loads plan from file instead of calculating.
	ReadPlanPath string

	// PostStateFunc is called at the end of ProcessUcm, within the state lifecycle scope
	// (after state is opened and IDs loaded, before deferred Finalize).
	PostStateFunc func(ctx context.Context, u *ucm.Ucm) error

	// IsPipelinesCLI is parameter parity with bundle.ProcessOptions; ucm has
	// no pipelines-CLI mode today, so this flag is currently a no-op.
	IsPipelinesCLI bool
}

// ProcessUcm loads the ucm.yml rooted at the working directory (or
// DATABRICKS_UCM_ROOT), selects the target indicated by --target, applies
// any --var overrides, resolves variables, and runs whichever phases are
// requested via opts. Mirrors cmd/bundle/utils.ProcessBundle.
//
// Errors are reported via logdiag. The caller should check
// logdiag.HasError(cmd.Context()) and render diagnostics before returning.
func ProcessUcm(cmd *cobra.Command, opts ProcessOptions) (*ucm.Ucm, error) {
	var err error
	ctx := cmd.Context()
	if opts.SkipInitContext {
		if !logdiag.IsSetup(ctx) {
			panic("SkipInitContext=true but InitContext was not called")
		}
	} else {
		ctx = logdiag.InitContext(ctx)
		cmd.SetContext(ctx)
	}

	// Load ucm config and apply target.
	u := MustConfigureUcm(cmd)

	// Log deploy telemetry on all exit paths. This is a defer to ensure
	// telemetry is logged even when the deploy command fails, for both
	// diagnostic errors and regular Go errors.
	if opts.Deploy {
		defer func() {
			if u == nil {
				return
			}
			errMsg := logdiag.GetFirstErrorSummary(ctx)
			if errMsg == "" && err != nil && !errors.Is(err, root.ErrAlreadyPrinted) {
				errMsg = err.Error()
			}
			// TODO(#100): wire real LogDeployTelemetry; this is a no-op stub.
			phases.LogDeployTelemetry(ctx, u, errMsg)
		}()
	}

	if logdiag.HasError(ctx) {
		return u, root.ErrAlreadyPrinted
	}

	variables, err := cmd.Flags().GetStringSlice("var")
	if err != nil {
		logdiag.LogDiag(ctx, diag.FromErr(err)[0])
		return u, err
	}

	// Initialize variables by assigning them values passed as command line flags
	configureVariables(ctx, u, variables)

	if u == nil || logdiag.HasError(ctx) {
		return u, root.ErrAlreadyPrinted
	}
	ctx = cmd.Context()

	if opts.InitFunc != nil {
		ucm.ApplyFuncContext(ctx, u, func(context.Context, *ucm.Ucm) { opts.InitFunc(u) })
	}

	if !opts.SkipInitialize {
		t0 := time.Now()
		// UCM's phases.Initialize requires a Backend that's only available
		// after the verb builds opts; we run only the workspace-context
		// mutators and variable resolution here, deferring state pull and
		// full Initialize to the verb's own phase calls. Sub-project C
		// (#103) will close this gap by plumbing Backend through
		// ProcessOptions so ProcessUcm can call phases.Initialize like
		// bundle does.
		// TODO(#103): wire Backend through ProcessOptions when verbs adopt
		// opts-driven orchestration (sub-project C).
		ucm.ApplySeqContext(ctx, u,
			mutator.PopulateCurrentUser(),
			mutator.DefineDefaultWorkspaceRoot(),
			mutator.ExpandWorkspaceRoot(),
		)
		if !logdiag.HasError(ctx) {
			phases.Variables(ctx, u)
		}
		u.Metrics.ExecutionTimes = append(u.Metrics.ExecutionTimes, protos.IntMapEntry{
			Key:   "phases.Initialize",
			Value: time.Since(t0).Milliseconds(),
		})
		// not checking error right away here, add locations first
	}

	if u != nil {
		// Include location information in the output if the flag is set.
		if opts.IncludeLocations {
			ucm.ApplyContext(ctx, u, mutator.PopulateLocations())
			if logdiag.HasError(ctx) {
				return u, root.ErrAlreadyPrinted
			}
		}
	}

	if logdiag.HasError(ctx) {
		return u, root.ErrAlreadyPrinted
	}

	if opts.PostInitFunc != nil {
		if perr := opts.PostInitFunc(ctx, u); perr != nil {
			return u, perr
		}
	}

	shouldReadState := opts.ReadState || opts.AlwaysPull || opts.InitIDs || opts.ErrorOnEmptyState || opts.PreDeployChecks || opts.Deploy || opts.ReadPlanPath != ""

	var stateEngine engine.EngineSetting
	if shouldReadState {
		requiredEngine, rerr := ResolveEngineSetting(ctx, &u.Config.Ucm)
		if rerr != nil {
			return u, rerr
		}
		stateEngine = requiredEngine

		// UCM's state is pulled inside phases.Initialize; there's no
		// cmd-layer PullResourcesState. The fork keeps this branch as a
		// place-holder so the bundle parallel stays visible.

		// Direct-engine state DB open is bundle-only; the corresponding
		// path in UCM is handled inside the direct-engine deploy code in
		// ucm/deploy/direct. Tracked in #95.
		if requiredEngine.Type.IsDirect() && (opts.InitIDs || opts.ErrorOnEmptyState || opts.Deploy || opts.ReadPlanPath != "" || opts.PreDeployChecks || opts.PostStateFunc != nil) {
			// TODO(#95): direct-engine path not yet wired through ProcessUcm.
			logdiag.LogError(ctx, errors.New(
				"ucm: direct engine is not yet supported via ProcessUcm; set engine: terraform or unset DATABRICKS_UCM_ENGINE"))
			return u, root.ErrAlreadyPrinted
		}

		// These are not safe in plan/deploy because they insert empty config settings for deleted resources.
		if opts.InitIDs || opts.ErrorOnEmptyState {
			ucm.ApplyFuncContext(ctx, u, func(ctx context.Context, u *ucm.Ucm) {
				for _, d := range statemgmt.Load(ctx, u) {
					logdiag.LogDiag(ctx, d)
				}
			})
			if logdiag.HasError(ctx) {
				return u, root.ErrAlreadyPrinted
			}
			// InitializeURLs makes an extra API call; only run it when URLs are needed.
			if opts.InitIDs {
				ucm.ApplyContext(ctx, u, mutator.InitializeURLs())
				if logdiag.HasError(ctx) {
					return u, root.ErrAlreadyPrinted
				}
			}
		}
	}

	if opts.ReadPlanPath != "" {
		if !stateEngine.Type.IsDirect() {
			logdiag.LogError(ctx, errors.New(`--plan is only supported with direct engine (set ucm.engine to "direct" or DATABRICKS_UCM_ENGINE=direct)`))
			return u, root.ErrAlreadyPrinted
		}
		// TODO(#95): plan-file loading and direct.ValidatePlanAgainstState
		// require the direct-engine state DB which is not wired through
		// ProcessUcm yet.
		logdiag.LogError(ctx, errors.New(
			"ucm: --plan is not yet supported via ProcessUcm"))
		return u, root.ErrAlreadyPrinted
	} else if opts.Deploy {
		opts.Build = true
		opts.PreDeployChecks = true
	}

	if opts.FastValidate {
		t1 := time.Now()
		// UCM has no FastValidate yet; validate.All is the closest equivalent
		// and runs the same set used by the Validate switch. Splitting the
		// fast/full pair lands with the validator pack work.
		validate.All(ctx, u)
		u.Metrics.ExecutionTimes = append(u.Metrics.ExecutionTimes, protos.IntMapEntry{
			Key:   "validate.FastValidate",
			Value: time.Since(t1).Milliseconds(),
		})

		if logdiag.HasError(ctx) {
			return u, root.ErrAlreadyPrinted
		}

		// Pipeline CLI only validation; ucm has no pipelines-CLI mode today.
		if opts.IsPipelinesCLI {
			rejectDefinitions(ctx, u)
			if logdiag.HasError(ctx) {
				return u, root.ErrAlreadyPrinted
			}
		}
	}

	if opts.Validate {
		validate.All(ctx, u)
		if logdiag.HasError(ctx) {
			return u, root.ErrAlreadyPrinted
		}
	}

	if opts.Build {
		t2 := time.Now()
		// TODO(#101): wire real artifact build; ucm has no artifacts today.
		// libs is the LibLocationMap bundle threads into phases.Deploy; ucm
		// has no library concept yet so we discard it.
		_ = phases.BuildArtifacts(ctx, u)
		u.Metrics.ExecutionTimes = append(u.Metrics.ExecutionTimes, protos.IntMapEntry{
			Key:   "phases.Build",
			Value: time.Since(t2).Milliseconds(),
		})

		if logdiag.HasError(ctx) {
			return u, root.ErrAlreadyPrinted
		}
	}

	if opts.PreDeployChecks {
		// UCM's PreDeployChecks today takes only (ctx, u, EngineType) —
		// the downgradeWarningToError toggle bundle exposes maps to the
		// pre-deploy validator pack and will land with #102.
		phases.PreDeployChecks(ctx, u, stateEngine.Type)

		if logdiag.HasError(ctx) {
			return u, root.ErrAlreadyPrinted
		}
	}

	if opts.Deploy {
		// UCM's phases.Deploy takes Options (Backend, factories) — the
		// foundation port (#98) doesn't have those wired through verbs yet,
		// so call with a zero-value Options and rely on the verb's existing
		// buildPhaseOptions chain (sub-project C) to supersede this in the
		// next phase of work.
		t3 := time.Now()
		phases.Deploy(ctx, u, phases.Options{})
		u.Metrics.ExecutionTimes = append(u.Metrics.ExecutionTimes, protos.IntMapEntry{
			Key:   "phases.Deploy",
			Value: time.Since(t3).Milliseconds(),
		})

		if logdiag.HasError(ctx) {
			return u, root.ErrAlreadyPrinted
		}
	}

	if opts.PostStateFunc != nil {
		if perr := opts.PostStateFunc(ctx, u); perr != nil {
			return u, perr
		}
	}

	return u, nil
}

// ResolveEngineSetting determines the effective engine setting by combining ucm config and env var.
// Priority: ucm.engine config > DATABRICKS_UCM_ENGINE env var > Default.
func ResolveEngineSetting(ctx context.Context, u *config.Ucm) (engine.EngineSetting, error) {
	var configEngine engine.EngineType
	if u != nil {
		configEngine = u.Engine
	}

	if configEngine != engine.EngineNotSet {
		source := sourceConfig
		// Try to enrich with file:line:col when we have a parent Ucm in
		// context — the dyn-tree call is best-effort and falls back to the
		// plain "config" label otherwise.
		if parent := ucm.GetOrNil(ctx); parent != nil {
			v := dyn.GetValue(parent.Config.Value(), "ucm.engine")
			if locs := v.Locations(); len(locs) > 0 {
				loc := locs[0]
				source = fmt.Sprintf("ucm.engine setting at %s:%d:%d", filepath.ToSlash(loc.File), loc.Line, loc.Column)
			}
		}
		return engine.EngineSetting{Type: configEngine, Source: source, ConfigType: configEngine}, nil
	}

	envEngine, err := engine.FromEnv(ctx)
	if err != nil {
		return engine.EngineSetting{}, err
	}
	if envEngine != engine.EngineNotSet {
		return engine.EngineSetting{Type: envEngine, Source: sourceEnv}, nil
	}

	return engine.EngineSetting{Type: engine.Default, Source: sourceDefault}, nil
}

// rejectDefinitions is the ucm-side parallel of bundle/utils.rejectDefinitions.
// UCM has no Definitions field yet, so this is a placeholder kept to mirror
// bundle's IsPipelinesCLI gate. When ucm gains a definitions concept this
// should reject open-source SDP definitions in the pipelines CLI mode.
func rejectDefinitions(_ context.Context, _ *ucm.Ucm) {
}
