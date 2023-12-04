package bundle

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testMutator struct {
	applyCalled    int
	nestedMutators []Mutator
}

func (t *testMutator) Name() string {
	return "test"
}

func (t *testMutator) Apply(ctx context.Context, b *Bundle) error {
	t.applyCalled++
	return Apply(ctx, b, Seq(t.nestedMutators...))
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
	err := Apply(context.Background(), b, m)
	assert.NoError(t, err)

	assert.Equal(t, 1, m.applyCalled)
	assert.Equal(t, 1, nested[0].applyCalled)
	assert.Equal(t, 1, nested[1].applyCalled)
}
