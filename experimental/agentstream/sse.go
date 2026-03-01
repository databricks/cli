package agentstream

import (
	"bufio"
	"io"
	"strings"
)

const maxSSELineSize = 1 << 20 // 1MB

// SSEEvent represents a single SSE event with its raw data payload.
type SSEEvent struct {
	Data string
}

// SSEReader reads SSE events from an io.Reader.
type SSEReader struct {
	scanner *bufio.Scanner
}

// NewSSEReader creates a new SSE reader with a 1MB buffer.
func NewSSEReader(r io.Reader) *SSEReader {
	s := bufio.NewScanner(r)
	s.Buffer(make([]byte, 0, 64*1024), maxSSELineSize)
	return &SSEReader{scanner: s}
}

// Next reads the next SSE event. Returns io.EOF when the stream ends.
func (r *SSEReader) Next() (*SSEEvent, error) {
	var dataLines []string

	for r.scanner.Scan() {
		line := r.scanner.Text()

		if line == "" {
			// Blank line terminates an event.
			if len(dataLines) > 0 {
				return &SSEEvent{Data: strings.Join(dataLines, "\n")}, nil
			}
			continue
		}

		// SSE data field.
		if strings.HasPrefix(line, "data: ") {
			dataLines = append(dataLines, line[len("data: "):])
		} else if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, line[len("data:"):])
		}
		// Ignore other SSE fields (event:, id:, retry:) and comments (:).
	}

	if err := r.scanner.Err(); err != nil {
		return nil, err
	}

	// Handle data accumulated at EOF without trailing blank line.
	if len(dataLines) > 0 {
		return &SSEEvent{Data: strings.Join(dataLines, "\n")}, nil
	}

	return nil, io.EOF
}
