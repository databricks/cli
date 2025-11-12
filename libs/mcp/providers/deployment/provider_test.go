package deployment

import (
	"strings"
	"testing"
)

func TestFormatDeployResult(t *testing.T) {
	tests := []struct {
		name     string
		result   *DeployResult
		contains []string
	}{
		{
			name: "successful deployment",
			result: &DeployResult{
				Success: true,
				Message: "Deployment completed successfully",
				AppURL:  "https://example.com/app",
				AppName: "test-app",
			},
			contains: []string{
				"Successfully deployed app 'test-app'",
				"https://example.com/app",
				"Deployment completed successfully",
			},
		},
		{
			name: "failed deployment",
			result: &DeployResult{
				Success: false,
				Message: "Build failed",
				AppName: "test-app",
			},
			contains: []string{
				"Deployment failed for app 'test-app'",
				"Build failed",
			},
		},
		{
			name: "successful deployment without URL",
			result: &DeployResult{
				Success: true,
				Message: "App deployed",
				AppURL:  "",
				AppName: "my-app",
			},
			contains: []string{
				"Successfully deployed app 'my-app'",
				"App deployed",
			},
		},
		{
			name: "failed deployment with detailed error",
			result: &DeployResult{
				Success: false,
				Message: "Failed to sync workspace: permission denied",
				AppName: "restricted-app",
			},
			contains: []string{
				"Deployment failed for app 'restricted-app'",
				"Failed to sync workspace: permission denied",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := formatDeployResult(tt.result)

			for _, substr := range tt.contains {
				if !strings.Contains(output, substr) {
					t.Errorf("expected output to contain %q\ngot: %s", substr, output)
				}
			}
		})
	}
}

func TestDeployResultStructure(t *testing.T) {
	result := &DeployResult{
		Success: true,
		Message: "Test message",
		AppURL:  "https://test.com",
		AppName: "test",
	}

	if !result.Success {
		t.Error("expected Success to be true")
	}

	if result.Message != "Test message" {
		t.Errorf("expected Message 'Test message', got %q", result.Message)
	}

	if result.AppURL != "https://test.com" {
		t.Errorf("expected AppURL 'https://test.com', got %q", result.AppURL)
	}

	if result.AppName != "test" {
		t.Errorf("expected AppName 'test', got %q", result.AppName)
	}
}

func TestDeployDatabricksAppInputValidation(t *testing.T) {
	tests := []struct {
		name        string
		input       DeployDatabricksAppInput
		expectValid bool
	}{
		{
			name: "valid input",
			input: DeployDatabricksAppInput{
				WorkDir:     "/absolute/path/to/app",
				Name:        "my-app",
				Description: "Test app",
				Force:       false,
			},
			expectValid: true,
		},
		{
			name: "valid input with force",
			input: DeployDatabricksAppInput{
				WorkDir:     "/another/path",
				Name:        "app-123",
				Description: "Another test",
				Force:       true,
			},
			expectValid: true,
		},
		{
			name: "valid input with special characters in name",
			input: DeployDatabricksAppInput{
				WorkDir:     "/path/to/app",
				Name:        "my-app-123",
				Description: "App with numbers",
				Force:       false,
			},
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.input.WorkDir == "" {
				t.Error("WorkDir should not be empty")
			}
			if tt.input.Name == "" {
				t.Error("Name should not be empty")
			}
			if tt.input.Description == "" {
				t.Error("Description should not be empty")
			}
		})
	}
}
