package utils

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
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

	// If true, do not read DATABRICKS_BUNDLE_ENGINE env var (for migrate command, which ignores this env var)
	SkipEngineEnvVar bool

	// If true, call corresponding phase:
	FastValidate    bool
	Validate        bool
	Build           bool
	PreDeployChecks bool
	Deploy          bool

	// Indicate whether the bundle operation originates from the pipelines CLI
	IsPipelinesCLI bool
}

func ProcessBundle(cmd *cobra.Command, opts ProcessOptions) (*bundle.Bundle, error) {
	b, _, err := ProcessBundleRet(cmd, opts)
	return b, err
}

func ProcessBundleRet(cmd *cobra.Command, opts ProcessOptions) (*bundle.Bundle, *statemgmt.StateDesc, error) {
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

	requiredEngine := engine.EngineNotSet

	if !opts.SkipEngineEnvVar {
		requiredEngine, err = engine.FromEnv(ctx)
		if err != nil {
			return nil, nil, err
		}
	}

	// Load bundle config and apply target
	b := root.MustConfigureBundle(cmd)
	if logdiag.HasError(ctx) {
		return b, nil, root.ErrAlreadyPrinted
	}

	variables, err := cmd.Flags().GetStringSlice("var")
	if err != nil {
		logdiag.LogDiag(ctx, diag.FromErr(err)[0])
		return b, nil, err
	}

	// Initialize variables by assigning them values passed as command line flags
	configureVariables(cmd, b, variables)

	if b == nil || logdiag.HasError(ctx) {
		return b, nil, root.ErrAlreadyPrinted
	}
	ctx = cmd.Context()

	if opts.InitFunc != nil {
		bundle.ApplyFuncContext(ctx, b, func(context.Context, *bundle.Bundle) { opts.InitFunc(b) })
	}

	if !opts.SkipInitialize {
		t0 := time.Now()
		phases.Initialize(ctx, b)
		b.Metrics.ExecutionTimes = append(b.Metrics.ExecutionTimes, protos.IntMapEntry{
			Key:   "phases.Initialize",
			Value: time.Since(t0).Milliseconds(),
		})
		// not checking error right away here, add locations first
	}

	if b != nil {
		// Include location information in the output if the flag is set.
		if opts.IncludeLocations {
			bundle.ApplyContext(ctx, b, mutator.PopulateLocations())
			if logdiag.HasError(ctx) {
				return b, nil, root.ErrAlreadyPrinted
			}
		}
	}

	if logdiag.HasError(ctx) {
		return b, nil, root.ErrAlreadyPrinted
	}

	if opts.PostInitFunc != nil {
		err := opts.PostInitFunc(ctx, b)
		if err != nil {
			return b, nil, err
		}
	}

	var stateDesc *statemgmt.StateDesc

	shouldReadState := opts.ReadState || opts.AlwaysPull || opts.InitIDs || opts.ErrorOnEmptyState || opts.PreDeployChecks || opts.Deploy

	if shouldReadState {
		// PullResourcesState depends on stateFiler which needs b.Config.Workspace.StatePath which is set in phases.Initialize
		ctx, stateDesc = statemgmt.PullResourcesState(ctx, b, statemgmt.AlwaysPull(opts.AlwaysPull), requiredEngine)
		if logdiag.HasError(ctx) {
			return b, stateDesc, root.ErrAlreadyPrinted
		}
		cmd.SetContext(ctx)

		// These are not safe in plan/deploy because they insert empty config settings for deleted resources.
		if opts.InitIDs || opts.ErrorOnEmptyState {
			var modes []statemgmt.LoadMode
			if opts.ErrorOnEmptyState {
				modes = append(modes, statemgmt.ErrorOnEmptyState)
			}
			bundle.ApplySeqContext(ctx, b,
				statemgmt.Load(stateDesc.Engine, modes...),
				mutator.InitializeURLs(),
			)
			if logdiag.HasError(ctx) {
				return b, stateDesc, root.ErrAlreadyPrinted
			}
		}
	}

	if opts.FastValidate {
		t1 := time.Now()
		bundle.ApplyContext(ctx, b, validate.FastValidate())
		b.Metrics.ExecutionTimes = append(b.Metrics.ExecutionTimes, protos.IntMapEntry{
			Key:   "validate.FastValidate",
			Value: time.Since(t1).Milliseconds(),
		})

		if logdiag.HasError(ctx) {
			return b, stateDesc, root.ErrAlreadyPrinted
		}

		// Pipeline CLI only validation.
		if opts.IsPipelinesCLI {
			rejectDefinitions(ctx, b)
			if logdiag.HasError(ctx) {
				return b, stateDesc, root.ErrAlreadyPrinted
			}
		}
	}

	if opts.Validate {
		validate.Validate(ctx, b)
		if logdiag.HasError(ctx) {
			return b, stateDesc, root.ErrAlreadyPrinted
		}
	}

	var libs phases.LibLocationMap

	if opts.Build || opts.Deploy {
		t2 := time.Now()
		libs = phases.Build(ctx, b)
		b.Metrics.ExecutionTimes = append(b.Metrics.ExecutionTimes, protos.IntMapEntry{
			Key:   "phases.Build",
			Value: time.Since(t2).Milliseconds(),
		})

		if logdiag.HasError(ctx) {
			return b, stateDesc, root.ErrAlreadyPrinted
		}
	}

	if opts.PreDeployChecks || opts.Deploy {
		downgradeWarningToError := !opts.Deploy
		phases.PreDeployChecks(ctx, b, downgradeWarningToError, stateDesc.Engine)

		if logdiag.HasError(ctx) {
			return b, stateDesc, root.ErrAlreadyPrinted
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
		phases.Deploy(ctx, b, outputHandler, stateDesc.Engine, libs)
		b.Metrics.ExecutionTimes = append(b.Metrics.ExecutionTimes, protos.IntMapEntry{
			Key:   "phases.Deploy",
			Value: time.Since(t3).Milliseconds(),
		})

		if logdiag.HasError(ctx) {
			return b, stateDesc, root.ErrAlreadyPrinted
		}

		if b != nil && stateDesc != nil && stateDesc.Engine.IsDirect() && stateDesc.HasRemoteTerraformState() {
			statemgmt.BackupRemoteTerraformState(ctx, b)

			if logdiag.HasError(ctx) {
				return b, stateDesc, root.ErrAlreadyPrinted
			}
		}
	}

	return b, stateDesc, nil
}

func rejectDefinitions(ctx context.Context, b *bundle.Bundle) {
	if b.Config.Definitions != nil {
		v := dyn.GetValue(b.Config.Value(), "definitions")
		loc := v.Locations()
		filename := "input yaml"
		if len(loc) > 0 {
			filename = filepath.ToSlash(loc[0].File)
		}
		logdiag.LogError(ctx, errors.New(filename+` seems to be formatted for open-source Spark Declarative Pipelines.
Pipelines CLI currently only supports Lakeflow Declarative Pipelines development.
To see an example of a supported pipelines template, create a new Pipelines CLI project with "pipelines init".`))
	}
}
