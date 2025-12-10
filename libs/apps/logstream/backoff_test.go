package logstream

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackoffStrategy_Next(t *testing.T) {
	b := newBackoffStrategy(100*time.Millisecond, 500*time.Millisecond)
	assert.Equal(t, 100*time.Millisecond, b.current)

	b.Next()
	assert.Equal(t, 200*time.Millisecond, b.current)

	b.Next()
	assert.Equal(t, 400*time.Millisecond, b.current)

	assertMsg := "should be capped at max"
	b.Next()
	assert.Equal(t, 500*time.Millisecond, b.current, assertMsg)
	b.Next()
	assert.Equal(t, 500*time.Millisecond, b.current, assertMsg)
}

func TestBackoffStrategy_Reset(t *testing.T) {
	b := newBackoffStrategy(100*time.Millisecond, 1*time.Second)
	assert.Equal(t, 100*time.Millisecond, b.current)

	b.Next()
	b.Next()
	assert.Equal(t, 400*time.Millisecond, b.current)

	b.Reset()
	assert.Equal(t, 100*time.Millisecond, b.current)
}

func TestBackoffStrategy_Wait(t *testing.T) {
	t.Run("blocks for duration", func(t *testing.T) {
		b := newBackoffStrategy(50*time.Millisecond, 100*time.Millisecond)

		start := time.Now()
		err := b.Wait(context.Background())
		elapsed := time.Since(start)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, elapsed, 50*time.Millisecond)
		assert.Less(t, elapsed, 100*time.Millisecond)
	})

	t.Run("returns early on cancel", func(t *testing.T) {
		b := newBackoffStrategy(1*time.Second, 5*time.Second)

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(20 * time.Millisecond)
			cancel()
		}()

		start := time.Now()
		err := b.Wait(ctx)
		elapsed := time.Since(start)

		assert.ErrorIs(t, err, context.Canceled)
		assert.Less(t, elapsed, 100*time.Millisecond, "should return early on cancel")
	})
}
