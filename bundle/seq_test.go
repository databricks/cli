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

	b := &Bundle{}
	diags := Apply(context.Background(), b, seqMutator)
	assert.NoError(t, diags.Error())

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

	b := &Bundle{}
	diags := Apply(context.Background(), b, seqMutator)
	assert.NoError(t, diags.Error())

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

	b := &Bundle{}
	diags := Apply(context.Background(), b, seqMutator)
	assert.Error(t, diags.Error())

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

	b := &Bundle{}
	diags := Apply(context.Background(), b, seqMutator)
	assert.Error(t, diags.Error())

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

	b := &Bundle{}
	diags := Apply(context.Background(), b, seqMutator)
	assert.Error(t, diags.Error())

	assert.Equal(t, 1, m1.applyCalled)
	assert.Equal(t, 1, m2.applyCalled)
	assert.Equal(t, 1, errorMut.applyCalled)
	assert.Equal(t, 0, m3.applyCalled)
}

func TestSeqWithErrorSequenceBreak(t *testing.T) {
	errorMut := &mutatorWithError{errorMsg: ErrorSequenceBreak.Error()}
	m1 := &testMutator{}
	m2 := &testMutator{}
	m3 := &testMutator{}
	seqMutator := Seq(m1, m2, errorMut, m3)

	b := &Bundle{}
	diags := Apply(context.Background(), b, seqMutator)
	assert.NoError(t, diags.Error())

	assert.Equal(t, 1, m1.applyCalled)
	assert.Equal(t, 1, m2.applyCalled)
	assert.Equal(t, 1, errorMut.applyCalled)

	// m3 is not called because the error mutator returns a break control signal.
	assert.Equal(t, 0, m3.applyCalled)
}

func TestSeqWithErrorSequenceBreakInsideDeferFirst(t *testing.T) {
	errorMut := &mutatorWithError{errorMsg: ErrorSequenceBreak.Error()}
	m1 := &testMutator{}
	m2 := &testMutator{}
	m3 := &testMutator{}
	seqMutator := Seq(m1, Defer(errorMut, m2), m3)

	b := &Bundle{}
	diags := Apply(context.Background(), b, seqMutator)
	assert.NoError(t, diags.Error())

	assert.Equal(t, 1, m1.applyCalled)
	assert.Equal(t, 1, errorMut.applyCalled)

	// m2 should still be called because it's inside a Defer
	assert.Equal(t, 1, m2.applyCalled)
	assert.Equal(t, 0, m3.applyCalled)
}

func TestSeqWithErrorSequenceBreakInsideDeferSecond(t *testing.T) {
	errorMut := &mutatorWithError{errorMsg: ErrorSequenceBreak.Error()}
	m1 := &testMutator{}
	m2 := &testMutator{}
	m3 := &testMutator{}
	seqMutator := Seq(m1, Defer(m2, errorMut), m3)

	b := &Bundle{}
	diags := Apply(context.Background(), b, seqMutator)
	assert.NoError(t, diags.Error())

	assert.Equal(t, 1, m1.applyCalled)
	assert.Equal(t, 1, m2.applyCalled)
	assert.Equal(t, 1, errorMut.applyCalled)

	// m3 is not called because the defer mutator returns a break control signal.
	assert.Equal(t, 0, m3.applyCalled)
}

func TestSeqErrorSequenceBreakDoesNotBreakMultipleSequences(t *testing.T) {
	errorMut := &mutatorWithError{errorMsg: ErrorSequenceBreak.Error()}
	m1 := &testMutator{}
	m2 := &testMutator{}
	m3 := &testMutator{}
	m4 := &testMutator{}
	seqMutator := Seq(Seq(m1, errorMut, m2), Seq(m3, m4))

	b := &Bundle{}
	diags := Apply(context.Background(), b, seqMutator)
	assert.NoError(t, diags.Error())

	assert.Equal(t, 1, m1.applyCalled)
	assert.Equal(t, 1, errorMut.applyCalled)

	// m2 is not applied because the error mutator returns a break control signal.
	assert.Equal(t, 0, m2.applyCalled)

	// m3 and m4 are still applied because the break control signal error should
	// only break the current sequence, not the top level one.
	assert.Equal(t, 1, m3.applyCalled)
	assert.Equal(t, 1, m4.applyCalled)
}
