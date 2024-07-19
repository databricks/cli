package bundle

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIfMutatorTrue(t *testing.T) {
	m1 := &testMutator{}
	m2 := &testMutator{}
	ifMutator := If(func(context.Context, *Bundle) (bool, error) {
		return true, nil
	}, m1, m2)

	b := &Bundle{}
	diags := Apply(context.Background(), b, ifMutator)
	assert.NoError(t, diags.Error())

	assert.Equal(t, 1, m1.applyCalled)
	assert.Equal(t, 0, m2.applyCalled)
}

func TestIfMutatorFalse(t *testing.T) {
	m1 := &testMutator{}
	m2 := &testMutator{}
	ifMutator := If(func(context.Context, *Bundle) (bool, error) {
		return false, nil
	}, m1, m2)

	b := &Bundle{}
	diags := Apply(context.Background(), b, ifMutator)
	assert.NoError(t, diags.Error())

	assert.Equal(t, 0, m1.applyCalled)
	assert.Equal(t, 1, m2.applyCalled)
}

func TestIfMutatorError(t *testing.T) {
	m1 := &testMutator{}
	m2 := &testMutator{}
	ifMutator := If(func(context.Context, *Bundle) (bool, error) {
		return true, assert.AnError
	}, m1, m2)

	b := &Bundle{}
	diags := Apply(context.Background(), b, ifMutator)
	assert.Error(t, diags.Error())

	assert.Equal(t, 0, m1.applyCalled)
	assert.Equal(t, 0, m2.applyCalled)
}
