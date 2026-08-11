package genie

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorCodeString(t *testing.T) {
	tests := []struct {
		name string
		err  SSEError
		want string
	}{
		{"error_code field", SSEError{ErrorCode: "RESOURCE_DOES_NOT_EXIST"}, "RESOURCE_DOES_NOT_EXIST"},
		{"code field", SSEError{Code: "INTERNAL"}, "INTERNAL"},
		{"error_code wins over code", SSEError{ErrorCode: "A", Code: "B"}, "A"},
		{"neither", SSEError{}, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.err.ErrorCodeString())
		})
	}
}

func TestOutputItemUIType(t *testing.T) {
	assert.Equal(t, "THOUGHT", OutputItem{Metadata: map[string]any{"ui_type": "THOUGHT"}}.UIType())
	assert.Empty(t, OutputItem{Metadata: map[string]any{"ui_type": 42}}.UIType())
	assert.Empty(t, OutputItem{}.UIType())
}
