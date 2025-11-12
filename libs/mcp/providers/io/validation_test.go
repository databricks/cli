package io

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/databricks/cli/libs/mcp/sandbox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))
}

type mockSandbox struct {
	execFunc func(ctx context.Context, command string) (*sandbox.ExecResult, error)
}

func (m *mockSandbox) Exec(ctx context.Context, command string) (*sandbox.ExecResult, error) {
	if m.execFunc != nil {
		return m.execFunc(ctx, command)
	}
	return &sandbox.ExecResult{ExitCode: 0, Stdout: "", Stderr: ""}, nil
}

func (m *mockSandbox) WriteFile(ctx context.Context, path, content string) error {
	return nil
}

func (m *mockSandbox) WriteFiles(ctx context.Context, files map[string]string) error {
	return nil
}

func (m *mockSandbox) ReadFile(ctx context.Context, path string) (string, error) {
	return "", nil
}

func (m *mockSandbox) DeleteFile(ctx context.Context, path string) error {
	return nil
}

func (m *mockSandbox) ListDirectory(ctx context.Context, path string) ([]string, error) {
	return nil, nil
}

func (m *mockSandbox) SetWorkdir(ctx context.Context, path string) error {
	return nil
}

func (m *mockSandbox) RefreshFromHost(ctx context.Context, hostPath, containerPath string) error {
	return nil
}

func (m *mockSandbox) ExportDirectory(ctx context.Context, containerPath, hostPath string) (string, error) {
	return "", nil
}

func (m *mockSandbox) Close() error {
	return nil
}

func TestValidationTRPC_Success(t *testing.T) {
	commandsExecuted := []string{}

	mock := &mockSandbox{
		execFunc: func(ctx context.Context, command string) (*sandbox.ExecResult, error) {
			commandsExecuted = append(commandsExecuted, command)
			return &sandbox.ExecResult{
				ExitCode: 0,
				Stdout:   fmt.Sprintf("Success: %s", command),
				Stderr:   "",
			}, nil
		},
	}

	validation := NewValidationTRPC()
	result, err := validation.Validate(context.Background(), mock, testLogger())

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Contains(t, result.Message, "All validation checks passed")
	assert.Nil(t, result.Details)

	expectedCommands := []string{
		"npm run build",
		"cd client && npx tsc --noEmit",
		"npm test",
	}
	assert.Equal(t, expectedCommands, commandsExecuted)

	require.NotEmpty(t, result.ProgressLog)
	assert.Contains(t, result.ProgressLog[0], "Starting validation")
	assert.Contains(t, result.ProgressLog[len(result.ProgressLog)-1], "All checks passed")
}

func TestValidationTRPC_BuildFailure(t *testing.T) {
	mock := &mockSandbox{
		execFunc: func(ctx context.Context, command string) (*sandbox.ExecResult, error) {
			if command == "npm run build" {
				return &sandbox.ExecResult{
					ExitCode: 1,
					Stdout:   "build output",
					Stderr:   "build error: missing dependency",
				}, nil
			}
			return &sandbox.ExecResult{ExitCode: 0, Stdout: "", Stderr: ""}, nil
		},
	}

	validation := NewValidationTRPC()
	result, err := validation.Validate(context.Background(), mock, testLogger())

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Equal(t, "Build failed", result.Message)
	require.NotNil(t, result.Details)
	assert.Equal(t, 1, result.Details.ExitCode)
	assert.Equal(t, "build output", result.Details.Stdout)
	assert.Contains(t, result.Details.Stderr, "build error")

	require.NotEmpty(t, result.ProgressLog)
	assert.Contains(t, result.ProgressLog[0], "Starting validation")
	assert.Contains(t, result.ProgressLog[len(result.ProgressLog)-1], "Build failed")
}

func TestValidationTRPC_TypeCheckFailure(t *testing.T) {
	mock := &mockSandbox{
		execFunc: func(ctx context.Context, command string) (*sandbox.ExecResult, error) {
			if command == "cd client && npx tsc --noEmit" {
				return &sandbox.ExecResult{
					ExitCode: 2,
					Stdout:   "type check output",
					Stderr:   "error TS2322: Type 'string' is not assignable to type 'number'",
				}, nil
			}
			return &sandbox.ExecResult{ExitCode: 0, Stdout: "", Stderr: ""}, nil
		},
	}

	validation := NewValidationTRPC()
	result, err := validation.Validate(context.Background(), mock, testLogger())

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Equal(t, "Type check failed", result.Message)
	require.NotNil(t, result.Details)
	assert.Equal(t, 2, result.Details.ExitCode)
	assert.Contains(t, result.Details.Stderr, "TS2322")
}

func TestValidationTRPC_TestFailure(t *testing.T) {
	mock := &mockSandbox{
		execFunc: func(ctx context.Context, command string) (*sandbox.ExecResult, error) {
			if command == "npm test" {
				return &sandbox.ExecResult{
					ExitCode: 1,
					Stdout:   "Test suite failed",
					Stderr:   "Expected 5 but got 3",
				}, nil
			}
			return &sandbox.ExecResult{ExitCode: 0, Stdout: "", Stderr: ""}, nil
		},
	}

	validation := NewValidationTRPC()
	result, err := validation.Validate(context.Background(), mock, testLogger())

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Equal(t, "Tests failed", result.Message)
	require.NotNil(t, result.Details)
	assert.Equal(t, 1, result.Details.ExitCode)
	assert.Contains(t, result.Details.Stdout, "Test suite failed")
}

func TestValidationTRPC_ExecError(t *testing.T) {
	mock := &mockSandbox{
		execFunc: func(ctx context.Context, command string) (*sandbox.ExecResult, error) {
			if command == "npm run build" {
				return nil, fmt.Errorf("sandbox connection lost")
			}
			return &sandbox.ExecResult{ExitCode: 0, Stdout: "", Stderr: ""}, nil
		},
	}

	validation := NewValidationTRPC()
	result, err := validation.Validate(context.Background(), mock, testLogger())

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Equal(t, "Build failed", result.Message)
	require.NotNil(t, result.Details)
	assert.Equal(t, -1, result.Details.ExitCode)
	assert.Contains(t, result.Details.Stderr, "Failed to run npm build")
}

func TestValidationTRPC_DockerImage(t *testing.T) {
	validation := NewValidationTRPC()
	assert.Equal(t, "node:20-alpine3.22", validation.DockerImage())
}

func TestValidationCmd_Success(t *testing.T) {
	mock := &mockSandbox{
		execFunc: func(ctx context.Context, command string) (*sandbox.ExecResult, error) {
			assert.Equal(t, "make test", command)
			return &sandbox.ExecResult{
				ExitCode: 0,
				Stdout:   "All tests passed",
				Stderr:   "",
			}, nil
		},
	}

	validation := NewValidationCmd("make test", "")
	result, err := validation.Validate(context.Background(), mock, testLogger())

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, "Custom validation passed", result.Message)
	assert.Nil(t, result.Details)
}

func TestValidationCmd_Failure(t *testing.T) {
	mock := &mockSandbox{
		execFunc: func(ctx context.Context, command string) (*sandbox.ExecResult, error) {
			return &sandbox.ExecResult{
				ExitCode: 1,
				Stdout:   "command output",
				Stderr:   "command failed",
			}, nil
		},
	}

	validation := NewValidationCmd("make test", "")
	result, err := validation.Validate(context.Background(), mock, testLogger())

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	assert.Equal(t, "Custom validation command failed", result.Message)
	require.NotNil(t, result.Details)
	assert.Equal(t, 1, result.Details.ExitCode)
	assert.Equal(t, "command output", result.Details.Stdout)
	assert.Equal(t, "command failed", result.Details.Stderr)
}

func TestValidationCmd_ExecError(t *testing.T) {
	mock := &mockSandbox{
		execFunc: func(ctx context.Context, command string) (*sandbox.ExecResult, error) {
			return nil, fmt.Errorf("command not found")
		},
	}

	validation := NewValidationCmd("invalid-command", "")
	result, err := validation.Validate(context.Background(), mock, testLogger())

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Success)
	require.NotNil(t, result.Details)
	assert.Equal(t, -1, result.Details.ExitCode)
	assert.Contains(t, result.Details.Stderr, "Failed to run validation command")
}

func TestValidationCmd_DockerImage(t *testing.T) {
	tests := []struct {
		name          string
		dockerImage   string
		expectedImage string
	}{
		{
			name:          "default image",
			dockerImage:   "",
			expectedImage: "node:20-alpine3.22",
		},
		{
			name:          "custom image",
			dockerImage:   "python:3.11-slim",
			expectedImage: "python:3.11-slim",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validation := NewValidationCmd("make test", tt.dockerImage)
			assert.Equal(t, tt.expectedImage, validation.DockerImage())
		})
	}
}

func TestValidateResult_String(t *testing.T) {
	tests := []struct {
		name     string
		result   *ValidateResult
		contains []string
	}{
		{
			name: "success result",
			result: &ValidateResult{
				Success: true,
				Message: "All checks passed",
			},
			contains: []string{"✓", "All checks passed"},
		},
		{
			name: "failure without details",
			result: &ValidateResult{
				Success: false,
				Message: "Validation failed",
			},
			contains: []string{"✗", "Validation failed"},
		},
		{
			name: "failure with details",
			result: &ValidateResult{
				Success: false,
				Message: "Build failed",
				Details: &ValidationDetail{
					ExitCode: 1,
					Stdout:   "build output",
					Stderr:   "build error",
				},
			},
			contains: []string{"✗", "Build failed", "Exit code: 1", "build output", "build error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.result.String()
			for _, expected := range tt.contains {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestValidationDetail_Error(t *testing.T) {
	details := &ValidationDetail{
		ExitCode: 2,
		Stdout:   "some output",
		Stderr:   "some error",
	}

	errMsg := details.Error()
	assert.Contains(t, errMsg, "validation failed")
	assert.Contains(t, errMsg, "exit code 2")
	assert.Contains(t, errMsg, "some output")
	assert.Contains(t, errMsg, "some error")
}
