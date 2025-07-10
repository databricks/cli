package bundle

import (
	"context"
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

	// Apply mutates the specified bundle object.
	Apply(context.Context, *Bundle) diag.Diagnostics
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

func ApplyContext(ctx context.Context, b *Bundle, m Mutator) {
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

	err := b.Config.MarkMutatorEntry(ctx)
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("entry error: %w", err))
		return
	}

	defer func() {
		err := b.Config.MarkMutatorExit(ctx)
		if err != nil {
			logdiag.LogError(ctx, fmt.Errorf("exit error: %w", err))
		}
	}()

	diags := m.Apply(ctx, b)

	for _, d := range diags {
		logdiag.LogDiag(ctx, d)
	}
}

func ApplySeqContext(ctx context.Context, b *Bundle, mutators ...Mutator) {
	for _, m := range mutators {
		ApplyContext(ctx, b, m)
		if logdiag.HasError(ctx) {
			break
		}
	}
}

type funcMutator struct {
	fn func(context.Context, *Bundle)
}

func (m funcMutator) Name() string {
	return "<func>"
}

func (m funcMutator) Apply(ctx context.Context, b *Bundle) diag.Diagnostics {
	m.fn(ctx, b)
	return nil
}

// ApplyFuncContext applies an inline-specified function mutator.
func ApplyFuncContext(ctx context.Context, b *Bundle, fn func(context.Context, *Bundle)) {
	ApplyContext(ctx, b, funcMutator{fn})
}

// Test helpers. TODO: move to separate package.

func Apply(ctx context.Context, b *Bundle, m Mutator) diag.Diagnostics {
	if !logdiag.IsSetup(ctx) {
		ctx = logdiag.InitContext(ctx)
	}
	previous := logdiag.FlushCollected(ctx)
	if len(previous) > 0 {
		panic(fmt.Sprintf("Already have %d diags collected: %v", len(previous), previous))
	}
	logdiag.SetCollect(ctx, true)
	ApplyContext(ctx, b, m)
	return logdiag.FlushCollected(ctx)
}

// Test helper to get diagnostics in this call
func ApplySeq(ctx context.Context, b *Bundle, mutators ...Mutator) diag.Diagnostics {
	if !logdiag.IsSetup(ctx) {
		ctx = logdiag.InitContext(ctx)
	}
	previous := logdiag.FlushCollected(ctx)
	if len(previous) > 0 {
		panic(fmt.Sprintf("Already have %d diags collected: %v", len(previous), previous))
	}
	logdiag.SetCollect(ctx, true)
	ApplySeqContext(ctx, b, mutators...)
	return logdiag.FlushCollected(ctx)
}
