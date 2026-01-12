package common

import (
	"strings"
	"testing"
)

func TestFormatBrandedHeader(t *testing.T) {
	result := FormatBrandedHeader("🚀", "Test message")

	// Check for key components
	if !strings.Contains(result, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━") {
		t.Error("Missing header line")
	}
	if !strings.Contains(result, "🚀 Databricks AI Tools MCP server: Test message") {
		t.Error("Missing branded message")
	}
}

func TestFormatScaffoldSuccess(t *testing.T) {
	result := FormatScaffoldSuccess("appkit", "/path/to/app", 42)

	// Check for key components
	if !strings.Contains(result, "🚀 Databricks AI Tools MCP server") {
		t.Error("Missing branded header")
	}
	if !strings.Contains(result, "✅") {
		t.Error("Missing success checkmark")
	}
	if !strings.Contains(result, "appkit") {
		t.Error("Missing template name")
	}
	if !strings.Contains(result, "/path/to/app") {
		t.Error("Missing work directory")
	}
	if !strings.Contains(result, "42") {
		t.Error("Missing file count")
	}
}

func TestFormatValidationSuccess(t *testing.T) {
	result := FormatValidationSuccess("All checks passed")

	if !strings.Contains(result, "🔍 Databricks AI Tools MCP server") {
		t.Error("Missing branded header")
	}
	if !strings.Contains(result, "✅") {
		t.Error("Missing success checkmark")
	}
	if !strings.Contains(result, "All checks passed") {
		t.Error("Missing success message")
	}
}

func TestFormatValidationFailure(t *testing.T) {
	result := FormatValidationFailure("Build failed", 1, "stdout output", "stderr output")

	if !strings.Contains(result, "🔍 Databricks AI Tools MCP server") {
		t.Error("Missing branded header")
	}
	if !strings.Contains(result, "❌") {
		t.Error("Missing failure mark")
	}
	if !strings.Contains(result, "Build failed") {
		t.Error("Missing failure message")
	}
	if !strings.Contains(result, "Exit code: 1") {
		t.Error("Missing exit code")
	}
	if !strings.Contains(result, "stdout output") {
		t.Error("Missing stdout")
	}
	if !strings.Contains(result, "stderr output") {
		t.Error("Missing stderr")
	}
}

func TestFormatDeploymentSuccess(t *testing.T) {
	result := FormatDeploymentSuccess("my-app", "https://example.com/app")

	if !strings.Contains(result, "🚢 Databricks AI Tools MCP server") {
		t.Error("Missing branded header")
	}
	if !strings.Contains(result, "✅") {
		t.Error("Missing success checkmark")
	}
	if !strings.Contains(result, "my-app") {
		t.Error("Missing app name")
	}
	if !strings.Contains(result, "https://example.com/app") {
		t.Error("Missing app URL")
	}
}

func TestFormatDeploymentFailure(t *testing.T) {
	result := FormatDeploymentFailure("my-app", "Connection timeout")

	if !strings.Contains(result, "🚢 Databricks AI Tools MCP server") {
		t.Error("Missing branded header")
	}
	if !strings.Contains(result, "❌") {
		t.Error("Missing failure mark")
	}
	if !strings.Contains(result, "my-app") {
		t.Error("Missing app name")
	}
	if !strings.Contains(result, "Connection timeout") {
		t.Error("Missing error message")
	}
}
