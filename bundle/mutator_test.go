package bundle

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

type testMutator struct {
	applyCalled    int
	nestedMutators []Mutator
}

func (t *testMutator) Name() string {
	return "test"
}

func (t *testMutator) Apply(ctx context.Context, b *Bundle) diag.Diagnostics {
	t.applyCalled++
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
	diags := Apply(context.Background(), b, m)
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
