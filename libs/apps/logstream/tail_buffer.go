package logstream

import (
	"fmt"
	"io"
	"slices"
)

// tailBuffer is a circular buffer that stores the last N lines of log output.
type tailBuffer struct {
	size  int
	lines []string
}

// Add adds a line to the buffer.
func (b *tailBuffer) Add(line string) {
	if b.size <= 0 {
		return
	}
	b.lines = append(b.lines, line)
	if len(b.lines) > b.size {
		b.lines = slices.Delete(b.lines, 0, len(b.lines)-b.size)
	}
}

// Len returns the number of lines in the buffer.
func (b *tailBuffer) Len() int {
	return len(b.lines)
}

// Flush writes the lines in the buffer to the writer.
func (b *tailBuffer) Flush(w io.Writer) error {
	if b.size == 0 {
		return nil
	}
	for _, line := range b.lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	b.lines = slices.Clip(b.lines[:0])
	return nil
}
