package ucm

import (
	"context"
	"testing"
	"time"

	"github.com/databricks/cli/libs/diag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testMutator struct {
	applyCalled    int
	nestedMutators []Mutator
}

func (t *testMutator) Name() string {
	return "test"
}

func (t *testMutator) Apply(ctx context.Context, u *Ucm) diag.Diagnostics {
	t.applyCalled++
	return ApplySeq(ctx, u, t.nestedMutators...)
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

	u := &Ucm{}
	diags := Apply(t.Context(), u, m)
	assert.NoError(t, diags.Error())

	assert.Equal(t, 1, m.applyCalled)
	assert.Equal(t, 1, nested[0].applyCalled)
	assert.Equal(t, 1, nested[1].applyCalled)
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
			expected: "ucm.(funcMutator)",
		},
		{
			name:     "funcMutator as pointer",
			mutator:  &funcMutator{fn: nil},
			expected: "ucm.(funcMutator)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := safeMutatorName(tt.mutator)
			assert.Equal(t, tt.expected, result, "mutatorName should return correct package.type format")
		})
	}
}

type slowMutator struct{}

func (slowMutator) Name() string { return "slow" }

func (slowMutator) Apply(_ context.Context, _ *Ucm) diag.Diagnostics {
	time.Sleep(5 * time.Millisecond)
	return nil
}

func TestApplyContextRecordsExecutionTime(t *testing.T) {
	u := &Ucm{}
	diags := Apply(t.Context(), u, slowMutator{})
	require.NoError(t, diags.Error())

	require.Len(t, u.Metrics.ExecutionTimes, 1)
	assert.Equal(t, "ucm.(slowMutator)", u.Metrics.ExecutionTimes[0].Key)
	assert.Positive(t, u.Metrics.ExecutionTimes[0].Value)
}
