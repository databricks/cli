package bundletest

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

func benchmarkRequiredMutator(b *testing.B, numJobs int) {
	myBundle := Bundle(b, numJobs)

	var diags diag.Diagnostics
	for b.Loop() {
		diags = bundle.Apply(context.Background(), myBundle, validate.Required())
	}
	assert.NotEmpty(b, diags)
}

func benchmarkWalkReadOnlyBaseline(b *testing.B, numJobs int) {
	myBundle := Bundle(b, numJobs)

	for b.Loop() {
		var paths []dyn.Path
		bundle.ApplyFuncContext(context.Background(), myBundle, func(ctx context.Context, b *bundle.Bundle) {
			_ = dyn.WalkReadOnly(b.Config.Value(), func(p dyn.Path, v dyn.Value) error {
				paths = append(paths, p)
				return nil
			})
		})
	}
}

func benchmarkNoopBaseline(b *testing.B, numJobs int) {
	myBundle := Bundle(b, numJobs)

	for b.Loop() {
		bundle.ApplyFuncContext(context.Background(), myBundle, func(ctx context.Context, b *bundle.Bundle) {})
	}
}

// This benchmark took 823ms to run on 10th July 2025.
func BenchmarkWalkReadOnlyBaseline(b *testing.B) {
	benchmarkWalkReadOnlyBaseline(b, 10000)
}

// This benchmark took 774ms to run on 10th July 2025.
func BenchmarkNoopBaseline(b *testing.B) {
	benchmarkNoopBaseline(b, 10000)
}

// This benchmark took 996ms to run on 10th July 2025.
func BenchmarkValidateRequired10000(b *testing.B) {
	benchmarkRequiredMutator(b, 10000)
}

// This benchmark took 98ms to run on 10th July 2025.
func BenchmarkValidateRequired1000(b *testing.B) {
	benchmarkRequiredMutator(b, 1000)
}

// This benchmark took 10ms to run on 10th July 2025.
func BenchmarkValidateRequired100(b *testing.B) {
	benchmarkRequiredMutator(b, 100)
}

// This benchmark took 1.1ms to run on 10th July 2025.
func BenchmarkValidateRequired10(b *testing.B) {
	benchmarkRequiredMutator(b, 10)
}
