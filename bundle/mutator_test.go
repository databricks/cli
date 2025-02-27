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
