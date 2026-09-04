package bundle

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testMutator struct {
	applyCalled    int
	nestedMutators []Mutator
	// fn, if set, runs inside Apply after the call is counted.
	fn func(ctx context.Context, b *Bundle) diag.Diagnostics
}

func (t *testMutator) Name() string {
	return "test"
}

func (t *testMutator) Apply(ctx context.Context, b *Bundle) diag.Diagnostics {
	t.applyCalled++
	if t.fn != nil {
		return t.fn(ctx, b)
	}
	return ApplySeq(ctx, b, t.nestedMutators...)
}

func TestMutator(t *testing.T) {
	nested := []*testMutator{
		{},
		{},
	}

	m := &testMutator{
		nestedMutators: []Mutator{
			nested[0],
			nested[1],
		},
	}

	b := &Bundle{}
	diags := Apply(t.Context(), b, m)
	assert.NoError(t, diags.Error())

	assert.Equal(t, 1, m.applyCalled)
	assert.Equal(t, 1, nested[0].applyCalled)
	assert.Equal(t, 1, nested[1].applyCalled)
}

// ApplySeqInScopeContext applies each mutator once, in order.
func TestApplySeqInScopeContext(t *testing.T) {
	var order []string
	makeMutator := func(name string) *testMutator {
		return &testMutator{fn: func(ctx context.Context, b *Bundle) diag.Diagnostics {
			order = append(order, name)
			return nil
		}}
	}
	first := makeMutator("first")
	second := makeMutator("second")

	b := &Bundle{}
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	ApplySeqInScopeContext(ctx, b, first, second)

	assert.Equal(t, 1, first.applyCalled)
	assert.Equal(t, 1, second.applyCalled)
	assert.Equal(t, []string{"first", "second"}, order)
	assert.Empty(t, logdiag.FlushCollected(ctx))
}

// ApplySeqInScopeContext collects the diagnostics returned by each mutator and
// stops at the first one that logs an error, without applying later mutators.
func TestApplySeqInScopeContextStopsOnError(t *testing.T) {
	failing := &testMutator{fn: func(ctx context.Context, b *Bundle) diag.Diagnostics {
		return diag.Diagnostics{diag.Diagnostic{Severity: diag.Error, Summary: "boom"}}
	}}
	later := &testMutator{}

	b := &Bundle{}
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	ApplySeqInScopeContext(ctx, b, failing, later)

	assert.Equal(t, 1, failing.applyCalled)
	assert.Equal(t, 0, later.applyCalled, "mutator after an error must not run")

	diags := logdiag.FlushCollected(ctx)
	require.Len(t, diags, 1)
	assert.Equal(t, "boom", diags[0].Summary)
}

// Unlike ApplySeqContext, ApplySeqInScopeContext does not open a mutator scope per
// mutator, so it must be called from within one. Changes made through Root.Mutate
// keep the typed and dynamic configuration in sync and survive without a per-mutator
// scope. This is the property ProcessRootIncludes relies on.
func TestApplySeqInScopeContextPreservesMutateChanges(t *testing.T) {
	setHost := &testMutator{fn: func(ctx context.Context, b *Bundle) diag.Diagnostics {
		err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
			return dyn.Set(v, "bundle", dyn.V(map[string]dyn.Value{"name": dyn.V("set-in-scope")}))
		})
		require.NoError(t, err)
		return nil
	}}

	// Enclosing mutator that applies setHost in its own scope, mirroring how
	// ProcessRootIncludes applies the per-file includes.
	outer := &testMutator{fn: func(ctx context.Context, b *Bundle) diag.Diagnostics {
		ApplySeqInScopeContext(ctx, b, setHost)
		return nil
	}}

	b := &Bundle{}
	diags := Apply(t.Context(), b, outer)
	require.NoError(t, diags.Error())

	// Visible in both representations once the enclosing scope exits.
	assert.Equal(t, "set-in-scope", b.Config.Bundle.Name)
	assert.Equal(t, "set-in-scope", b.Config.Value().Get("bundle").Get("name").MustString())
}

func TestSafeMutatorName(t *testing.T) {
	tests := []struct {
		name     string
		mutator  Mutator
		expected string
	}{
		{
			name:     "funcMutator",
			mutator:  funcMutator{fn: nil},
			expected: "bundle.(funcMutator)",
		},
		{
			name:     "setDefault mutator",
			mutator:  SetDefaultMutator(dyn.NewPattern(dyn.Key("test")), "key", "value"),
			expected: "bundle.(setDefault)",
		},
		{
			name:     "funcMutator as pointer",
			mutator:  &funcMutator{fn: nil},
			expected: "bundle.(funcMutator)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := safeMutatorName(tt.mutator)
			assert.Equal(t, tt.expected, result, "mutatorName should return correct package.type format")
		})
	}
}
