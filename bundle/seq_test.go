package bundle

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSeqMutator(t *testing.T) {
	m1 := &testMutator{}
	m2 := &testMutator{}
	m3 := &testMutator{}
	seqMutator := Seq(m1, m2, m3)

	bundle := &Bundle{}
	err := Apply(context.Background(), bundle, seqMutator)
	assert.NoError(t, err)

	assert.Equal(t, 1, m1.applyCalled)
	assert.Equal(t, 1, m2.applyCalled)
	assert.Equal(t, 1, m3.applyCalled)
}

func TestSeqWithDeferredMutator(t *testing.T) {
	m1 := &testMutator{}
	m2 := &testMutator{}
	m3 := &testMutator{}
	m4 := &testMutator{}
	seqMutator := Seq(m1, Defer(m2, m3), m4)

	bundle := &Bundle{}
	err := Apply(context.Background(), bundle, seqMutator)
	assert.NoError(t, err)

	assert.Equal(t, 1, m1.applyCalled)
	assert.Equal(t, 1, m2.applyCalled)
	assert.Equal(t, 1, m3.applyCalled)
	assert.Equal(t, 1, m4.applyCalled)
}

func TestSeqWithErrorAndDeferredMutator(t *testing.T) {
	errorMut := &mutatorWithError{errorMsg: "error msg"}
	m1 := &testMutator{}
	m2 := &testMutator{}
	m3 := &testMutator{}
	seqMutator := Seq(errorMut, Defer(m1, m2), m3)

	bundle := &Bundle{}
	err := Apply(context.Background(), bundle, seqMutator)
	assert.Error(t, err)

	assert.Equal(t, 1, errorMut.applyCalled)
	assert.Equal(t, 0, m1.applyCalled)
	assert.Equal(t, 0, m2.applyCalled)
	assert.Equal(t, 0, m3.applyCalled)
}

func TestSeqWithErrorInsideDeferredMutator(t *testing.T) {
	errorMut := &mutatorWithError{errorMsg: "error msg"}
	m1 := &testMutator{}
	m2 := &testMutator{}
	m3 := &testMutator{}
	seqMutator := Seq(m1, Defer(errorMut, m2), m3)

	bundle := &Bundle{}
	err := Apply(context.Background(), bundle, seqMutator)
	assert.Error(t, err)

	assert.Equal(t, 1, m1.applyCalled)
	assert.Equal(t, 1, errorMut.applyCalled)
	assert.Equal(t, 1, m2.applyCalled)
	assert.Equal(t, 0, m3.applyCalled)
}

func TestSeqWithErrorInsideFinallyStage(t *testing.T) {
	errorMut := &mutatorWithError{errorMsg: "error msg"}
	m1 := &testMutator{}
	m2 := &testMutator{}
	m3 := &testMutator{}
	seqMutator := Seq(m1, Defer(m2, errorMut), m3)

	bundle := &Bundle{}
	err := Apply(context.Background(), bundle, seqMutator)
	assert.Error(t, err)

	assert.Equal(t, 1, m1.applyCalled)
	assert.Equal(t, 1, m2.applyCalled)
	assert.Equal(t, 1, errorMut.applyCalled)
	assert.Equal(t, 0, m3.applyCalled)
}
