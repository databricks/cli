package bundle

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/diag"
	"github.com/stretchr/testify/assert"
)

type mutatorWithError struct {
	applyCalled int
	errorMsg    string
}

func (t *mutatorWithError) Name() string {
	return "mutatorWithError"
}

func (t *mutatorWithError) Apply(_ context.Context, b *Bundle) diag.Diagnostics {
	t.applyCalled++
	return diag.Errorf(t.errorMsg)
}

func TestDeferredMutatorWhenAllMutatorsSucceed(t *testing.T) {
	m1 := &testMutator{}
	m2 := &testMutator{}
	m3 := &testMutator{}
	cleanup := &testMutator{}
	deferredMutator := Defer(Seq(m1, m2, m3), cleanup)

	b := &Bundle{}
	diags := Apply(context.Background(), b, deferredMutator)
	assert.NoError(t, diags.Error())

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
	diags := Apply(context.Background(), b, deferredMutator)
	assert.ErrorContains(t, diags.Error(), "mutator error occurred")

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
	diags := Apply(context.Background(), b, deferredMutator)
	assert.ErrorContains(t, diags.Error(), "mutator error occurred")

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
	diags := Apply(context.Background(), b, deferredMutator)
	assert.ErrorContains(t, diags.Error(), "mutator error occurred")

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
	diags := Apply(context.Background(), b, deferredMutator)

	var errs []string
	for _, d := range diags {
		errs = append(errs, d.Summary)
	}
	assert.Contains(t, errs, "mutator error occurred")
	assert.Contains(t, errs, "cleanup error occurred")

	assert.Equal(t, 1, m1.applyCalled)
	assert.Equal(t, 1, m2.applyCalled)
	assert.Equal(t, 1, mErr.applyCalled)
	assert.Equal(t, 1, cleanupErr.applyCalled)
}
