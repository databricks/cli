package logstream

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTailBuffer_Add(t *testing.T) {
	tests := []struct {
		name     string
		size     int
		lines    []string
		wantLen  int
		wantLast string
	}{
		{
			name:     "adds lines up to size",
			size:     3,
			lines:    []string{"a", "b", "c"},
			wantLen:  3,
			wantLast: "c",
		},
		{
			name:     "keeps only last N lines when exceeding size",
			size:     2,
			lines:    []string{"a", "b", "c", "d"},
			wantLen:  2,
			wantLast: "d",
		},
		{
			name:    "size zero ignores all lines",
			size:    0,
			lines:   []string{"a", "b"},
			wantLen: 0,
		},
		{
			name:    "negative size ignores all lines",
			size:    -1,
			lines:   []string{"a", "b"},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &tailBuffer{size: tt.size}
			for _, line := range tt.lines {
				b.Add(line)
			}
			assert.Equal(t, tt.wantLen, b.Len())
			if tt.wantLen > 0 {
				assert.Equal(t, tt.wantLast, b.lines[len(b.lines)-1])
			}
		})
	}
}

func TestTailBuffer_Flush(t *testing.T) {
	t.Run("writes all lines to writer", func(t *testing.T) {
		b := &tailBuffer{size: 3}
		b.Add("line1")
		b.Add("line2")
		b.Add("line3")

		var buf bytes.Buffer
		err := b.Flush(&buf)

		require.NoError(t, err)
		assert.Equal(t, "line1\nline2\nline3\n", buf.String())
		assert.Equal(t, 0, b.Len(), "buffer should be empty after flush")
	})

	t.Run("handles buffer with no lines", func(t *testing.T) {
		for _, size := range []int{0, 3} {
			b := &tailBuffer{size: size}
			var buf bytes.Buffer
			require.NoError(t, b.Flush(&buf))
			assert.Empty(t, buf.String())
		}
	})

	t.Run("propagates write error", func(t *testing.T) {
		b := &tailBuffer{size: 2}
		b.Add("line1")

		writeErr := errors.New("write failed")
		fw := &failingWriter{err: writeErr}
		err := b.Flush(fw)

		assert.ErrorIs(t, err, writeErr)
	})
}

type failingWriter struct {
	err error
}

func (w *failingWriter) Write(p []byte) (int, error) {
	return 0, w.err
}
