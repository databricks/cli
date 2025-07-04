package bundle

import (
	"context"
	"reflect"
	"time"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry/protos"
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

func Apply(ctx context.Context, b *Bundle, m Mutator) diag.Diagnostics {
	// Track the execution time of the mutator.
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
		log.Errorf(ctx, "entry error: %s", err)
		return diag.Errorf("entry error: %s", err)
	}

	defer func() {
		err := b.Config.MarkMutatorExit(ctx)
		if err != nil {
			log.Errorf(ctx, "exit error: %s", err)
		}
	}()

	diags := m.Apply(ctx, b)

	// Log error in diagnostics if any.
	// Note: errors should be logged when constructing them
	// such that they are not logged multiple times.
	// If this is done, we can omit this block.
	if err := diags.Error(); err != nil {
		log.Debugf(ctx, "Error: %s", err)
	}

	return diags
}

func ApplySeq(ctx context.Context, b *Bundle, mutators ...Mutator) diag.Diagnostics {
	diags := diag.Diagnostics{}
	for _, m := range mutators {
		diags = diags.Extend(Apply(ctx, b, m))
		if diags.HasError() {
			return diags
		}
	}
	return diags
}

type funcMutator struct {
	fn func(context.Context, *Bundle) diag.Diagnostics
}

func (m funcMutator) Name() string {
	return "<func>"
}

func (m funcMutator) Apply(ctx context.Context, b *Bundle) diag.Diagnostics {
	return m.fn(ctx, b)
}

// ApplyFunc applies an inline-specified function mutator.
func ApplyFunc(ctx context.Context, b *Bundle, fn func(context.Context, *Bundle) diag.Diagnostics) diag.Diagnostics {
	return Apply(ctx, b, funcMutator{fn})
}
