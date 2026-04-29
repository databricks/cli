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

	// BuildPhaseOptions, when non-nil, supplies the phases.Options ProcessUcm
	// passes to phases.Deploy (and that verbs that drive their own phase
	// calls retrieve via utils.BuildPhaseOptionsHook). When nil, ProcessUcm
	// falls back to BuildPhaseOptionsHook (which defaults to
	// DefaultBuildPhaseOptions and is the seam tests overwrite to inject a
	// fake Backend + factories). Verbs almost always leave this nil — set it
	// only if a single call needs a non-default Options without disturbing
	// the package-level hook.
	BuildPhaseOptions func(ctx context.Context, u *ucm.Ucm) (phases.Options, error)

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
//
// When opts.Deploy is set, ProcessUcm defers a call to
// phases.LogDeployTelemetry on every exit path, including error returns.
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
	if TestProcessHook != nil {
		TestProcessHook(ctx, u)
	}

	if !opts.SkipInitialize {
		t0 := time.Now()
		// Workspace-context mutators + variables run unconditionally; the
		// state pull side of bundle's phases.Initialize is performed below
		// in shouldReadState, gated by opts so that read-only verbs don't
		// pay for a state pull they don't need. Verbs that do need Options
		// (Deploy, Bind, Unbind, Plan, Destroy, Drift, Import) set
		// BuildPhaseOptions on ProcessOptions or rely on
		// BuildPhaseOptionsHook so ProcessUcm can supply real Backend +
		// factories.
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

		// State pull happens inside phases.Plan/Deploy/Destroy, not at the
		// cmd layer. The bundle parallel runs PullResourcesState here;
		// ucm's verbs drive their own phase calls which Initialize and
		// Pull internally. Direct-engine state DB open is handled inside
		// ucm/deploy/direct (tracked in #95) — ProcessUcm no longer gates
		// on engine type, so direct flows through naturally and any
		// not-yet-wired code path errors at its own boundary.

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
			logdiag.LogError(ctx, errors.New(`ucm: --plan is only supported with direct engine (set ucm.engine to "direct" or DATABRICKS_UCM_ENGINE=direct)`))
			return u, root.ErrAlreadyPrinted
		}
		// TODO(#95): plan-file loading + direct.ValidatePlanAgainstState
		// hooks into the direct-engine state DB which is not wired through
		// ProcessUcm yet. ProcessUcm no longer gates on this; the not-yet-
		// wired code path errors at its own boundary inside phases.Deploy.
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
		buildOpts := opts.BuildPhaseOptions
		if buildOpts == nil {
			buildOpts = BuildPhaseOptionsHook
		}
		phaseOpts, perr := buildOpts(ctx, u)
		if perr != nil {
			return u, fmt.Errorf("resolve deploy options: %w", perr)
		}
		phaseOpts.ForceLock = u.ForceLock
		phaseOpts.AutoApprove = u.AutoApprove

		t3 := time.Now()
		phases.Deploy(ctx, u, phaseOpts)
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

// ResolveEngineSetting returns the effective engine for a ucm project.
// Priority: ucm.engine config > DATABRICKS_UCM_ENGINE env var > Default.
// Source labels use short forms ("config"/"env"/"default") rather than
// bundle's longer descriptions, matching ucm's existing test expectations.
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

// rejectDefinitions parallels bundle's Pipelines-CLI guard. UCM has no
// Pipelines CLI mode today; reachable only via IsPipelinesCLI which no verb
// sets. Retained for fork-shape parity with cmd/bundle/utils/process.go.
func rejectDefinitions(_ context.Context, _ *ucm.Ucm) {
}
