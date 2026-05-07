package client

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsStreamResetError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "raw http2 stream error string",
			err:  errors.New("stream error: stream ID 15; NO_ERROR"),
			want: true,
		},
		{
			name: "wrapped peer-reset error (Go HTTP/2 client format)",
			err:  fmt.Errorf(`Post "https://example/api/2.0/workspace-files/import-file/...": %w`, errors.New("stream error: stream ID 15; NO_ERROR; received from peer")),
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
