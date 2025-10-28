package utils

import (
	"context"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/sync"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
)

type ProcessOptions struct {
	InitFunc          func(b *bundle.Bundle)
	PostInitFunc      func(context context.Context, b *bundle.Bundle) error
	ReadState         bool
	AlwaysPull        bool
	InitIDs           bool
	ErrorOnEmptyState bool
	IncludeLocations  bool
	Verbose           bool
	FastValidate      bool
	Validate          bool
	Build             bool
	Deploy            bool
}

func ProcessBundle(cmd *cobra.Command, opts ProcessOptions) (*bundle.Bundle, error) {
	ctx := logdiag.InitContext(cmd.Context())
	cmd.SetContext(ctx)

	b := ConfigureBundleWithVariables(cmd)
	if b == nil || logdiag.HasError(ctx) {
		return nil, root.ErrAlreadyPrinted
	}
	ctx = cmd.Context()

	if opts.InitFunc != nil {
		bundle.ApplyFuncContext(ctx, b, func(context.Context, *bundle.Bundle) { opts.InitFunc(b) })
	}

	var outputHandler sync.OutputHandler
	if opts.Verbose {
		outputHandler = func(ctx context.Context, c <-chan sync.Event) {
			sync.TextOutput(ctx, c, cmd.OutOrStdout())
		}
	}

	t0 := time.Now()
	phases.Initialize(ctx, b)
	b.Metrics.ExecutionTimes = append(b.Metrics.ExecutionTimes, protos.IntMapEntry{
		Key:   "phases.Initialize",
		Value: time.Since(t0).Milliseconds(),
	})

	if logdiag.HasError(ctx) {
		return nil, root.ErrAlreadyPrinted
	}

	if opts.PostInitFunc != nil {
		err := opts.PostInitFunc(ctx, b)
		if err != nil {
			return nil, err
		}
	}

	if opts.ReadState || opts.AlwaysPull || opts.InitIDs || opts.ErrorOnEmptyState {
		// PullResourcesState depends on stateFiler which needs b.Config.Workspace.StatePath which is set in phases.Initialize
		ctx = statemgmt.PullResourcesState(ctx, b, statemgmt.AlwaysPull(opts.AlwaysPull))
		if logdiag.HasError(ctx) {
			return nil, root.ErrAlreadyPrinted
		}
		cmd.SetContext(ctx)

		// These are not safe in plan/deploy because they insert empty config settings for deleted resources.
		if opts.InitIDs {
			var modes []statemgmt.LoadMode
			if opts.ErrorOnEmptyState {
				modes = append(modes, statemgmt.ErrorOnEmptyState)
			}
			bundle.ApplySeqContext(ctx, b,
				statemgmt.Load(modes...),
				mutator.InitializeURLs(),
			)
			if logdiag.HasError(ctx) {
				return nil, root.ErrAlreadyPrinted
			}
		}

		// Include location information in the output if the flag is set.
		if opts.IncludeLocations {
			bundle.ApplyContext(ctx, b, mutator.PopulateLocations())
			if logdiag.HasError(ctx) {
				return nil, root.ErrAlreadyPrinted
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
			return nil, root.ErrAlreadyPrinted
		}
	}

	if opts.Validate {
		validate.Validate(ctx, b)
		if logdiag.HasError(ctx) {
			return nil, root.ErrAlreadyPrinted
		}
	}

	if opts.Build {
		t2 := time.Now()
		phases.Build(ctx, b)
		b.Metrics.ExecutionTimes = append(b.Metrics.ExecutionTimes, protos.IntMapEntry{
			Key:   "phases.Build",
			Value: time.Since(t2).Milliseconds(),
		})

		if logdiag.HasError(ctx) {
			return nil, root.ErrAlreadyPrinted
		}
	}

	if opts.Deploy {
		t3 := time.Now()
		phases.Deploy(ctx, b, outputHandler)
		b.Metrics.ExecutionTimes = append(b.Metrics.ExecutionTimes, protos.IntMapEntry{
			Key:   "phases.Deploy",
			Value: time.Since(t3).Milliseconds(),
		})

		if logdiag.HasError(ctx) {
			return nil, root.ErrAlreadyPrinted
		}
	}

	return b, nil
}
