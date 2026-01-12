package errors

import (
	"errors"
	"testing"
)

func TestError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected string
	}{
		{
			name: "error without details",
			err: &Error{
				Code:    CodeInternalError,
				Message: "something went wrong",
			},
			expected: "[-32603] something went wrong",
		},
		{
			name: "error with details",
			err: &Error{
				Code:    CodeInvalidParams,
				Message: "invalid parameter",
				Details: map[string]any{
					"field": "username",
					"value": "invalid",
				},
			},
			expected: `[-32602] invalid parameter (details: {"field":"username","value":"invalid"})`,
		},
		{
			name: "error with empty details map",
			err: &Error{
				Code:    CodeInternalError,
				Message: "error",
				Details: map[string]any{},
			},
			expected: "[-32603] error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestError_WithDetail(t *testing.T) {
	t.Run("add detail to error without details", func(t *testing.T) {
		err := &Error{
			Code:    CodeInternalError,
			Message: "error",
		}

		result := err.WithDetail("key", "value")

		if result != err {
			t.Error("WithDetail should return the same error instance")
		}

		if len(err.Details) != 1 {
			t.Errorf("Details length = %d, want 1", len(err.Details))
		}

		if err.Details["key"] != "value" {
			t.Errorf("Details[key] = %v, want %q", err.Details["key"], "value")
		}
	})

	t.Run("add detail to error with existing details", func(t *testing.T) {
		err := &Error{
			Code:    CodeInternalError,
			Message: "error",
			Details: map[string]any{
				"existing": "value",
			},
		}

		err = err.WithDetail("new", "data")

		if len(err.Details) != 2 {
			t.Errorf("Details length = %d, want 2", len(err.Details))
		}

		if err.Details["existing"] != "value" {
			t.Error("existing detail should be preserved")
		}

		if err.Details["new"] != "data" {
			t.Error("new detail should be added")
		}
	})

	t.Run("chain multiple details", func(t *testing.T) {
		err := &Error{
			Code:    CodeInternalError,
			Message: "error",
		}

		err = err.WithDetail("key1", "value1").WithDetail("key2", "value2")

		if len(err.Details) != 2 {
			t.Errorf("Details length = %d, want 2", len(err.Details))
		}
	})
}

func TestInvalidParams(t *testing.T) {
	message := "missing required field"
	err := InvalidParams(message)

	if err.Code != CodeInvalidParams {
		t.Errorf("Code = %d, want %d", err.Code, CodeInvalidParams)
	}

	if err.Message != message {
		t.Errorf("Message = %q, want %q", err.Message, message)
	}

	if err.Details != nil {
		t.Error("Details should be nil by default")
	}
}

func TestInvalidRequest(t *testing.T) {
	message := "malformed request"
	err := InvalidRequest(message)

	if err.Code != CodeInvalidRequest {
		t.Errorf("Code = %d, want %d", err.Code, CodeInvalidRequest)
	}

	if err.Message != message {
		t.Errorf("Message = %q, want %q", err.Message, message)
	}
}

func TestInternalError(t *testing.T) {
	message := "internal server error"
	err := InternalError(message)

	if err.Code != CodeInternalError {
		t.Errorf("Code = %d, want %d", err.Code, CodeInternalError)
	}

	if err.Message != message {
		t.Errorf("Message = %q, want %q", err.Message, message)
	}
}

func TestMethodNotFound(t *testing.T) {
	method := "unknown_method"
	err := MethodNotFound(method)

	if err.Code != CodeMethodNotFound {
		t.Errorf("Code = %d, want %d", err.Code, CodeMethodNotFound)
	}

	expectedMessage := "method not found: unknown_method"
	if err.Message != expectedMessage {
		t.Errorf("Message = %q, want %q", err.Message, expectedMessage)
	}
}

func TestProviderNotAvailable(t *testing.T) {
	provider := "databricks"
	err := ProviderNotAvailable(provider)

	if err.Code != CodeServerError {
		t.Errorf("Code = %d, want %d", err.Code, CodeServerError)
	}

	expectedMessage := "provider not available: databricks"
	if err.Message != expectedMessage {
		t.Errorf("Message = %q, want %q", err.Message, expectedMessage)
	}

	if err.Details == nil {
		t.Fatal("Details should not be nil")
	}

	if err.Details["provider"] != provider {
		t.Errorf("Details[provider] = %v, want %q", err.Details["provider"], provider)
	}
}

func TestParseError(t *testing.T) {
	message := "failed to parse JSON"
	err := ParseError(message)

	if err.Code != CodeParseError {
		t.Errorf("Code = %d, want %d", err.Code, CodeParseError)
	}

	if err.Message != message {
		t.Errorf("Message = %q, want %q", err.Message, message)
	}
}

func TestWrapError(t *testing.T) {
	t.Run("wrap nil error", func(t *testing.T) {
		err := WrapError(nil)

		if err.Code != CodeInternalError {
			t.Errorf("Code = %d, want %d", err.Code, CodeInternalError)
		}

		if err.Message != "unknown error" {
			t.Errorf("Message = %q, want %q", err.Message, "unknown error")
		}
	})

	t.Run("wrap standard error", func(t *testing.T) {
		originalErr := errors.New("standard error")
		err := WrapError(originalErr)

		if err.Code != CodeInternalError {
			t.Errorf("Code = %d, want %d", err.Code, CodeInternalError)
		}

		if err.Message != "standard error" {
			t.Errorf("Message = %q, want %q", err.Message, "standard error")
		}
	})

	t.Run("wrap Error returns same error", func(t *testing.T) {
		originalErr := &Error{
			Code:    CodeInvalidParams,
			Message: "invalid params",
		}

		err := WrapError(originalErr)

		if err != originalErr {
			t.Error("WrapError should return the same Error instance")
		}

		if err.Code != CodeInvalidParams {
			t.Errorf("Code = %d, want %d", err.Code, CodeInvalidParams)
		}
	})
}

func TestErrorCodes(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected int
	}{
		{"ParseError", CodeParseError, -32700},
		{"InvalidRequest", CodeInvalidRequest, -32600},
		{"MethodNotFound", CodeMethodNotFound, -32601},
		{"InvalidParams", CodeInvalidParams, -32602},
		{"InternalError", CodeInternalError, -32603},
		{"ServerError", CodeServerError, -32000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code != tt.expected {
				t.Errorf("%s = %d, want %d", tt.name, tt.code, tt.expected)
			}
		})
	}
}
