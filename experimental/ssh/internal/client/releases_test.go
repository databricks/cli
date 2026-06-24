package client

import (
	"errors"
	"fmt"
	"net/http"
	"syscall"
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/http2"
)

func TestIsProxyUploadError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "413 request entity too large",
			err:  &apierr.APIError{StatusCode: http.StatusRequestEntityTooLarge, Message: "request too large"},
			want: true,
		},
		{
			name: "connection reset mid-body",
			err:  fmt.Errorf(`Post "https://example/...": write tcp: %w`, syscall.ECONNRESET),
			want: true,
		},
		{
			name: "typed http2.StreamError wrapped",
			err:  fmt.Errorf(`Post "https://example/api/2.0/workspace-files/import-file/...": %w`, http2.StreamError{StreamID: 15, Code: http2.ErrCodeNo}),
			want: true,
		},
		{
			name: "stringified stream error",
			err:  errors.New("stream error: stream ID 15; NO_ERROR; received from peer"),
			want: true,
		},
		{
			name: "non-413 API error",
			err:  &apierr.APIError{StatusCode: http.StatusForbidden, Message: "permission denied"},
			want: false,
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
			assert.Equal(t, tt.want, isProxyUploadError(tt.err))
		})
	}
}

func TestNewHTTP11TransportDisablesHTTP2(t *testing.T) {
	tr := newHTTP11Transport(&config.Config{})
	assert.False(t, tr.ForceAttemptHTTP2)
	assert.NotNil(t, tr.TLSNextProto)
	assert.Empty(t, tr.TLSNextProto)
}
