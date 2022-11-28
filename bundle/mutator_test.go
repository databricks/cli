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

func (t *testMutator) Apply(_ context.Context, b *Bundle) ([]Mutator, error) {
	t.applyCalled++
	return t.nestedMutators, nil
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

	bundle := &Bundle{}
	err := Apply(context.Background(), bundle, []Mutator{m})
	assert.NoError(t, err)

	assert.Equal(t, 1, m.applyCalled)
	assert.Equal(t, 1, nested[0].applyCalled)
	assert.Equal(t, 1, nested[1].applyCalled)
}
