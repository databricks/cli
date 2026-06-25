package bundle

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry/protos"

	"github.com/databricks/cli/libs/logdiag"
)

// Mutator is the interface type that mutates a bundle's configuration or internal state.
// This makes every mutation or action observable and debuggable.
type Mutator interface {
	// Name returns the mutators name.
	Name() string

	// Apply mutates the specified bundle object. It returns an error to abort
	// the pipeline; warnings and recommendations are emitted via logdiag.LogDiag.
	Apply(context.Context, *Bundle) error
}

// safeMutatorName returns the package name and type name of the underlying mutator type
// in the format "package_name.(type_name)".
//
// We cannot rely on the .Name() method for the name here because it can contain user
// input which would then leak into the telemetry. E.g. [SetDefault] or [loader.ProcessInclude]
func safeMutatorName(m Mutator) string {
	t := reflect.TypeOf(m)

	// Handle pointer types by getting the element type
	if t.Kind() == reflect.Pointer {
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

// ApplyContext applies a single mutator. It returns the error produced by the
// mutator (if any); warnings and recommendations are emitted directly via
// logdiag.LogDiag and are not part of the returned error.
func ApplyContext(ctx context.Context, b *Bundle, m Mutator) (err error) {
	t0 := time.Now()
	defer func() {
		duration := time.Since(t0).Milliseconds()

		// Don't track the mutator if it takes less than 1ms to execute.
		if duration == 0 {
			return
		}

		b.Metrics.ExecutionTimes = append(b.Metrics.ExecutionTimes, protos.IntMapEntry{
			Key:   safeMutatorName(m),
			Value: duration,
		})
	}()

	ctx = log.NewContext(ctx, log.GetLogger(ctx).With("mutator", m.Name()))

	log.Debugf(ctx, "Apply")

	if entryErr := b.Config.MarkMutatorEntry(ctx); entryErr != nil {
		return fmt.Errorf("entry error: %w", entryErr)
	}

	defer func() {
		if exitErr := b.Config.MarkMutatorExit(ctx); exitErr != nil && err == nil {
			err = fmt.Errorf("exit error: %w", exitErr)
		}
	}()

	return m.Apply(ctx, b)
}

// ApplySeqContext applies mutators in order, stopping at the first error.
func ApplySeqContext(ctx context.Context, b *Bundle, mutators ...Mutator) error {
	for _, m := range mutators {
		if err := ApplyContext(ctx, b, m); err != nil {
			return err
		}
	}
	return nil
}

type funcMutator struct {
	fn func(context.Context, *Bundle)
}

func (m funcMutator) Name() string {
	return "<func>"
}

func (m funcMutator) Apply(ctx context.Context, b *Bundle) error {
	m.fn(ctx, b)
	return nil
}

// ApplyFuncContext applies an inline-specified function mutator.
func ApplyFuncContext(ctx context.Context, b *Bundle, fn func(context.Context, *Bundle)) error {
	return ApplyContext(ctx, b, funcMutator{fn})
}

// Test helpers. TODO: move to separate package.

// Apply is a test helper that runs a mutator and returns the diagnostics it
// produced: warnings/recommendations collected via logdiag plus the returned
// error (if any) as a trailing error diagnostic.
func Apply(ctx context.Context, b *Bundle, m Mutator) diag.Diagnostics {
	if !logdiag.IsSetup(ctx) {
		ctx = logdiag.InitContext(ctx)
	}
	previous := logdiag.FlushCollected(ctx)
	if len(previous) > 0 {
		panic(fmt.Sprintf("Already have %d diags collected: %v", len(previous), previous))
	}
	logdiag.SetCollect(ctx, true)
	err := ApplyContext(ctx, b, m)
	diags := logdiag.FlushCollected(ctx)
	// A mutator that renders its own diagnostics returns the ErrAlreadyPrinted
	// sentinel; those diagnostics are already in Collected, so only append errors
	// that were returned directly (not yet rendered).
	if err != nil && !errors.Is(err, logdiag.ErrAlreadyPrinted) {
		diags = append(diags, diag.DiagnosticFromError(err))
	}
	return diags
}

// ApplySeq is a test helper to get diagnostics in this call
func ApplySeq(ctx context.Context, b *Bundle, mutators ...Mutator) diag.Diagnostics {
	if !logdiag.IsSetup(ctx) {
		ctx = logdiag.InitContext(ctx)
	}
	previous := logdiag.FlushCollected(ctx)
	if len(previous) > 0 {
		panic(fmt.Sprintf("Already have %d diags collected: %v", len(previous), previous))
	}
	logdiag.SetCollect(ctx, true)
	err := ApplySeqContext(ctx, b, mutators...)
	diags := logdiag.FlushCollected(ctx)
	// A mutator that renders its own diagnostics returns the ErrAlreadyPrinted
	// sentinel; those diagnostics are already in Collected, so only append errors
	// that were returned directly (not yet rendered).
	if err != nil && !errors.Is(err, logdiag.ErrAlreadyPrinted) {
		diags = append(diags, diag.DiagnosticFromError(err))
	}
	return diags
}
