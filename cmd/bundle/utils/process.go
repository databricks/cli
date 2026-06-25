package utils

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/sync"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
)

type ProcessOptions struct {
	// If true, do not call logdiag.InitContext(); will panic if logdiag context is not initialized
	SkipInitContext bool

	// Function to call after bundle is loaded but before phases.Initialize() is called
	InitFunc func(b *bundle.Bundle)

	// If true, phases.Initialize() is not called
	SkipInitialize bool

	// If true, call PopulateLocations()
	IncludeLocations bool

	// Function to call after phases.Initialize()
	PostInitFunc func(context context.Context, b *bundle.Bundle) error

	// If true, call PullResourcesState() to read state
	ReadState bool

	// AlwaysPull parameter to PullResourcesState()
	// Implies ReadState
	AlwaysPull bool

	// If true, calls statemgmt.Load() to read the state and update resources with IDs; also calls InitializeURLs()
	// Implies ReadState
	InitIDs bool

	// if true, pass ErrorOnEmptyState to statemgmt.Load
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

	// PostStateFunc is called at the end of ProcessBundleRet, within the state lifecycle scope
	// (after state is opened and IDs loaded, before deferred Finalize).
	PostStateFunc func(ctx context.Context, b *bundle.Bundle, stateDesc *statemgmt.StateDesc) error

	// Indicate whether the bundle operation originates from the pipelines CLI
	IsPipelinesCLI bool
}

func ProcessBundle(cmd *cobra.Command, opts ProcessOptions) (*bundle.Bundle, error) {
	b, _, err := ProcessBundleRet(cmd, opts)
	return b, err
}

func ProcessBundleRet(cmd *cobra.Command, opts ProcessOptions) (b *bundle.Bundle, stateDesc *statemgmt.StateDesc, retErr error) {
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

	// Load bundle config and apply target
	b, cfgErr := root.MustConfigureBundle(cmd)

	// Log deploy telemetry on all exit paths. This is a defer to ensure
	// telemetry is logged even when the deploy command fails, for both
	// diagnostic errors and regular Go errors.
	if opts.Deploy {
		defer func() {
			if b == nil {
				return
			}
			// Prefer the first logged error summary: when a mutator renders its
			// errors and returns the opaque ErrAlreadyPrinted sentinel, retErr no
			// longer carries the original message.
			errMsg := logdiag.GetFirstErrorSummary(ctx)
			if errMsg == "" && retErr != nil && !errors.Is(retErr, root.ErrAlreadyPrinted) {
				errMsg = retErr.Error()
			}
			phases.LogDeployTelemetry(ctx, b, errMsg)
		}()
	}

	if cfgErr != nil {
		return b, nil, root.RenderAndReturnError(ctx, cfgErr)
	}

	variables, err := cmd.Flags().GetStringSlice("var")
	if err != nil {
		return b, nil, err
	}

	// Initialize variables by assigning them values passed as command line flags
	if err := configureVariables(cmd, b, variables); err != nil {
		return b, nil, root.RenderAndReturnError(ctx, err)
	}

	if b == nil {
		return b, nil, root.ErrAlreadyPrinted
	}
	ctx = cmd.Context()

	if opts.InitFunc != nil {
		if err := bundle.ApplyFuncContext(ctx, b, func(context.Context, *bundle.Bundle) { opts.InitFunc(b) }); err != nil {
			return b, nil, root.RenderAndReturnError(ctx, err)
		}
	}

	var initErr error
	if !opts.SkipInitialize {
		t0 := time.Now()
		initErr = phases.Initialize(ctx, b)
		b.Metrics.ExecutionTimes = append(b.Metrics.ExecutionTimes, protos.IntMapEntry{
			Key:   "phases.Initialize",
			Value: time.Since(t0).Milliseconds(),
		})
		// not returning the error right away here, add locations first
	}

	if b != nil {
		// Include location information in the output if the flag is set.
		if opts.IncludeLocations {
			if err := bundle.ApplyContext(ctx, b, mutator.PopulateLocations()); err != nil {
				return b, nil, root.RenderAndReturnError(ctx, err)
			}
		}
	}

	if initErr != nil {
		return b, nil, root.RenderAndReturnError(ctx, initErr)
	}

	if opts.PostInitFunc != nil {
		err := opts.PostInitFunc(ctx, b)
		if err != nil {
			return b, nil, err
		}
	}

	shouldReadState := opts.ReadState || opts.AlwaysPull || opts.InitIDs || opts.ErrorOnEmptyState || opts.PreDeployChecks || opts.Deploy || opts.ReadPlanPath != ""

	if shouldReadState {
		requiredEngine, err := ResolveEngineSetting(ctx, b)
		if err != nil {
			return b, nil, err
		}

		// PullResourcesState depends on stateFiler which needs b.Config.Workspace.StatePath which is set in phases.Initialize
		var pullErr error
		ctx, stateDesc, pullErr = statemgmt.PullResourcesState(ctx, b, statemgmt.AlwaysPull(opts.AlwaysPull), requiredEngine)
		if pullErr != nil {
			return b, stateDesc, root.RenderAndReturnError(ctx, pullErr)
		}
		cmd.SetContext(ctx)

		// --select is only supported by the direct engine, which tracks resource
		// dependencies in the plan graph (used to expand the selection transitively).
		// The engine is only known for certain after the state is pulled, so reject it
		// here rather than silently planning/deploying every resource on terraform.
		if len(b.Select) > 0 && !stateDesc.Engine.IsDirect() {
			return b, stateDesc, root.RenderAndReturnError(ctx, errors.New("--select is only supported with the direct engine. See https://docs.databricks.com/aws/en/dev-tools/bundles/direct"))
		}

		// Open direct engine state once for all subsequent operations (ExportState, CalculatePlan, Apply, etc.)
		needDirectState := stateDesc.Engine.IsDirect() && (opts.InitIDs || opts.ErrorOnEmptyState || opts.Deploy || opts.ReadPlanPath != "" || opts.PreDeployChecks || opts.PostStateFunc != nil)
		if needDirectState {
			_, localPath := b.StateFilenameDirect(ctx)
			if err := b.DeploymentBundle.StateDB.Open(ctx, localPath, dstate.WithRecovery(true), dstate.WithWrite(false)); err != nil {
				return b, stateDesc, root.RenderAndReturnError(ctx, err)
			}
		}

		// These are not safe in plan/deploy because they insert empty config settings for deleted resources.
		if opts.InitIDs || opts.ErrorOnEmptyState {
			var modes []statemgmt.LoadMode
			if opts.ErrorOnEmptyState {
				modes = append(modes, statemgmt.ErrorOnEmptyState)
			}
			var state statemgmt.ExportedResourcesMap
			if stateDesc.Engine.IsDirect() {
				state = b.DeploymentBundle.ExportState(ctx)
			} else {
				var err error
				state, err = terraform.ParseResourcesState(ctx, b)
				if err != nil {
					return b, stateDesc, root.RenderAndReturnError(ctx, err)
				}
			}
			mutators := []bundle.Mutator{
				statemgmt.Load(state, modes...),
			}
			// InitializeURLs makes an extra API call; only run it when URLs are needed.
			if opts.InitIDs {
				mutators = append(mutators, mutator.InitializeURLs())
			}
			if err := bundle.ApplySeqContext(ctx, b, mutators...); err != nil {
				return b, stateDesc, root.RenderAndReturnError(ctx, err)
			}
		}
	}

	var plan *deployplan.Plan

	if opts.ReadPlanPath != "" {
		if !stateDesc.Engine.IsDirect() {
			return b, stateDesc, root.RenderAndReturnError(ctx, errors.New("--plan is only supported with direct engine (set bundle.engine to \"direct\" or DATABRICKS_BUNDLE_ENGINE=direct)"))
		}
		opts.Build = false
		opts.PreDeployChecks = false

		var err error
		plan, err = deployplan.LoadPlanFromFile(opts.ReadPlanPath)
		if err != nil {
			return b, stateDesc, root.RenderAndReturnError(ctx, err)
		}
		currentVersion := build.GetInfo().Version
		if plan.CLIVersion != currentVersion {
			log.Warnf(ctx, "Plan was created with CLI version %s but current version is %s", plan.CLIVersion, currentVersion)
		}

		// Validate that the plan's lineage and serial match the current state
		// This must happen before any file operations
		err = direct.ValidatePlanAgainstState(&b.DeploymentBundle.StateDB, plan)
		if err != nil {
			return b, stateDesc, root.RenderAndReturnError(ctx, err)
		}
	} else if opts.Deploy {
		opts.Build = true
		opts.PreDeployChecks = true
	}

	if opts.FastValidate {
		t1 := time.Now()
		fastValidateErr := bundle.ApplyContext(ctx, b, validate.FastValidate())
		b.Metrics.ExecutionTimes = append(b.Metrics.ExecutionTimes, protos.IntMapEntry{
			Key:   "validate.FastValidate",
			Value: time.Since(t1).Milliseconds(),
		})

		if fastValidateErr != nil {
			return b, stateDesc, root.RenderAndReturnError(ctx, fastValidateErr)
		}

		// Pipeline CLI only validation.
		if opts.IsPipelinesCLI {
			if err := rejectDefinitions(ctx, b); err != nil {
				return b, stateDesc, root.RenderAndReturnError(ctx, err)
			}
		}
	}

	if opts.Validate {
		if err := validate.Validate(ctx, b); err != nil {
			return b, stateDesc, root.RenderAndReturnError(ctx, err)
		}
	}

	var libs phases.LibLocationMap

	if opts.Build {
		t2 := time.Now()
		var buildErr error
		libs, buildErr = phases.Build(ctx, b)
		b.Metrics.ExecutionTimes = append(b.Metrics.ExecutionTimes, protos.IntMapEntry{
			Key:   "phases.Build",
			Value: time.Since(t2).Milliseconds(),
		})

		if buildErr != nil {
			return b, stateDesc, root.RenderAndReturnError(ctx, buildErr)
		}
	}

	if opts.PreDeployChecks {
		downgradeWarningToError := !opts.Deploy
		if err := phases.PreDeployChecks(ctx, b, downgradeWarningToError, stateDesc.Engine); err != nil {
			return b, stateDesc, root.RenderAndReturnError(ctx, err)
		}
	}

	if opts.Deploy {
		var outputHandler sync.OutputHandler
		if opts.Verbose {
			outputHandler = func(ctx context.Context, c <-chan sync.Event) {
				sync.TextOutput(ctx, c, cmd.OutOrStdout())
			}
		}

		t3 := time.Now()
		deployErr := phases.Deploy(ctx, b, outputHandler, stateDesc.Engine, libs, plan)
		b.Metrics.ExecutionTimes = append(b.Metrics.ExecutionTimes, protos.IntMapEntry{
			Key:   "phases.Deploy",
			Value: time.Since(t3).Milliseconds(),
		})

		if deployErr != nil {
			return b, stateDesc, root.RenderAndReturnError(ctx, deployErr)
		}

		if b != nil && stateDesc != nil && stateDesc.Engine.IsDirect() && stateDesc.HasRemoteTerraformState() {
			if err := statemgmt.BackupRemoteTerraformState(ctx, b); err != nil {
				return b, stateDesc, root.RenderAndReturnError(ctx, err)
			}
		}
	}

	if opts.PostStateFunc != nil {
		if err := opts.PostStateFunc(ctx, b, stateDesc); err != nil {
			return b, stateDesc, err
		}
	}

	return b, stateDesc, nil
}

// ResolveEngineSetting determines the effective engine setting by combining bundle config and env var.
// Priority: bundle.engine config > DATABRICKS_BUNDLE_ENGINE env var.
func ResolveEngineSetting(ctx context.Context, b *bundle.Bundle) (engine.EngineSetting, error) {
	configEngine := b.Config.Bundle.Engine

	if configEngine != engine.EngineNotSet {
		source := "bundle.engine setting"
		v := dyn.GetValue(b.Config.Value(), "bundle.engine")
		if locs := v.Locations(); len(locs) > 0 {
			loc := locs[0]
			source = fmt.Sprintf("bundle.engine setting at %s:%d:%d", filepath.ToSlash(loc.File), loc.Line, loc.Column)
		}
		return engine.EngineSetting{Type: configEngine, Source: source, ConfigType: configEngine}, nil
	}

	envEngine, err := engine.FromEnv(ctx)
	if err != nil {
		return engine.EngineSetting{}, err
	}
	if envEngine != engine.EngineNotSet {
		return engine.EngineSetting{Type: envEngine, Source: engine.EnvVar + " environment variable"}, nil
	}

	return engine.EngineSetting{}, nil
}

func rejectDefinitions(ctx context.Context, b *bundle.Bundle) error {
	if b.Config.Definitions != nil {
		v := dyn.GetValue(b.Config.Value(), "definitions")
		loc := v.Locations()
		filename := "input yaml"
		if len(loc) > 0 {
			filename = filepath.ToSlash(loc[0].File)
		}
		return errors.New(filename + ` seems to be formatted for open-source Spark Declarative Pipelines.
Pipelines CLI currently only supports Lakeflow Spark Declarative Pipelines development.
To see an example of a supported pipelines template, create a new Pipelines CLI project with "pipelines init".`)
	}
	return nil
}
