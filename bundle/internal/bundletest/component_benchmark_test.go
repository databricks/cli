package bundletest

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/stretchr/testify/require"
)

// The benchmarks below run one mutator (or loading) on a bundle with 1000 jobs. They
// only use APIs that also exist in the libs/dyn implementation, so the same file can
// be run there to compare components one by one.

const componentJobs = 1000

func benchmarkMutator(b *testing.B, m bundle.Mutator) {
	myBundle := Bundle(b, componentJobs)
	myBundle.BundleRootPath = b.TempDir()
	myBundle.SyncRootPath = myBundle.BundleRootPath
	for b.Loop() {
		bundle.Apply(b.Context(), myBundle, m)
	}
}

// BenchmarkComponentNoop measures the cost bundle.Apply adds around every mutator, so
// it can be subtracted from the other results.
func BenchmarkComponentNoop(b *testing.B) {
	myBundle := Bundle(b, componentJobs)
	for b.Loop() {
		bundle.ApplyFuncContext(b.Context(), myBundle, func(context.Context, *bundle.Bundle) {})
	}
}

func BenchmarkComponentRequired(b *testing.B) { benchmarkMutator(b, validate.Required()) }

func BenchmarkComponentEnum(b *testing.B) { benchmarkMutator(b, validate.Enum()) }

func BenchmarkComponentSingleNodeCluster(b *testing.B) {
	benchmarkMutator(b, validate.SingleNodeCluster())
}

func BenchmarkComponentTFOnlyReferences(b *testing.B) {
	benchmarkMutator(b, validate.TFOnlyReferences())
}

func BenchmarkComponentTranslatePaths(b *testing.B) { benchmarkMutator(b, mutator.TranslatePaths()) }

func BenchmarkComponentNormalizePaths(b *testing.B) { benchmarkMutator(b, mutator.NormalizePaths()) }

func BenchmarkComponentRewriteWorkspacePrefix(b *testing.B) {
	benchmarkMutator(b, mutator.RewriteWorkspacePrefix())
}

func BenchmarkComponentPrependWorkspacePrefix(b *testing.B) {
	benchmarkMutator(b, mutator.PrependWorkspacePrefix())
}

func BenchmarkComponentDropEmptyStrings(b *testing.B) {
	benchmarkMutator(b, mutator.DropEmptyStrings())
}

func BenchmarkComponentResolveVariableReferences(b *testing.B) {
	benchmarkMutator(b, mutator.ResolveVariableReferencesOnlyResources("bundle", "workspace", "variables"))
}

func BenchmarkComponentExpandGlobReferences(b *testing.B) {
	benchmarkMutator(b, libraries.ExpandGlobReferences())
}

// componentLoadInput returns a configuration file with 1000 jobs (JSON is valid YAML).
func componentLoadInput(b *testing.B) []byte {
	var job map[string]any
	require.NoError(b, json.Unmarshal([]byte(jobExample), &job))
	jobs := map[string]any{}
	for i := range componentJobs {
		jobs[fmt.Sprintf("job_%d", i)] = job
	}
	raw, err := json.Marshal(map[string]any{
		"bundle":    map[string]any{"name": "benchmark"},
		"resources": map[string]any{"jobs": jobs},
	})
	require.NoError(b, err)
	require.True(b, strings.HasPrefix(string(raw), "{"))
	return raw
}

// BenchmarkComponentLoad loads a configuration file with 1000 jobs into the typed
// configuration: [BenchmarkComponentLoadDyn] plus [BenchmarkComponentFinishLoading].
func BenchmarkComponentLoad(b *testing.B) {
	raw := componentLoadInput(b)
	for b.Loop() {
		_, diags := config.LoadFromBytes("databricks.yml", raw)
		require.False(b, diags.HasError(), "%v", diags)
	}
}

// BenchmarkComponentLoadDyn loads a configuration file with 1000 jobs into a loading
// configuration (configuration tree), like the load phase does for every file.
func BenchmarkComponentLoadDyn(b *testing.B) {
	raw := componentLoadInput(b)
	for b.Loop() {
		_, diags := config.LoadDynFromBytes("databricks.yml", raw)
		require.False(b, diags.HasError(), "%v", diags)
	}
}

// BenchmarkComponentFinishLoading converts a loading configuration with 1000 jobs into
// the typed configuration, which the load phase does once.
func BenchmarkComponentFinishLoading(b *testing.B) {
	r, diags := config.LoadDynFromBytes("databricks.yml", componentLoadInput(b))
	require.False(b, diags.HasError(), "%v", diags)
	for b.Loop() {
		c := *r
		_, err := c.FinishLoading()
		require.NoError(b, err)
	}
}
