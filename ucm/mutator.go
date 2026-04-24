package ucm

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/telemetry/protos"
)

// Mutator is a function-object that transforms a Ucm's configuration tree or
// runtime state. Mirrors bundle.Mutator so converting code between the two
// stays mechanical.
type Mutator interface {
	// Name returns the mutator's display name.
	Name() string

	// Apply runs the mutation against u.
	Apply(context.Context, *Ucm) diag.Diagnostics
}

// safeMutatorName returns the package name and type name of the underlying mutator type
// in the format "package_name.(type_name)".
//
// We cannot rely on the .Name() method for the name here because it can contain user
// input which would then leak into the telemetry. E.g. [SetDefault] or [loader.ProcessInclude]
func safeMutatorName(m Mutator) string {
	t := reflect.TypeOf(m)

	// Handle pointer types by getting the element type
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Get the package path and type name
	pkgPath := t.PkgPath()
	typeName := t.Name()

	// Extract just the package name from the full package path
	// e.g., "github.com/databricks/cli/bundle/config/mutator" -> "mutator"
	packageName := pkgPath
	if lastSlash := len(pkgPath) - 1; lastSlash >= 0 {
		for i := lastSlash; i >= 0; i-- {
			if pkgPath[i] == '/' {
				packageName = pkgPath[i+1:]
				break
			}
		}
	}

	return packageName + ".(" + typeName + ")"
}

// ApplyContext runs a single mutator, keeping the dynamic/typed trees in sync
// and forwarding diagnostics to logdiag.
func ApplyContext(ctx context.Context, u *Ucm, m Mutator) {
	t0 := time.Now()
	defer func() {
		duration := time.Since(t0).Milliseconds()

		// Don't track the mutator if it takes less than 1ms to execute.
		if duration == 0 {
			return
		}

		u.Metrics.ExecutionTimes = append(u.Metrics.ExecutionTimes, protos.IntMapEntry{
			Key:   safeMutatorName(m),
			Value: duration,
		})
	}()

	ctx = log.NewContext(ctx, log.GetLogger(ctx).With("mutator", m.Name()))
	log.Debugf(ctx, "Apply")

	if err := u.Config.MarkMutatorEntry(ctx); err != nil {
		logdiag.LogError(ctx, fmt.Errorf("entry error: %w", err))
		return
	}
	defer func() {
		if err := u.Config.MarkMutatorExit(ctx); err != nil {
			logdiag.LogError(ctx, fmt.Errorf("exit error: %w", err))
		}
	}()

	for _, d := range m.Apply(ctx, u) {
		logdiag.LogDiag(ctx, d)
	}
}

// ApplySeqContext runs mutators in order, stopping at the first one that logs
// an error.
func ApplySeqContext(ctx context.Context, u *Ucm, mutators ...Mutator) {
	for _, m := range mutators {
		ApplyContext(ctx, u, m)
		if logdiag.HasError(ctx) {
			break
		}
	}
}

type funcMutator struct {
	fn func(context.Context, *Ucm)
}

func (m funcMutator) Name() string { return "<func>" }

func (m funcMutator) Apply(ctx context.Context, u *Ucm) diag.Diagnostics {
	m.fn(ctx, u)
	return nil
}

// ApplyFuncContext applies an inline-specified function mutator.
func ApplyFuncContext(ctx context.Context, u *Ucm, fn func(context.Context, *Ucm)) {
	ApplyContext(ctx, u, funcMutator{fn})
}

// Apply runs a single mutator in a test-friendly way: it ensures logdiag is
// set up, collects diagnostics into a slice, and returns them.
func Apply(ctx context.Context, u *Ucm, m Mutator) diag.Diagnostics {
	if !logdiag.IsSetup(ctx) {
		ctx = logdiag.InitContext(ctx)
	}
	previous := logdiag.FlushCollected(ctx)
	if len(previous) > 0 {
		panic(fmt.Sprintf("Already have %d diags collected: %v", len(previous), previous))
	}
	logdiag.SetCollect(ctx, true)
	ApplyContext(ctx, u, m)
	return logdiag.FlushCollected(ctx)
}

// ApplySeq is the test-helper equivalent of ApplySeqContext.
func ApplySeq(ctx context.Context, u *Ucm, mutators ...Mutator) diag.Diagnostics {
	if !logdiag.IsSetup(ctx) {
		ctx = logdiag.InitContext(ctx)
	}
	previous := logdiag.FlushCollected(ctx)
	if len(previous) > 0 {
		panic(fmt.Sprintf("Already have %d diags collected: %v", len(previous), previous))
	}
	logdiag.SetCollect(ctx, true)
	ApplySeqContext(ctx, u, mutators...)
	return logdiag.FlushCollected(ctx)
}
