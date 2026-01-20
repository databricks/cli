// Package errors provides JSON-RPC 2.0 compliant error types for the MCP server.
package errors

import (
	"encoding/json"
	"fmt"
)

// JSON-RPC 2.0 error codes as defined in the specification.
const (
	// CodeParseError indicates invalid JSON was received by the server.
	CodeParseError = -32700
	// CodeInvalidRequest indicates the JSON sent is not a valid Request object.
	CodeInvalidRequest = -32600
	// CodeMethodNotFound indicates the method does not exist or is not available.
	CodeMethodNotFound = -32601
	// CodeInvalidParams indicates invalid method parameter(s).
	CodeInvalidParams = -32602
	// CodeInternalError indicates an internal JSON-RPC error.
	CodeInternalError = -32603
	// CodeServerError indicates a server-specific error.
	CodeServerError = -32000
)

// Application-specific error codes (extending JSON-RPC 2.0).
const (
	// Configuration errors (-32100 to -32109)
	CodeConfigInvalid = -32100
	CodeConfigMissing = -32101
	CodeConfigWorkDir = -32102

	// Databricks errors (-32120 to -32129)
	CodeDatabricksAuth      = -32120
	CodeDatabricksQuery     = -32121
	CodeDatabricksWarehouse = -32122
	CodeDatabricksConnect   = -32123

	// Validation errors (-32130 to -32139)
	CodeValidationFailed = -32130
	CodeValidationState  = -32131
	CodeValidationBuild  = -32132
	CodeValidationTest   = -32133

	// Deployment errors (-32140 to -32149)
	CodeDeploymentFailed = -32140
	CodeDeploymentState  = -32141

	// State machine errors (-32150 to -32159)
	CodeStateTransition = -32150
	CodeStateInvalid    = -32151
)

// Error represents a structured error following JSON-RPC 2.0 specification.
// It includes an error code, message, and optional details for debugging.
type Error struct {
	Code    int            `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"data,omitempty"`
}

// Error implements the error interface, formatting the error with its code and message.
func (e *Error) Error() string {
	if len(e.Details) > 0 {
		details, _ := json.Marshal(e.Details)
		return fmt.Sprintf("[%d] %s (details: %s)", e.Code, e.Message, string(details))
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// WithDetail adds a key-value pair to the error's details map and returns the error for chaining.
// This allows for adding contextual information to errors.
func (e *Error) WithDetail(key string, value any) *Error {
	if e.Details == nil {
		e.Details = make(map[string]any)
	}
	e.Details[key] = value
	return e
}

// InvalidParams creates a JSON-RPC 2.0 error indicating invalid method parameters.
// Use this when client-provided parameters fail validation.
func InvalidParams(message string) *Error {
	return &Error{
		Code:    CodeInvalidParams,
		Message: message,
	}
}

// InvalidRequest creates a JSON-RPC 2.0 error for invalid request objects.
// Use this when the JSON sent is not a valid Request object.
func InvalidRequest(message string) *Error {
	return &Error{
		Code:    CodeInvalidRequest,
		Message: message,
	}
}

// InternalError creates a JSON-RPC 2.0 error for internal server errors.
// Use this for unexpected errors that occur during request processing.
func InternalError(message string) *Error {
	return &Error{
		Code:    CodeInternalError,
		Message: message,
	}
}

// MethodNotFound creates a JSON-RPC 2.0 error indicating the requested method doesn't exist.
// The method parameter is included in the error message for debugging.
func MethodNotFound(method string) *Error {
	return &Error{
		Code:    CodeMethodNotFound,
		Message: "method not found: " + method,
	}
}

// ProviderNotAvailable creates a server error indicating a provider is not available.
// The provider name is included both in the message and as a detail for debugging.
func ProviderNotAvailable(provider string) *Error {
	err := &Error{
		Code:    CodeServerError,
		Message: "provider not available: " + provider,
	}
	return err.WithDetail("provider", provider)
}

// ParseError creates a JSON-RPC 2.0 error for JSON parsing failures.
// Use this when the server receives invalid JSON.
func ParseError(message string) *Error {
	return &Error{
		Code:    CodeParseError,
		Message: message,
	}
}

// WrapError converts a standard Go error into an Error.
// If the error is already an Error, it is returned as-is.
// Otherwise, it is wrapped as an InternalError with the original error message.
func WrapError(err error) *Error {
	if err == nil {
		return InternalError("unknown error")
	}
	if mcpErr, ok := err.(*Error); ok {
		return mcpErr
	}
	return InternalError(err.Error())
}

// WithSuggestion adds a helpful suggestion to an error.
// The suggestion appears in the error details under the "suggestion" key.
func WithSuggestion(err *Error, suggestion string) *Error {
	if err == nil {
		return nil
	}
	return err.WithDetail("suggestion", suggestion)
}

// NewWithCode creates a new error with a specific code and message.
func NewWithCode(code int, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// Configuration error helpers

// ConfigInvalid creates an error for invalid configuration.
func ConfigInvalid(message string) *Error {
	return NewWithCode(CodeConfigInvalid, message)
}

// ConfigMissing creates an error for missing configuration.
func ConfigMissing(param string) *Error {
	err := NewWithCode(CodeConfigMissing, "missing required configuration: "+param)
	return WithSuggestion(err, "Check your ~/.databricks/aitools/config.json or environment variables")
}

// ConfigWorkDir creates an error for workspace directory issues.
func ConfigWorkDir(message string) *Error {
	return NewWithCode(CodeConfigWorkDir, message)
}

// Databricks error helpers

// DatabricksAuth creates an error for authentication failures.
func DatabricksAuth(message string) *Error {
	err := NewWithCode(CodeDatabricksAuth, message)
	return WithSuggestion(err, "Check that DATABRICKS_HOST and DATABRICKS_TOKEN are set correctly")
}

// DatabricksWarehouse creates an error for warehouse access issues.
func DatabricksWarehouse(warehouseID string) *Error {
	err := NewWithCode(CodeDatabricksWarehouse, "warehouse not accessible: "+warehouseID)
	return WithSuggestion(err, "Verify the warehouse ID and ensure it is running")
}

// DatabricksConnect creates an error for connection failures.
func DatabricksConnect(message string) *Error {
	err := NewWithCode(CodeDatabricksConnect, message)
	return WithSuggestion(err, "Check network connectivity and DATABRICKS_HOST configuration")
}

// Validation error helpers

// ValidationFailed creates an error for validation failures.
func ValidationFailed(step, message string) *Error {
	err := NewWithCode(CodeValidationFailed, fmt.Sprintf("validation failed at %s: %s", step, message))
	return err.WithDetail("step", step)
}

// ValidationState creates an error for invalid project state during validation.
func ValidationState(currentState, message string) *Error {
	err := NewWithCode(CodeValidationState, message)
	return err.WithDetail("current_state", currentState)
}

// Deployment error helpers

// DeploymentFailed creates an error for deployment failures.
func DeploymentFailed(message string) *Error {
	return NewWithCode(CodeDeploymentFailed, message)
}

// DeploymentState creates an error for invalid project state during deployment.
func DeploymentState(currentState, message string) *Error {
	err := NewWithCode(CodeDeploymentState, message)
	err = err.WithDetail("current_state", currentState)
	return WithSuggestion(err, "Ensure the project is validated before deployment")
}

// State machine error helpers

// StateTransition creates an error for invalid state transitions.
func StateTransition(from, to, message string) *Error {
	err := NewWithCode(CodeStateTransition, fmt.Sprintf("invalid state transition %s -> %s: %s", from, to, message))
	return err.WithDetail("from_state", from).WithDetail("to_state", to)
}

// StateInvalid creates an error for invalid state values.
func StateInvalid(state string) *Error {
	return NewWithCode(CodeStateInvalid, "invalid state: "+state)
}
