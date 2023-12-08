package bundle

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mutatorWithError struct {
	applyCalled int
	errorMsg    string
}

func (t *mutatorWithError) Name() string {
	return "mutatorWithError"
}

func (t *mutatorWithError) Apply(_ context.Context, b *Bundle) error {
	t.applyCalled++
	return fmt.Errorf(t.errorMsg)
}

type checkErrorMutator struct {
	applyCalled   int
	expectedError string
}

func (c *checkErrorMutator) Name() string {
	return "checkError"
}

func (c *checkErrorMutator) Apply(ctx context.Context, b *Bundle) error {
	c.applyCalled++
	mainErr := ErrFromContext(ctx)
	if mainErr != nil && mainErr.Error() == c.expectedError {
		return nil // Correct error found
	}
	return fmt.Errorf("expected error '%s', but found '%v'", c.expectedError, mainErr)
}

func TestDeferredMutatorWhenAllMutatorsSucceed(t *testing.T) {
	m1 := &testMutator{}
	m2 := &testMutator{}
	m3 := &testMutator{}
	cleanup := &testMutator{}
	deferredMutator := Defer(Seq(m1, m2, m3), cleanup)

	b := &Bundle{}
	err := Apply(context.Background(), b, deferredMutator)
	assert.NoError(t, err)

	assert.Equal(t, 1, m1.applyCalled)
	assert.Equal(t, 1, m2.applyCalled)
	assert.Equal(t, 1, m3.applyCalled)
	assert.Equal(t, 1, cleanup.applyCalled)
}

func TestDeferredMutatorWhenFirstFails(t *testing.T) {
	m1 := &testMutator{}
	m2 := &testMutator{}
	mErr := &mutatorWithError{errorMsg: "mutator error occurred"}
	cleanup := &testMutator{}
	deferredMutator := Defer(Seq(mErr, m1, m2), cleanup)

	b := &Bundle{}
	err := Apply(context.Background(), b, deferredMutator)
	assert.ErrorContains(t, err, "mutator error occurred")

	assert.Equal(t, 1, mErr.applyCalled)
	assert.Equal(t, 0, m1.applyCalled)
	assert.Equal(t, 0, m2.applyCalled)
	assert.Equal(t, 1, cleanup.applyCalled)
}

func TestDeferredMutatorWhenMiddleOneFails(t *testing.T) {
	m1 := &testMutator{}
	m2 := &testMutator{}
	mErr := &mutatorWithError{errorMsg: "mutator error occurred"}
	cleanup := &testMutator{}
	deferredMutator := Defer(Seq(m1, mErr, m2), cleanup)

	b := &Bundle{}
	err := Apply(context.Background(), b, deferredMutator)
	assert.ErrorContains(t, err, "mutator error occurred")

	assert.Equal(t, 1, m1.applyCalled)
	assert.Equal(t, 1, mErr.applyCalled)
	assert.Equal(t, 0, m2.applyCalled)
	assert.Equal(t, 1, cleanup.applyCalled)
}

func TestDeferredMutatorWhenLastOneFails(t *testing.T) {
	m1 := &testMutator{}
	m2 := &testMutator{}
	mErr := &mutatorWithError{errorMsg: "mutator error occurred"}
	cleanup := &testMutator{}
	deferredMutator := Defer(Seq(m1, m2, mErr), cleanup)

	b := &Bundle{}
	err := Apply(context.Background(), b, deferredMutator)
	assert.ErrorContains(t, err, "mutator error occurred")

	assert.Equal(t, 1, m1.applyCalled)
	assert.Equal(t, 1, m2.applyCalled)
	assert.Equal(t, 1, mErr.applyCalled)
	assert.Equal(t, 1, cleanup.applyCalled)
}

func TestDeferredMutatorCombinesErrorMessages(t *testing.T) {
	m1 := &testMutator{}
	m2 := &testMutator{}
	mErr := &mutatorWithError{errorMsg: "mutator error occurred"}
	cleanupErr := &mutatorWithError{errorMsg: "cleanup error occurred"}
	deferredMutator := Defer(Seq(m1, m2, mErr), cleanupErr)

	b := &Bundle{}
	err := Apply(context.Background(), b, deferredMutator)
	assert.ErrorContains(t, err, "mutator error occurred\ncleanup error occurred")

	assert.Equal(t, 1, m1.applyCalled)
	assert.Equal(t, 1, m2.applyCalled)
	assert.Equal(t, 1, mErr.applyCalled)
	assert.Equal(t, 1, cleanupErr.applyCalled)
}

func TestDeferredMutatorPassesErrorToFinally(t *testing.T) {
	mErr := &mutatorWithError{errorMsg: "mutator error occurred"}
	finalCheck := &testMutator{
		nestedMutators: []Mutator{
			&checkErrorMutator{expectedError: "mutator error occurred"},
		},
	}
	deferredMutator := Defer(mErr, finalCheck)

	b := &Bundle{}
	err := Apply(context.Background(), b, deferredMutator)

	assert.ErrorContains(t, err, "mutator error occurred")
	assert.Equal(t, 1, mErr.applyCalled)
	assert.Equal(t, 1, finalCheck.applyCalled)
	assert.Equal(t, 1, finalCheck.nestedMutators[0].(*checkErrorMutator).applyCalled)
}
