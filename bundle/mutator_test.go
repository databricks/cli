package bundle

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/diag"
	"github.com/stretchr/testify/assert"
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
			name:     "setDefaults mutator",
			mutator:  &setDefaults{},
			expected: "bundle.(setDefaults)",
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
