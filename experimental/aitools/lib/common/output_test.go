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
