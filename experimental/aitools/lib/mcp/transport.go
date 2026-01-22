package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

// StdioTransport implements MCP over stdio using line-delimited JSON.
type StdioTransport struct {
	reader *bufio.Reader
	writer io.Writer
	mu     sync.Mutex
}

// NewStdioTransport creates a new stdio transport.
func NewStdioTransport() *StdioTransport {
	return &StdioTransport{
		reader: bufio.NewReader(os.Stdin),
		writer: os.Stdout,
	}
}

// Read reads a JSON-RPC message from stdin.
func (t *StdioTransport) Read(ctx context.Context) (*JSONRPCRequest, error) {
	line, err := t.reader.ReadBytes('\n')
	if err != nil {
		if err == io.EOF {
			return nil, io.EOF
		}
		return nil, fmt.Errorf("failed to read from stdin: %w", err)
	}

	var req JSONRPCRequest
	if err := json.Unmarshal(line, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request: %w", err)
	}

	return &req, nil
}

// Write writes a JSON-RPC response to stdout.
func (t *StdioTransport) Write(ctx context.Context, resp *JSONRPCResponse) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	if _, err := t.writer.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write to stdout: %w", err)
	}

	return nil
}
