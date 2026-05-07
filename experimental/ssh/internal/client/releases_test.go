package client

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/http2"
)

func TestIsStreamResetError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "http2 stream error type",
			err:  http2.StreamError{StreamID: 15, Code: http2.ErrCodeNo},
			want: true,
		},
		{
			name: "wrapped http2 stream error type",
			err:  fmt.Errorf("post failed: %w", http2.StreamError{StreamID: 15, Code: http2.ErrCodeNo}),
			want: true,
		},
		{
			name: "string match from peer reset (Go HTTP/2 client format)",
			err:  errors.New(`Post "https://example/api/2.0/workspace-files/import-file/...": stream error: stream ID 15; NO_ERROR; received from peer`),
			want: true,
		},
		{
			name: "unrelated error",
			err:  errors.New("connection refused"),
			want: false,
		},
		{
			name: "API error message",
			err:  errors.New("RESOURCE_DOES_NOT_EXIST: path does not exist"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isStreamResetError(tt.err))
		})
	}
}
