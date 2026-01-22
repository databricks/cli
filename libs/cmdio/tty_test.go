package cmdio

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsTTY(t *testing.T) {
	tests := []struct {
		name     string
		writer   io.Writer
		expected bool
	}{
		{
			name:     "os.Stdout",
			writer:   os.Stdout,
			expected: true, // May be false in CI, but we handle that below
		},
		{
			name:     "os.Stderr",
			writer:   os.Stderr,
			expected: true, // May be false in CI, but we handle that below
		},
		{
			name:     "bytes.Buffer",
			writer:   &bytes.Buffer{},
			expected: false,
		},
		{
			name:     "io.Discard",
			writer:   io.Discard,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTTY(tt.writer)
			// For os.File tests (stdout/stderr), we can't assert the exact value
			// since it depends on the environment (TTY in terminal vs non-TTY in CI)
			if _, ok := tt.writer.(*os.File); ok {
				// Just verify it returns a boolean (no panic)
				_ = result
			} else {
				// For non-file writers, we can assert they're not TTYs
				assert.False(t, result)
			}
		})
	}
}

func TestFakeTTY(t *testing.T) {
	// Test that a regular buffer is not a TTY
	buf := &bytes.Buffer{}
	assert.False(t, isTTY(buf))

	// Test that FakeTTY wrapping makes isTTY return true
	fakeTTY := FakeTTY(buf)
	assert.True(t, isTTY(fakeTTY))

	// Test that FakeTTY is still a valid io.Writer
	n, err := fakeTTY.Write([]byte("test"))
	assert.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.Equal(t, "test", buf.String())
}

func TestFakeTTY_PreservesWriterBehavior(t *testing.T) {
	buf := &bytes.Buffer{}
	fake := FakeTTY(buf)

	// Write multiple times
	_, err := fake.Write([]byte("hello "))
	assert.NoError(t, err)
	_, err = fake.Write([]byte("world"))
	assert.NoError(t, err)

	// Verify the underlying buffer received all writes
	assert.Equal(t, "hello world", buf.String())

	// Verify isTTY still returns true
	assert.True(t, isTTY(fake))
}

func TestFakeTTY_CanBeNested(t *testing.T) {
	buf := &bytes.Buffer{}
	fake1 := FakeTTY(buf)
	fake2 := FakeTTY(fake1)

	// Both should be recognized as TTYs
	assert.True(t, isTTY(fake1))
	assert.True(t, isTTY(fake2))

	// Writing to nested fake should still work
	n, err := fake2.Write([]byte("nested"))
	assert.NoError(t, err)
	assert.Equal(t, 6, n)
	assert.Equal(t, "nested", buf.String())
}
