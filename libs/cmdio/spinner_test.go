package cmdio

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSpinnerModelInit(t *testing.T) {
	m := newSpinnerModel()
	assert.False(t, m.quitting)
	assert.Equal(t, "", m.suffix)
	assert.NotNil(t, m.spinner)
}

func TestSpinnerModelUpdateSuffixMsg(t *testing.T) {
	m := newSpinnerModel()
	msg := suffixMsg("processing files")

	updatedModel, _ := m.Update(msg)
	updated := updatedModel.(spinnerModel)

	assert.Equal(t, "processing files", updated.suffix)
	assert.False(t, updated.quitting)
}

func TestSpinnerModelUpdateQuitMsg(t *testing.T) {
	m := newSpinnerModel()
	msg := quitMsg{}

	updatedModel, cmd := m.Update(msg)
	updated := updatedModel.(spinnerModel)

	assert.True(t, updated.quitting)
	assert.NotNil(t, cmd) // Should return tea.Quit
}

func TestSpinnerModelViewActive(t *testing.T) {
	m := newSpinnerModel()
	m.suffix = "loading"

	view := m.View()

	assert.NotEmpty(t, view)
	assert.Contains(t, view, "loading")
}

func TestSpinnerModelViewQuitting(t *testing.T) {
	m := newSpinnerModel()
	m.quitting = true

	view := m.View()

	assert.Empty(t, view)
}

func TestSpinnerStructUpdateBeforeClose(t *testing.T) {
	ctx := context.Background()
	ctx, _ = NewTestContextWithStderr(ctx)

	sp := NewSpinner(ctx)

	sp.Update("test message 1")
	sp.Update("test message 2")
	sp.Close()

	// No panics = success
}

func TestSpinnerStructCloseIdempotent(t *testing.T) {
	ctx := context.Background()
	ctx, _ = NewTestContextWithStderr(ctx)

	sp := NewSpinner(ctx)

	sp.Close()
	sp.Close() // Should not panic
	sp.Close() // Should not panic
}

func TestSpinnerStructUpdateAfterClose(t *testing.T) {
	ctx := context.Background()
	ctx, _ = NewTestContextWithStderr(ctx)

	sp := NewSpinner(ctx)

	sp.Close()
	sp.Update("after close") // Should not panic
}

func TestSpinnerStructNonInteractive(t *testing.T) {
	ctx := context.Background()
	// Create context without TTY simulation (non-interactive)
	ctx, _ = NewTestContextWithStderr(ctx)

	sp := NewSpinner(ctx)
	sp.Update("should be discarded")
	sp.Close()

	// Should complete without error in non-interactive mode
}

func TestSpinnerBackwardCompatibility(t *testing.T) {
	ctx := context.Background()
	ctx, _ = NewTestContextWithStderr(ctx)

	// Old API should still work
	spinner := Spinner(ctx)
	spinner <- "old api message"
	close(spinner)

	// No panics = success
}

func TestSpinnerStructContextCancellation(t *testing.T) {
	ctx := context.Background()
	ctx, _ = NewTestContextWithStderr(ctx)

	ctx, cancel := context.WithCancel(ctx)

	sp := NewSpinner(ctx)

	sp.Update("message")
	cancel() // Should trigger cleanup
	time.Sleep(100 * time.Millisecond)

	// Spinner should handle cancellation gracefully
	// Close should still be safe to call
	sp.Close()
}

func TestSpinnerStructConcurrentClose(t *testing.T) {
	ctx := context.Background()
	ctx, _ = NewTestContextWithStderr(ctx)

	ctx, cancel := context.WithCancel(ctx)

	sp := NewSpinner(ctx)
	sp.Update("message")

	// Trigger concurrent Close() from context cancellation and explicit call
	var wg sync.WaitGroup

	// Goroutine 1: Cancel context (triggers Close via context handler)
	wg.Go(func() {
		cancel()
	})

	// Goroutine 2: Explicit Close
	wg.Go(func() {
		sp.Close()
	})

	// Both should complete without deadlock or race
	wg.Wait()

	// Additional Close should still be safe
	sp.Close()
}
